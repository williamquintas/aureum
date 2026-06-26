package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const emailOTPPrefix = "otp:verify:"

type EmailOTPStore struct {
	rdb redis.UniversalClient
}

type EmailOTPData struct {
	Email     string `json:"email"`
	OTP       string `json:"otp"`
	ExpiresAt int64  `json:"expires_at"`
}

func NewEmailOTPStore(rdb redis.UniversalClient) *EmailOTPStore {
	return &EmailOTPStore{rdb: rdb}
}

func (s *EmailOTPStore) Save(ctx context.Context, email, otp string, ttl time.Duration) error {
	data := EmailOTPData{
		Email:     email,
		OTP:       otp,
		ExpiresAt: time.Now().Add(ttl).Unix(),
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, emailOTPPrefix+email, b, ttl).Err()
}

func (s *EmailOTPStore) GetAndDelete(ctx context.Context, email string) (string, error) {
	b, err := s.rdb.GetDel(ctx, emailOTPPrefix+email).Bytes()
	if err != nil {
		return "", err
	}
	var data EmailOTPData
	if err := json.Unmarshal(b, &data); err != nil {
		return "", err
	}
	return data.OTP, nil
}
