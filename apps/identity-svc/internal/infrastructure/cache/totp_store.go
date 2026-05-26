package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const totpPrefix = "totp:setup:"

type TOTPStore struct {
	rdb redis.UniversalClient
}

type TOTPData struct {
	Secret    string `json:"secret"`
	QRCodeURL string `json:"qr_code_url"`
	UserID    string `json:"user_id"`
	ExpiresAt int64  `json:"expires_at"`
}

func NewTOTPStore(rdb redis.UniversalClient) *TOTPStore {
	return &TOTPStore{rdb: rdb}
}

func (s *TOTPStore) Save(ctx context.Context, userID string, data interface{}, ttl time.Duration) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, totpPrefix+userID, b, ttl).Err()
}

func (s *TOTPStore) GetAndDelete(ctx context.Context, userID string) (interface{}, error) {
	b, err := s.rdb.GetDel(ctx, totpPrefix+userID).Bytes()
	if err != nil {
		return nil, err
	}
	var data TOTPData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
