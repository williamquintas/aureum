// Package auth provides JWT claims parsing and validation utilities.
package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the custom JWT claims structure including user roles and permissions.
type Claims struct {
	jwt.RegisteredClaims
	Email    string                 `json:"email"`
	Name     string                 `json:"name"`
	Roles    []string               `json:"roles"`
	TenantID string                 `json:"tenant_id"`
	Custom   map[string]interface{} `json:"custom"`
}

// ExtractClaims parses and validates a JWT token string, returning the embedded Claims.
func ExtractClaims(tokenString string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}
	return claims, nil
}

// HasRole checks if the claims contain a specific role.
func (c *Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if the claims grant a specific action on a resource.
func (c *Claims) HasPermission(resource, action string) bool {
	if c.Custom == nil {
		return false
	}
	perms, ok := c.Custom["permissions"].(map[string]interface{})
	if !ok {
		return false
	}
	actions, ok := perms[resource].([]interface{})
	if !ok {
		return false
	}
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}
