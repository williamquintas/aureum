package domain

import (
	"fmt"
	"time"
)

type CardBrand string

const (
	CardBrandVisa       CardBrand = "visa"
	CardBrandMastercard CardBrand = "mastercard"
	CardBrandAmex       CardBrand = "amex"
	CardBrandElo        CardBrand = "elo"
	CardBrandHipercard  CardBrand = "hipercard"
	CardBrandDiners     CardBrand = "diners"
	CardBrandOther      CardBrand = "other"
)

func ValidCardBrands() []CardBrand {
	return []CardBrand{CardBrandVisa, CardBrandMastercard, CardBrandAmex, CardBrandElo, CardBrandHipercard, CardBrandDiners, CardBrandOther}
}

func (b CardBrand) Valid() bool {
	for _, valid := range ValidCardBrands() {
		if b == valid {
			return true
		}
	}
	return false
}

type CardType string

const (
	CardTypeCredit   CardType = "credit"
	CardTypeDebit    CardType = "debit"
	CardTypeMultiple CardType = "multiple"
)

func ValidCardTypes() []CardType {
	return []CardType{CardTypeCredit, CardTypeDebit, CardTypeMultiple}
}

func (t CardType) Valid() bool {
	for _, valid := range ValidCardTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

type CreditCard struct {
	ID              string
	UserID          string
	Name            string
	Brand           CardBrand
	CardType        CardType
	LastFourDigits  string
	ClosingDay      int
	DueDay          int
	CreditLimit     int64
	AvailableCredit int64
	Active          bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

type CreateCreditCardInput struct {
	UserID         string
	Name           string
	Brand          CardBrand
	CardType       CardType
	LastFourDigits string
	ClosingDay     int
	DueDay         int
	CreditLimit    int64
	IdempotencyKey string
}

type UpdateCreditCardInput struct {
	ID             string
	UserID         string
	Name           *string
	ClosingDay     *int
	DueDay         *int
	CreditLimit    *int64
	Active         *bool
	IdempotencyKey string
}

func NewCreditCard(input CreateCreditCardInput) (*CreditCard, error) {
	if input.UserID == "" {
		return nil, ErrMissingField
	}
	if input.Name == "" {
		return nil, ErrMissingField
	}
	if input.Brand == "" {
		return nil, ErrMissingField
	}
	if !input.Brand.Valid() {
		return nil, ErrInvalidCardBrand
	}
	if input.CardType == "" {
		return nil, ErrMissingField
	}
	if !input.CardType.Valid() {
		return nil, ErrInvalidCardType
	}
	if input.LastFourDigits == "" {
		return nil, ErrMissingField
	}
	if input.ClosingDay < 1 || input.ClosingDay > 31 {
		return nil, ErrInvalidDay
	}
	if input.DueDay < 1 || input.DueDay > 31 {
		return nil, ErrInvalidDay
	}
	if input.CreditLimit < 0 {
		return nil, fmt.Errorf("credit limit cannot be negative: %w", ErrNegativeAmount)
	}

	now := time.Now()
	return &CreditCard{
		UserID:          input.UserID,
		Name:            input.Name,
		Brand:           input.Brand,
		CardType:        input.CardType,
		LastFourDigits:  input.LastFourDigits,
		ClosingDay:      input.ClosingDay,
		DueDay:          input.DueDay,
		CreditLimit:     input.CreditLimit,
		AvailableCredit: input.CreditLimit,
		Active:          true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func (c *CreditCard) ApplyUpdate(input UpdateCreditCardInput) error {
	if input.UserID != "" && input.UserID != c.UserID {
		return ErrAccessDenied
	}
	if input.Name != nil {
		if *input.Name == "" {
			return ErrMissingField
		}
		c.Name = *input.Name
	}
	if input.ClosingDay != nil {
		if *input.ClosingDay < 1 || *input.ClosingDay > 31 {
			return ErrInvalidDay
		}
		c.ClosingDay = *input.ClosingDay
	}
	if input.DueDay != nil {
		if *input.DueDay < 1 || *input.DueDay > 31 {
			return ErrInvalidDay
		}
		c.DueDay = *input.DueDay
	}
	if input.CreditLimit != nil {
		if *input.CreditLimit < 0 {
			return fmt.Errorf("credit limit cannot be negative: %w", ErrNegativeAmount)
		}
		diff := *input.CreditLimit - c.CreditLimit
		c.CreditLimit = *input.CreditLimit
		c.AvailableCredit += diff
	}
	if input.Active != nil {
		c.Active = *input.Active
	}
	c.UpdatedAt = time.Now()
	return nil
}
