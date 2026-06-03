package auth

import (
	"context"
	"fmt"
	"time"

	gocloak "github.com/Nerzal/gocloak/v13"

	"github.com/aureum/identity-svc/internal/application"
	"github.com/aureum/identity-svc/internal/domain"
)

type Client struct {
	client   *gocloak.GoCloak
	realm    string
	clientID string
	secret   string
}

func NewKeycloakClient(baseURL, realm, clientID, secret string) *Client {
	return &Client{
		client:   gocloak.NewClient(baseURL),
		realm:    realm,
		clientID: clientID,
		secret:   secret,
	}
}

func (c *Client) getToken(ctx context.Context) (*gocloak.JWT, error) {
	token, err := c.client.LoginClient(ctx, c.clientID, c.secret, c.realm)
	if err != nil {
		return nil, fmt.Errorf("keycloak login client: %w", err)
	}
	return token, nil
}

func (c *Client) CreateUser(ctx context.Context, email, password, name string) (string, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return "", err
	}

	attrs := map[string][]string{
		"email_verified": {"true"},
	}
	user := gocloak.User{
		Email:      &email,
		Enabled:    boolPtr(true),
		FirstName:  &name,
		Attributes: &attrs,
	}

	keycloakID, err := c.client.CreateUser(ctx, token.AccessToken, c.realm, user)
	if err != nil {
		return "", fmt.Errorf("keycloak create user: %w", err)
	}

	err = c.client.SetPassword(ctx, token.AccessToken, keycloakID, c.realm, password, false)
	if err != nil {
		return "", fmt.Errorf("keycloak set password: %w", err)
	}

	return keycloakID, nil
}

func (c *Client) Authenticate(ctx context.Context, email, password string) (*application.LoginResponse, error) {
	token, err := c.client.Login(ctx, c.clientID, c.secret, c.realm, email, password)
	if err != nil {
		return nil, err
	}

	resp := &application.LoginResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		IDToken:      token.IDToken,
		ExpiresIn:    token.ExpiresIn,
		TokenType:    "Bearer",
	}

	if resp.ExpiresIn == 0 {
		resp.ExpiresIn = 900
	}

	return resp, nil
}

func (c *Client) VerifyEmail(ctx context.Context, userID string) error {
	token, err := c.getToken(ctx)
	if err != nil {
		return err
	}

	user, err := c.client.GetUserByID(ctx, token.AccessToken, c.realm, userID)
	if err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	attrs := map[string][]string{
		"email_verified":    {"true"},
		"email_verified_at": {now},
	}
	user.Attributes = &attrs
	user.EmailVerified = boolPtr(true)

	err = c.client.UpdateUser(ctx, token.AccessToken, c.realm, *user)
	if err != nil {
		return fmt.Errorf("keycloak update user: %w", err)
	}

	return nil
}

func (c *Client) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}

	users, err := c.client.GetUsers(ctx, token.AccessToken, c.realm, gocloak.GetUsersParams{
		Email: &email,
		Exact: boolPtr(true),
	})
	if err != nil {
		return nil, fmt.Errorf("keycloak get users: %w", err)
	}

	if len(users) == 0 {
		return nil, domain.ErrUserNotFound
	}

	kcUser := users[0]
	user := &domain.User{
		KeycloakID:    *kcUser.ID,
		Email:         *kcUser.Email,
		EmailVerified: kcUser.EmailVerified != nil && *kcUser.EmailVerified,
		Status:        domain.UserStatusActive,
	}

	if kcUser.FirstName != nil {
		user.Name = *kcUser.FirstName
	}

	return user, nil
}

func (c *Client) ValidateToken(ctx context.Context, accessToken string) (*domain.User, error) {
	rpt, err := c.client.RetrospectToken(ctx, accessToken, c.clientID, c.secret, c.realm)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}
	if !*rpt.Active {
		return nil, domain.ErrTokenExpired
	}

	_, claims, err := c.client.DecodeAccessToken(ctx, accessToken, c.realm)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}

	user := &domain.User{}

	if sub, ok := (*claims)["sub"].(string); ok {
		user.ID = sub
	}
	if email, ok := (*claims)["email"].(string); ok {
		user.Email = email
	}
	if name, ok := (*claims)["preferred_username"].(string); ok {
		user.Name = name
	}

	return user, nil
}

func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*application.LoginResponse, error) {
	token, err := c.client.RefreshToken(ctx, refreshToken, c.clientID, c.secret, c.realm)
	if err != nil {
		return nil, fmt.Errorf("keycloak refresh token: %w", err)
	}

	resp := &application.LoginResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		IDToken:      token.IDToken,
		ExpiresIn:    token.ExpiresIn,
		TokenType:    "Bearer",
	}

	if resp.ExpiresIn == 0 {
		resp.ExpiresIn = 900
	}

	return resp, nil
}

func (c *Client) Logout(ctx context.Context, refreshToken string) error {
	token, err := c.getToken(ctx)
	if err != nil {
		return err
	}

	err = c.client.Logout(ctx, c.clientID, c.secret, c.realm, refreshToken)
	if err != nil {
		return fmt.Errorf("keycloak logout: %w", err)
	}

	_ = c.client.LogoutAllSessions(ctx, token.AccessToken, c.realm, "")
	return nil
}

func (c *Client) UpdatePassword(ctx context.Context, userID, newPassword string) error {
	token, err := c.getToken(ctx)
	if err != nil {
		return err
	}

	err = c.client.SetPassword(ctx, token.AccessToken, userID, c.realm, newPassword, false)
	if err != nil {
		return fmt.Errorf("keycloak set password: %w", err)
	}

	return nil
}

func (c *Client) GetUserSessions(ctx context.Context, userID string) ([]application.UserSessionRepresentation, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}

	kcSessions, err := c.client.GetUserSessions(ctx, token.AccessToken, c.realm, userID)
	if err != nil {
		return nil, fmt.Errorf("keycloak get user sessions: %w", err)
	}

	sessions := make([]application.UserSessionRepresentation, 0, len(kcSessions))
	for _, ks := range kcSessions {
		s := application.UserSessionRepresentation{
			ID:        safeDeref(ks.ID),
			UserID:    safeDeref(ks.UserID),
			IPAddress: safeDeref(ks.IPAddress),
		}
		if ks.Start != nil {
			s.Start = time.UnixMilli(*ks.Start)
		}
		if ks.LastAccess != nil {
			s.LastAccess = time.UnixMilli(*ks.LastAccess)
		}
		sessions = append(sessions, s)
	}

	return sessions, nil
}

func (c *Client) LogoutUserSession(ctx context.Context, sessionID string) error {
	token, err := c.getToken(ctx)
	if err != nil {
		return err
	}

	return c.client.LogoutUserSession(ctx, token.AccessToken, c.realm, sessionID)
}

func boolPtr(b bool) *bool {
	return &b
}

func safeDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
