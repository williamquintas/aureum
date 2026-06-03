package application

import "github.com/aureum/investment-svc/internal/domain"

// ── Investment DTOs ──────────────────────────────────────────────────────────

type CreateInvestmentRequest struct {
	UserID         string
	Name           string
	Ticker         string
	AssetType      string
	Quantity       int64
	AveragePrice   int64
	Broker         string
	Status         string
	IdempotencyKey string
}

type CreateInvestmentResponse struct {
	ID            string
	UserID        string
	Name          string
	Ticker        string
	AssetType     string
	Quantity      int64
	AveragePrice  int64
	TotalInvested int64
	Status        string
	Broker        string
	CreatedAt     int64
	UpdatedAt     int64
}

type GetInvestmentResponse struct {
	ID            string
	UserID        string
	Name          string
	Ticker        string
	AssetType     string
	Quantity      int64
	AveragePrice  int64
	TotalInvested int64
	Status        string
	Broker        string
	CreatedAt     int64
	UpdatedAt     int64
}

type UpdateInvestmentRequest struct {
	ID             string
	UserID         string
	Name           *string
	Ticker         *string
	AssetType      *string
	Quantity       *int64
	AveragePrice   *int64
	Broker         *string
	Status         *string
	IdempotencyKey string
}

// ── Transaction DTOs ─────────────────────────────────────────────────────────

type RecordTransactionRequest struct {
	UserID          string
	InvestmentID    string
	TransactionType string
	Quantity        int64
	UnitPrice       int64
	TransactionDate string
	Notes           string
	IdempotencyKey  string
}

type RecordTransactionResponse struct {
	ID              string
	InvestmentID    string
	UserID          string
	TransactionType string
	Quantity        int64
	UnitPrice       int64
	TotalAmount     int64
	TransactionDate string
	Notes           string
	CreatedAt       int64
}

type GetTransactionResponse struct {
	ID              string
	InvestmentID    string
	UserID          string
	TransactionType string
	Quantity        int64
	UnitPrice       int64
	TotalAmount     int64
	TransactionDate string
	Notes           string
	CreatedAt       int64
}

// ── Portfolio DTOs ───────────────────────────────────────────────────────────

type PortfolioSummaryResponse struct {
	TotalInvested     int64
	CurrentValue      int64
	TotalReturn       int64
	ReturnPercentage  float64
	ActiveInvestments int32
	Allocation        []AssetAllocationDTO
}

type AssetAllocationDTO struct {
	AssetType    string
	Invested     int64
	CurrentValue int64
	Percentage   float64
}

// ── Enum converters ──────────────────────────────────────────────────────────

func toDomainAssetType(t string) (domain.AssetType, error) {
	switch t {
	case "stock":
		return domain.AssetTypeStock, nil
	case "etf":
		return domain.AssetTypeETF, nil
	case "real_estate_fund":
		return domain.AssetTypeRealEstateFund, nil
	case "treasury":
		return domain.AssetTypeTreasury, nil
	case "cdb":
		return domain.AssetTypeCDB, nil
	case "lci":
		return domain.AssetTypeLCI, nil
	case "lca":
		return domain.AssetTypeLCA, nil
	case "crypto":
		return domain.AssetTypeCrypto, nil
	case "pension":
		return domain.AssetTypePension, nil
	case "fund":
		return domain.AssetTypeFund, nil
	case "dollar":
		return domain.AssetTypeDollar, nil
	case "gold":
		return domain.AssetTypeGold, nil
	case "other":
		return domain.AssetTypeOther, nil
	default:
		return "", domain.ErrInvalidAssetType
	}
}

func toDomainTransactionType(t string) (domain.TransactionType, error) {
	switch t {
	case "buy":
		return domain.TransactionBuy, nil
	case "sell":
		return domain.TransactionSell, nil
	case "dividend":
		return domain.TransactionDividend, nil
	case "jcp":
		return domain.TransactionJCP, nil
	case "amortization":
		return domain.TransactionAmortization, nil
	default:
		return "", domain.ErrInvalidTransactionType
	}
}

func toDomainStatus(s string) (domain.InvestmentStatus, error) {
	switch s {
	case "active":
		return domain.StatusActive, nil
	case "sold":
		return domain.StatusSold, nil
	case "cancelled":
		return domain.StatusCancelled, nil
	default:
		return "", domain.ErrInvalidStatus
	}
}
