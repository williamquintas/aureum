package domain

import (
	"fmt"
	"time"
)

// AssetType represents the type of investment asset.
type AssetType string

const (
	AssetTypeStock          AssetType = "stock"
	AssetTypeETF            AssetType = "etf"
	AssetTypeRealEstateFund AssetType = "real_estate_fund"
	AssetTypeTreasury       AssetType = "treasury"
	AssetTypeCDB            AssetType = "cdb"
	AssetTypeLCI            AssetType = "lci"
	AssetTypeLCA            AssetType = "lca"
	AssetTypeCrypto         AssetType = "crypto"
	AssetTypePension        AssetType = "pension"
	AssetTypeFund           AssetType = "fund"
	AssetTypeDollar         AssetType = "dollar"
	AssetTypeGold           AssetType = "gold"
	AssetTypeOther          AssetType = "other"
)

func ValidAssetTypes() []AssetType {
	return []AssetType{
		AssetTypeStock, AssetTypeETF, AssetTypeRealEstateFund,
		AssetTypeTreasury, AssetTypeCDB, AssetTypeLCI, AssetTypeLCA,
		AssetTypeCrypto, AssetTypePension, AssetTypeFund,
		AssetTypeDollar, AssetTypeGold, AssetTypeOther,
	}
}

func (a AssetType) Valid() bool {
	for _, v := range ValidAssetTypes() {
		if a == v {
			return true
		}
	}
	return false
}

// InvestmentStatus represents the lifecycle status of an investment.
type InvestmentStatus string

const (
	StatusActive    InvestmentStatus = "active"
	StatusSold      InvestmentStatus = "sold"
	StatusCancelled InvestmentStatus = "cancelled"
)

func (s InvestmentStatus) Valid() bool {
	switch s {
	case StatusActive, StatusSold, StatusCancelled:
		return true
	}
	return false
}

