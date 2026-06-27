// Package api provides the gRPC API handler for the investment service.
package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/aureum/investment-svc/internal/application"
	"github.com/aureum/investment-svc/internal/domain"
	pkgErr "github.com/aureum/pkg/errors"
	"github.com/aureum/pkg/telemetry"
	investmentv1 "github.com/aureum/proto/gen/investment/investmentv1"
)

// GRPCHandler implements the gRPC InvestmentService server.
type GRPCHandler struct {
	investmentv1.UnimplementedInvestmentServiceServer
	svc application.InvestmentService
}

// NewGRPCHandler creates a new gRPC handler with the given application service.
func NewGRPCHandler(svc application.InvestmentService) *GRPCHandler {
	return &GRPCHandler{svc: svc}
}

// ── Investment ──────────────────────────────────────────────────────────────

// CreateInvestment handles the gRPC CreateInvestment request.
func (h *GRPCHandler) CreateInvestment(ctx context.Context, req *investmentv1.CreateInvestmentRequest) (*investmentv1.Investment, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	resp, err := h.svc.CreateInvestment(ctx, application.CreateInvestmentRequest{
		UserID:         userID,
		Name:           req.Name,
		Ticker:         req.Ticker,
		AssetType:      assetTypeFromProto(req.AssetType),
		Quantity:       req.Quantity,
		AveragePrice:   req.AveragePrice,
		Broker:         req.Broker,
		Status:         investmentStatusFromProto(req.Status),
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		telemetry.RecordRequest(ctx, "CreateInvestment", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "CreateInvestment", "ok", time.Since(start))
	return investmentFromCreate(resp), nil
}

// GetInvestment handles the gRPC GetInvestment request.
func (h *GRPCHandler) GetInvestment(ctx context.Context, req *investmentv1.GetInvestmentRequest) (*investmentv1.Investment, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	resp, err := h.svc.GetInvestment(ctx, req.Id, userID)
	if err != nil {
		telemetry.RecordRequest(ctx, "GetInvestment", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "GetInvestment", "ok", time.Since(start))
	return investmentFromGet(resp), nil
}

// UpdateInvestment handles the gRPC UpdateInvestment request.
func (h *GRPCHandler) UpdateInvestment(ctx context.Context, req *investmentv1.UpdateInvestmentRequest) (*investmentv1.Investment, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	appReq := application.UpdateInvestmentRequest{
		ID:             req.Id,
		UserID:         userID,
		IdempotencyKey: req.IdempotencyKey,
	}
	if req.Name != nil {
		appReq.Name = req.Name
	}
	if req.Ticker != nil {
		appReq.Ticker = req.Ticker
	}
	if req.AssetType != nil {
		t := assetTypeFromProto(*req.AssetType)
		appReq.AssetType = &t
	}
	if req.Quantity != nil {
		appReq.Quantity = req.Quantity
	}
	if req.AveragePrice != nil {
		appReq.AveragePrice = req.AveragePrice
	}
	if req.Broker != nil {
		appReq.Broker = req.Broker
	}
	if req.Status != nil {
		s := investmentStatusFromProto(*req.Status)
		appReq.Status = &s
	}

	resp, err := h.svc.UpdateInvestment(ctx, appReq)
	if err != nil {
		telemetry.RecordRequest(ctx, "UpdateInvestment", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "UpdateInvestment", "ok", time.Since(start))
	return investmentFromGet(resp), nil
}

// DeleteInvestment handles the gRPC DeleteInvestment request.
func (h *GRPCHandler) DeleteInvestment(ctx context.Context, req *investmentv1.DeleteInvestmentRequest) (*emptypb.Empty, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	if err := h.svc.DeleteInvestment(ctx, req.Id, userID); err != nil {
		telemetry.RecordRequest(ctx, "DeleteInvestment", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "DeleteInvestment", "ok", time.Since(start))
	return &emptypb.Empty{}, nil
}

// ListInvestments handles the gRPC ListInvestments request.
func (h *GRPCHandler) ListInvestments(ctx context.Context, req *investmentv1.ListInvestmentsRequest) (*investmentv1.ListInvestmentsResponse, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	filter := domain.InvestmentFilter{
		Limit:  int(req.PageSize),
		Offset: offsetFromToken(req.PageToken),
	}
	if req.TypeFilter != nil {
		t := domain.AssetType(assetTypeFromProto(*req.TypeFilter))
		filter.TypeFilter = &t
	}
	if req.StatusFilter != nil {
		s := domain.InvestmentStatus(investmentStatusFromProto(*req.StatusFilter))
		filter.StatusFilter = &s
	}

	items, total, err := h.svc.ListInvestments(ctx, userID, filter)
	if err != nil {
		telemetry.RecordRequest(ctx, "ListInvestments", "error", time.Since(start))
		return nil, mapError(err)
	}

	protoItems := make([]*investmentv1.Investment, len(items))
	for i, inv := range items {
		protoItems[i] = investmentFromGet(inv)
	}
	telemetry.RecordRequest(ctx, "ListInvestments", "ok", time.Since(start))
	return &investmentv1.ListInvestmentsResponse{
		Investments: protoItems,
		TotalCount:  int32(total),
	}, nil
}

// ── Transaction ─────────────────────────────────────────────────────────────

// RecordTransaction handles the gRPC RecordTransaction request.
func (h *GRPCHandler) RecordTransaction(ctx context.Context, req *investmentv1.RecordTransactionRequest) (*investmentv1.InvestmentTransaction, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	resp, err := h.svc.RecordTransaction(ctx, application.RecordTransactionRequest{
		UserID:          userID,
		InvestmentID:    req.InvestmentId,
		TransactionType: transactionTypeFromProto(req.TransactionType),
		Quantity:        req.Quantity,
		UnitPrice:       req.UnitPrice,
		TransactionDate: req.TransactionDate,
		Notes:           req.Notes,
		IdempotencyKey:  req.IdempotencyKey,
	})
	if err != nil {
		telemetry.RecordRequest(ctx, "RecordTransaction", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "RecordTransaction", "ok", time.Since(start))
	return transactionFromRecord(resp), nil
}

// ListTransactions handles the gRPC ListTransactions request.
func (h *GRPCHandler) ListTransactions(ctx context.Context, req *investmentv1.ListTransactionsRequest) (*investmentv1.ListTransactionsResponse, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	filter := domain.TransactionFilter{
		Limit:  int(req.PageSize),
		Offset: offsetFromToken(req.PageToken),
	}
	if req.TypeFilter != nil {
		t := domain.TransactionType(transactionTypeFromProto(*req.TypeFilter))
		filter.TypeFilter = &t
	}
	if req.DateFrom != nil {
		filter.DateFrom = req.DateFrom
	}
	if req.DateTo != nil {
		filter.DateTo = req.DateTo
	}

	items, total, err := h.svc.ListTransactions(ctx, userID, req.InvestmentId, filter)
	if err != nil {
		telemetry.RecordRequest(ctx, "ListTransactions", "error", time.Since(start))
		return nil, mapError(err)
	}

	protoItems := make([]*investmentv1.InvestmentTransaction, len(items))
	for i, t := range items {
		protoItems[i] = transactionFromGet(t)
	}
	telemetry.RecordRequest(ctx, "ListTransactions", "ok", time.Since(start))
	return &investmentv1.ListTransactionsResponse{
		Transactions: protoItems,
		TotalCount:   int32(total),
	}, nil
}

// ── Portfolio ───────────────────────────────────────────────────────────────

// GetPortfolioSummary handles the gRPC GetPortfolioSummary request.
func (h *GRPCHandler) GetPortfolioSummary(ctx context.Context, _ *investmentv1.GetPortfolioSummaryRequest) (*investmentv1.PortfolioSummary, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	resp, err := h.svc.GetPortfolioSummary(ctx, userID)
	if err != nil {
		telemetry.RecordRequest(ctx, "GetPortfolioSummary", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "GetPortfolioSummary", "ok", time.Since(start))
	return portfolioSummaryToProto(resp), nil
}

// ── Proto enum → domain string ────────────────────────────────────────────

func assetTypeFromProto(t investmentv1.AssetType) string {
	switch t {
	case investmentv1.AssetType_STOCK:
		return "stock"
	case investmentv1.AssetType_ETF:
		return "etf"
	case investmentv1.AssetType_REAL_ESTATE_FUND:
		return "real_estate_fund"
	case investmentv1.AssetType_TREASURY:
		return "treasury"
	case investmentv1.AssetType_CDB:
		return "cdb"
	case investmentv1.AssetType_LCI:
		return "lci"
	case investmentv1.AssetType_LCA:
		return "lca"
	case investmentv1.AssetType_CRYPTO:
		return "crypto"
	case investmentv1.AssetType_PENSION:
		return "pension"
	case investmentv1.AssetType_FUND:
		return "fund"
	case investmentv1.AssetType_DOLLAR:
		return "dollar"
	case investmentv1.AssetType_GOLD:
		return "gold"
	case investmentv1.AssetType_OTHER_ASSET:
		return "other"
	default:
		return "other"
	}
}

func transactionTypeFromProto(t investmentv1.TransactionType) string {
	switch t {
	case investmentv1.TransactionType_BUY:
		return "buy"
	case investmentv1.TransactionType_SELL:
		return "sell"
	case investmentv1.TransactionType_DIVIDEND:
		return "dividend"
	case investmentv1.TransactionType_JCP:
		return "jcp"
	case investmentv1.TransactionType_AMORTIZATION:
		return "amortization"
	default:
		return "buy"
	}
}

func investmentStatusFromProto(s investmentv1.InvestmentStatus) string {
	switch s {
	case investmentv1.InvestmentStatus_ACTIVE:
		return "active"
	case investmentv1.InvestmentStatus_SOLD:
		return "sold"
	case investmentv1.InvestmentStatus_CANCELLED:
		return "cancelled"
	default:
		return "active"
	}
}

// ── Domain string → Proto enum ────────────────────────────────────────────

func assetTypeToProto(t string) investmentv1.AssetType {
	switch t {
	case "stock":
		return investmentv1.AssetType_STOCK
	case "etf":
		return investmentv1.AssetType_ETF
	case "real_estate_fund":
		return investmentv1.AssetType_REAL_ESTATE_FUND
	case "treasury":
		return investmentv1.AssetType_TREASURY
	case "cdb":
		return investmentv1.AssetType_CDB
	case "lci":
		return investmentv1.AssetType_LCI
	case "lca":
		return investmentv1.AssetType_LCA
	case "crypto":
		return investmentv1.AssetType_CRYPTO
	case "pension":
		return investmentv1.AssetType_PENSION
	case "fund":
		return investmentv1.AssetType_FUND
	case "dollar":
		return investmentv1.AssetType_DOLLAR
	case "gold":
		return investmentv1.AssetType_GOLD
	case "other":
		return investmentv1.AssetType_OTHER_ASSET
	default:
		return investmentv1.AssetType_ASSET_TYPE_UNSPECIFIED
	}
}

func transactionTypeToProto(t string) investmentv1.TransactionType {
	switch t {
	case "buy":
		return investmentv1.TransactionType_BUY
	case "sell":
		return investmentv1.TransactionType_SELL
	case "dividend":
		return investmentv1.TransactionType_DIVIDEND
	case "jcp":
		return investmentv1.TransactionType_JCP
	case "amortization":
		return investmentv1.TransactionType_AMORTIZATION
	default:
		return investmentv1.TransactionType_TRANSACTION_TYPE_UNSPECIFIED
	}
}

func investmentStatusToProto(s string) investmentv1.InvestmentStatus {
	switch s {
	case "active":
		return investmentv1.InvestmentStatus_ACTIVE
	case "sold":
		return investmentv1.InvestmentStatus_SOLD
	case "cancelled":
		return investmentv1.InvestmentStatus_CANCELLED
	default:
		return investmentv1.InvestmentStatus_INVESTMENT_STATUS_UNSPECIFIED
	}
}

// ── DTO → Proto ───────────────────────────────────────────────────────────

func investmentFromCreate(resp *application.CreateInvestmentResponse) *investmentv1.Investment {
	return &investmentv1.Investment{
		Id:            resp.ID,
		UserId:        resp.UserID,
		Name:          resp.Name,
		Ticker:        resp.Ticker,
		AssetType:     assetTypeToProto(resp.AssetType),
		Quantity:      resp.Quantity,
		AveragePrice:  resp.AveragePrice,
		TotalInvested: resp.TotalInvested,
		Status:        investmentStatusToProto(resp.Status),
		Broker:        resp.Broker,
		CreatedAt:     timestamppb.New(time.Unix(resp.CreatedAt, 0)),
		UpdatedAt:     timestamppb.New(time.Unix(resp.UpdatedAt, 0)),
	}
}

func investmentFromGet(resp *application.GetInvestmentResponse) *investmentv1.Investment {
	return &investmentv1.Investment{
		Id:            resp.ID,
		UserId:        resp.UserID,
		Name:          resp.Name,
		Ticker:        resp.Ticker,
		AssetType:     assetTypeToProto(resp.AssetType),
		Quantity:      resp.Quantity,
		AveragePrice:  resp.AveragePrice,
		TotalInvested: resp.TotalInvested,
		Status:        investmentStatusToProto(resp.Status),
		Broker:        resp.Broker,
		CreatedAt:     timestamppb.New(time.Unix(resp.CreatedAt, 0)),
		UpdatedAt:     timestamppb.New(time.Unix(resp.UpdatedAt, 0)),
	}
}

func transactionFromRecord(resp *application.RecordTransactionResponse) *investmentv1.InvestmentTransaction {
	return &investmentv1.InvestmentTransaction{
		Id:              resp.ID,
		InvestmentId:    resp.InvestmentID,
		UserId:          resp.UserID,
		TransactionType: transactionTypeToProto(resp.TransactionType),
		Quantity:        resp.Quantity,
		UnitPrice:       resp.UnitPrice,
		TotalAmount:     resp.TotalAmount,
		TransactionDate: resp.TransactionDate,
		Notes:           resp.Notes,
		CreatedAt:       timestamppb.New(time.Unix(resp.CreatedAt, 0)),
	}
}

func transactionFromGet(resp *application.GetTransactionResponse) *investmentv1.InvestmentTransaction {
	return &investmentv1.InvestmentTransaction{
		Id:              resp.ID,
		InvestmentId:    resp.InvestmentID,
		UserId:          resp.UserID,
		TransactionType: transactionTypeToProto(resp.TransactionType),
		Quantity:        resp.Quantity,
		UnitPrice:       resp.UnitPrice,
		TotalAmount:     resp.TotalAmount,
		TransactionDate: resp.TransactionDate,
		Notes:           resp.Notes,
		CreatedAt:       timestamppb.New(time.Unix(resp.CreatedAt, 0)),
	}
}

func portfolioSummaryToProto(resp *application.PortfolioSummaryResponse) *investmentv1.PortfolioSummary {
	allocations := make([]*investmentv1.AssetAllocation, len(resp.Allocation))
	for i, a := range resp.Allocation {
		allocations[i] = &investmentv1.AssetAllocation{
			AssetType:    assetTypeToProto(a.AssetType),
			Invested:     a.Invested,
			CurrentValue: a.CurrentValue,
			Percentage:   a.Percentage,
		}
	}
	return &investmentv1.PortfolioSummary{
		TotalInvested:     resp.TotalInvested,
		CurrentValue:      resp.CurrentValue,
		TotalReturn:       resp.TotalReturn,
		ReturnPercentage:  resp.ReturnPercentage,
		ActiveInvestments: resp.ActiveInvestments,
		Allocation:        allocations,
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func offsetFromToken(token string) int {
	if token == "" {
		return 0
	}
	var offset int
	_, _ = fmt.Sscanf(token, "%d", &offset)
	return offset
}

type ctxKey string

const userIDKey ctxKey = "user_id"

func mustExtractUserID(ctx context.Context) string {
	uid, _ := ctx.Value(userIDKey).(string)
	return uid
}

// UserContext embeds the user ID into the context.
func UserContext(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func mapError(err error) error {
	if grpcErr := pkgErr.MapToGRPC(err); status.Code(grpcErr) != codes.Unknown {
		return grpcErr
	}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrNegativeAmount):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidAssetType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidTransactionType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidQuantity):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidPrice):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInsufficientQuantity):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrInvalidStatus):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrStatusTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrMissingField):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidEnum):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrAccessDenied):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
