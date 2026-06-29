// Package domain contains domain entities, value objects, and repository interfaces for credit card management.
package domain

import (
	"fmt"
	"time"
)

// CardBrand represents a credit card brand.
type CardBrand string

const (
	// CardBrandVisa is the Visa brand.
	CardBrandVisa CardBrand = "visa"
	// CardBrandMastercard is the Mastercard brand.
	CardBrandMastercard CardBrand = "mastercard"
	// CardBrandAmex is the American Express brand.
	CardBrandAmex CardBrand = "amex"
	// CardBrandElo is the Elo brand.
	CardBrandElo CardBrand = "elo"
	// CardBrandHipercard is the Hipercard brand.
	CardBrandHipercard CardBrand = "hipercard"
	// CardBrandDiners is the Diners Club brand.
	CardBrandDiners CardBrand = "diners"
	// CardBrandOther represents other/unknown card brands.
	CardBrandOther CardBrand = "other"
)

// ValidCardBrands returns all valid card brands.
func ValidCardBrands() []CardBrand {
	return []CardBrand{
		CardBrandVisa, CardBrandMastercard, CardBrandAmex,
		CardBrandElo, CardBrandHipercard, CardBrandDiners, CardBrandOther,
	}
}

// Valid checks if the card brand is a recognized value.
func (b CardBrand) Valid() bool {
	for _, valid := range ValidCardBrands() {
		if b == valid {
			return true
		}
	}
	return false
}

// CardType represents the type of a credit card.
type CardType string

const (
	// CardTypeCredit is a credit card.
	CardTypeCredit CardType = "credit"
	// CardTypeDebit is a debit card.
	CardTypeDebit CardType = "debit"
	// CardTypeMultiple is a multiple (credit + debit) card.
	CardTypeMultiple CardType = "multiple"
)

// ValidCardTypes returns all valid card types.
func ValidCardTypes() []CardType {
	return []CardType{CardTypeCredit, CardTypeDebit, CardTypeMultiple}
}

// Valid checks if the card type is a recognized value.
func (t CardType) Valid() bool {
	for _, valid := range ValidCardTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

// CreditCard represents a credit card entity.
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

// CreateCreditCardInput contains validated input for creating a new credit card.
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

// UpdateCreditCardInput contains optional fields for updating a credit card.
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

// NewCreditCard creates a new CreditCard with validation.
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

// ApplyUpdate applies partial updates to a credit card.
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