// Investment is the core domain entity.
type Investment struct {
	ID            string
	UserID        string
	Name          string
	Ticker        string
	AssetType     AssetType
	Quantity      int64
	AveragePrice  int64 // cents per unit
	TotalInvested int64 // cents
	Status        InvestmentStatus
	Broker        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

// CreateInvestmentInput is the input for creating a new Investment.
type CreateInvestmentInput struct {
	UserID         string
	Name           string
	Ticker         string
	AssetType      AssetType
	Quantity       int64
	AveragePrice   int64 // cents per unit
	Broker         string
	Status         InvestmentStatus
	IdempotencyKey string
}

// UpdateInvestmentInput is the input for updating an existing Investment.
type UpdateInvestmentInput struct {
	ID             string
	UserID         string
	Name           *string
	Ticker         *string
	AssetType      *AssetType
	Quantity       *int64
	AveragePrice   *int64
	Broker         *string
	Status         *InvestmentStatus
	IdempotencyKey string
}

// NewInvestment creates a new Investment entity with validation.
func NewInvestment(input CreateInvestmentInput) (*Investment, error) {
	if input.UserID == "" {
		return nil, fmt.Errorf("user_id: %w", ErrMissingField)
	}
	if input.Name == "" {
		return nil, fmt.Errorf("name: %w", ErrMissingField)
	}
	if input.Ticker == "" {
		return nil, fmt.Errorf("ticker: %w", ErrMissingField)
	}
	if input.AssetType == "" {
		return nil, fmt.Errorf("asset_type: %w", ErrMissingField)
	}
	if !input.AssetType.Valid() {
		return nil, fmt.Errorf("asset_type %q: %w", input.AssetType, ErrInvalidAssetType)
	}
	if input.Quantity <= 0 {
		return nil, fmt.Errorf("quantity %d: %w", input.Quantity, ErrInvalidQuantity)
	}
	if input.AveragePrice < 0 {
		return nil, fmt.Errorf("average_price %d: %w", input.AveragePrice, ErrInvalidPrice)
	}
	if input.Status == "" {
		return nil, fmt.Errorf("status: %w", ErrMissingField)
	}
	if !input.Status.Valid() {
		return nil, fmt.Errorf("status %q: %w", input.Status, ErrInvalidStatus)
	}

	now := time.Now()
	return &Investment{
		UserID:        input.UserID,
		Name:          input.Name,
		Ticker:        input.Ticker,
		AssetType:     input.AssetType,
		Quantity:      input.Quantity,
		AveragePrice:  input.AveragePrice,
		TotalInvested: input.Quantity * input.AveragePrice,
		Status:        input.Status,
		Broker:        input.Broker,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// Sell reduces quantity and recalculates average price.
// For a partial sell, the average_price remains the same,
// but total_invested is reduced proportionally.
func (inv *Investment) Sell(sellQuantity int64, sellPrice int64) error {
	if sellQuantity <= 0 {
		return fmt.Errorf("sell_quantity %d: %w", sellQuantity, ErrInvalidQuantity)
	}
	if sellPrice < 0 {
		return fmt.Errorf("sell_price %d: %w", sellPrice, ErrInvalidPrice)
	}
	if sellQuantity > inv.Quantity {
		return fmt.Errorf("sell %d of %d: %w", sellQuantity, inv.Quantity, ErrInsufficientQuantity)
	}

	// Proportionally reduce total_invested.
	reduction := (inv.TotalInvested * sellQuantity) / inv.Quantity
	inv.TotalInvested -= reduction
	inv.Quantity -= sellQuantity
	inv.UpdatedAt = time.Now()

	if inv.Quantity == 0 {
		inv.Status = StatusSold
	}
	return nil
}

// UpdateAveragePrice adjusts average price after a new buy.
// Calculates weighted average: new_avg = (current_total + buy_amount) / (current_qty + buy_qty).
func (inv *Investment) UpdateAveragePrice(buyQuantity int64, buyPrice int64) {
	totalCost := inv.TotalInvested + (buyQuantity * buyPrice)
	newQty := inv.Quantity + buyQuantity
	inv.AveragePrice = totalCost / newQty
	inv.Quantity = newQty
	inv.TotalInvested = totalCost
	inv.UpdatedAt = time.Now()
}

// Cancel marks the investment as cancelled.
func (inv *Investment) Cancel() error {
	if inv.Status != StatusActive {
		return fmt.Errorf("cannot cancel investment with status %q: %w", inv.Status, ErrStatusTransition)
	}
	inv.Status = StatusCancelled
	inv.UpdatedAt = time.Now()
	return nil
}

// ApplyUpdate applies partial updates to the investment.
func (inv *Investment) ApplyUpdate(input UpdateInvestmentInput) error {
	if input.UserID != "" && input.UserID != inv.UserID {
		return ErrAccessDenied
	}
	if input.Name != nil {
		if *input.Name == "" {
			return fmt.Errorf("name: %w", ErrMissingField)
		}
		inv.Name = *input.Name
	}
	if input.Ticker != nil {
		if *input.Ticker == "" {
			return fmt.Errorf("ticker: %w", ErrMissingField)
		}
		inv.Ticker = *input.Ticker
	}
	if input.AssetType != nil {
		if !input.AssetType.Valid() {
			return fmt.Errorf("asset_type %q: %w", *input.AssetType, ErrInvalidAssetType)
		}
		inv.AssetType = *input.AssetType
	}
	if input.Quantity != nil {
		if *input.Quantity <= 0 {
			return fmt.Errorf("quantity %d: %w", *input.Quantity, ErrInvalidQuantity)
		}
		inv.Quantity = *input.Quantity
	}
	if input.AveragePrice != nil {
		if *input.AveragePrice < 0 {
			return fmt.Errorf("average_price %d: %w", *input.AveragePrice, ErrInvalidPrice)
		}
		inv.AveragePrice = *input.AveragePrice
	}
	if input.Broker != nil {
		inv.Broker = *input.Broker
	}
	if input.Status != nil {
		if err := inv.TransitionStatus(*input.Status); err != nil {
			return err
		}
	}
	inv.UpdatedAt = time.Now()
	return nil
}

// TransitionStatus validates and applies status transitions.
func (inv *Investment) TransitionStatus(newStatus InvestmentStatus) error {
	if !newStatus.Valid() {
		return fmt.Errorf("status %q: %w", newStatus, ErrInvalidStatus)
	}
	allowed := map[InvestmentStatus][]InvestmentStatus{
		StatusActive:    {StatusSold, StatusCancelled},
		StatusSold:      {},
		StatusCancelled: {},
	}
	transitions, ok := allowed[inv.Status]
	if !ok {
		return ErrInvalidStatus
	}
	for _, s := range transitions {
		if s == newStatus {
			inv.Status = newStatus
			return nil
		}
	}
	return fmt.Errorf("cannot transition from %q to %q: %w", inv.Status, newStatus, ErrStatusTransition)
}
