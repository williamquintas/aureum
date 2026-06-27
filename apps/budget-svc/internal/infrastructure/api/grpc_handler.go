// Package api provides the gRPC API handler for the budget service.
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

	"github.com/aureum/budget-svc/internal/application"
	"github.com/aureum/budget-svc/internal/domain"
	pkgErr "github.com/aureum/pkg/errors"
	"github.com/aureum/pkg/telemetry"
	budgetv1 "github.com/aureum/proto/gen/budget/budgetv1"
)

// GRPCHandler implements the budgetv1.BudgetServiceServer interface.
type GRPCHandler struct {
	budgetv1.UnimplementedBudgetServiceServer
	svc application.BudgetService
}

// NewGRPCHandler creates a new gRPC handler for budget operations.
func NewGRPCHandler(svc application.BudgetService) *GRPCHandler {
	return &GRPCHandler{svc: svc}
}

// CreateBudget handles gRPC requests for creating a budget.
func (h *GRPCHandler) CreateBudget(ctx context.Context, req *budgetv1.CreateBudgetRequest) (*budgetv1.Budget, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)

	catDTOs := make([]application.CreateCategoryDTO, 0, len(req.Categories))
	for _, c := range req.Categories {
		catDTOs = append(catDTOs, application.CreateCategoryDTO{
			Name:        c.Name,
			LimitAmount: c.LimitAmount,
			Category:    c.Category,
		})
	}

	resp, err := h.svc.Create(ctx, application.CreateBudgetRequest{
		UserID:         userID,
		Name:           req.Name,
		Description:    req.Description,
		Period:         periodFromProto(req.Period),
		TotalLimit:     req.TotalLimit,
		StartDate:      req.StartDate,
		EndDate:        req.EndDate,
		Categories:     catDTOs,
		Status:         statusFromProto(req.Status),
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		telemetry.RecordRequest(ctx, "CreateBudget", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "CreateBudget", "ok", time.Since(start))
	return budgetToProto(resp), nil
}

// GetBudget handles gRPC requests for retrieving a budget.
func (h *GRPCHandler) GetBudget(ctx context.Context, req *budgetv1.GetBudgetRequest) (*budgetv1.Budget, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	resp, err := h.svc.Get(ctx, req.Id, userID)
	if err != nil {
		telemetry.RecordRequest(ctx, "GetBudget", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "GetBudget", "ok", time.Since(start))
	return getBudgetToProto(resp), nil
}

// UpdateBudget handles gRPC requests for updating a budget.
func (h *GRPCHandler) UpdateBudget(ctx context.Context, req *budgetv1.UpdateBudgetRequest) (*budgetv1.Budget, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)

	appReq := application.UpdateBudgetRequest{
		ID:             req.Id,
		UserID:         userID,
		IdempotencyKey: req.IdempotencyKey,
	}
	if req.Name != nil {
		appReq.Name = req.Name
	}
	if req.Description != nil {
		appReq.Description = req.Description
	}
	if req.Period != nil {
		p := periodFromProto(*req.Period)
		appReq.Period = &p
	}
	if req.TotalLimit != nil {
		appReq.TotalLimit = req.TotalLimit
	}
	if req.StartDate != nil {
		appReq.StartDate = req.StartDate
	}
	if req.EndDate != nil {
		appReq.EndDate = req.EndDate
	}
	if req.Status != nil {
		s := statusFromProto(*req.Status)
		appReq.Status = &s
	}

	resp, err := h.svc.Update(ctx, appReq)
	if err != nil {
		telemetry.RecordRequest(ctx, "UpdateBudget", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "UpdateBudget", "ok", time.Since(start))
	return getBudgetToProto(resp), nil
}

// DeleteBudget handles gRPC requests for deleting a budget.
func (h *GRPCHandler) DeleteBudget(ctx context.Context, req *budgetv1.DeleteBudgetRequest) (*emptypb.Empty, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	if err := h.svc.Delete(ctx, req.Id, userID); err != nil {
		telemetry.RecordRequest(ctx, "DeleteBudget", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "DeleteBudget", "ok", time.Since(start))
	return &emptypb.Empty{}, nil
}

// ListBudgets handles gRPC requests for listing budgets.
func (h *GRPCHandler) ListBudgets(ctx context.Context, req *budgetv1.ListBudgetsRequest) (*budgetv1.ListBudgetsResponse, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	filter := domain.BudgetFilter{
		Limit:  int(req.PageSize),
		Offset: offsetFromToken(req.PageToken),
	}
	if req.StatusFilter != nil {
		s := domain.BudgetStatus(statusFromProto(*req.StatusFilter))
		filter.Status = &s
	}
	if req.DateFrom != nil {
		filter.DateFrom = req.DateFrom
	}
	if req.DateTo != nil {
		filter.DateTo = req.DateTo
	}

	items, total, err := h.svc.List(ctx, userID, filter)
	if err != nil {
		telemetry.RecordRequest(ctx, "ListBudgets", "error", time.Since(start))
		return nil, mapError(err)
	}

	protoItems := make([]*budgetv1.Budget, len(items))
	for i, b := range items {
		protoItems[i] = getBudgetToProto(b)
	}

	nextToken := ""
	if len(items) == int(req.PageSize) {
		nextToken = fmt.Sprintf("%d", filter.Offset+len(items))
	}

	telemetry.RecordRequest(ctx, "ListBudgets", "ok", time.Since(start))
	return &budgetv1.ListBudgetsResponse{
		Budgets:       protoItems,
		NextPageToken: nextToken,
		TotalCount:    int32(total),
	}, nil
}

// GetBudgetSummary handles gRPC requests for retrieving a budget summary.
func (h *GRPCHandler) GetBudgetSummary(ctx context.Context, req *budgetv1.GetBudgetSummaryRequest) (*budgetv1.BudgetSummary, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	summary, err := h.svc.GetSummary(ctx, req.Id, userID)
	if err != nil {
		telemetry.RecordRequest(ctx, "GetBudgetSummary", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "GetBudgetSummary", "ok", time.Since(start))
	return summaryToProto(summary), nil
}

// ── Proto enum → Domain string ────────────────────────────────────────────

func periodFromProto(p budgetv1.BudgetPeriod) string {
	switch p {
	case budgetv1.BudgetPeriod_MONTHLY:
		return "monthly"
	case budgetv1.BudgetPeriod_BIMONTHLY:
		return "bimonthly"
	case budgetv1.BudgetPeriod_QUARTERLY:
		return "quarterly"
	case budgetv1.BudgetPeriod_SEMESTRAL:
		return "semestral"
	case budgetv1.BudgetPeriod_YEARLY:
		return "yearly"
	case budgetv1.BudgetPeriod_CUSTOM:
		return "custom"
	default:
		return "monthly"
	}
}

func statusFromProto(s budgetv1.BudgetStatus) string {
	switch s {
	case budgetv1.BudgetStatus_ACTIVE:
		return "active"
	case budgetv1.BudgetStatus_PAUSED:
		return "paused"
	case budgetv1.BudgetStatus_COMPLETED:
		return "completed"
	case budgetv1.BudgetStatus_CANCELLED:
		return "cancelled"
	default:
		return "active"
	}
}

// ── Domain string → Proto enum ────────────────────────────────────────────

func periodToProto(p string) budgetv1.BudgetPeriod {
	switch p {
	case "monthly":
		return budgetv1.BudgetPeriod_MONTHLY
	case "bimonthly":
		return budgetv1.BudgetPeriod_BIMONTHLY
	case "quarterly":
		return budgetv1.BudgetPeriod_QUARTERLY
	case "semestral":
		return budgetv1.BudgetPeriod_SEMESTRAL
	case "yearly":
		return budgetv1.BudgetPeriod_YEARLY
	case "custom":
		return budgetv1.BudgetPeriod_CUSTOM
	default:
		return budgetv1.BudgetPeriod_BUDGET_PERIOD_UNSPECIFIED
	}
}

func statusToProto(s string) budgetv1.BudgetStatus {
	switch s {
	case "active":
		return budgetv1.BudgetStatus_ACTIVE
	case "paused":
		return budgetv1.BudgetStatus_PAUSED
	case "completed":
		return budgetv1.BudgetStatus_COMPLETED
	case "cancelled":
		return budgetv1.BudgetStatus_CANCELLED
	default:
		return budgetv1.BudgetStatus_BUDGET_STATUS_UNSPECIFIED
	}
}

// ── Application DTO → Proto ───────────────────────────────────────────────

func budgetToProto(resp *application.CreateBudgetResponse) *budgetv1.Budget {
	proto := &budgetv1.Budget{
		Id:          resp.ID,
		UserId:      resp.UserID,
		Name:        resp.Name,
		Description: resp.Description,
		Period:      periodToProto(resp.Period),
		TotalLimit:  resp.TotalLimit,
		SpentAmount: resp.SpentAmount,
		Status:      statusToProto(resp.Status),
		StartDate:   resp.StartDate,
		EndDate:     resp.EndDate,
		CreatedAt:   timestamppb.New(timestampFromUnix(resp.CreatedAt)),
		UpdatedAt:   timestamppb.New(timestampFromUnix(resp.UpdatedAt)),
	}
	for _, c := range resp.Categories {
		proto.Categories = append(proto.Categories, categoryToProto(&c))
	}
	return proto
}

func getBudgetToProto(resp *application.GetBudgetResponse) *budgetv1.Budget {
	proto := &budgetv1.Budget{
		Id:          resp.ID,
		UserId:      resp.UserID,
		Name:        resp.Name,
		Description: resp.Description,
		Period:      periodToProto(resp.Period),
		TotalLimit:  resp.TotalLimit,
		SpentAmount: resp.SpentAmount,
		Status:      statusToProto(resp.Status),
		StartDate:   resp.StartDate,
		EndDate:     resp.EndDate,
		CreatedAt:   timestamppb.New(timestampFromUnix(resp.CreatedAt)),
		UpdatedAt:   timestamppb.New(timestampFromUnix(resp.UpdatedAt)),
	}
	for _, c := range resp.Categories {
		proto.Categories = append(proto.Categories, categoryToProto(&c))
	}
	return proto
}

func categoryToProto(c *application.CategoryDTO) *budgetv1.BudgetCategory {
	return &budgetv1.BudgetCategory{
		Id:          c.ID,
		BudgetId:    c.BudgetID,
		Name:        c.Name,
		LimitAmount: c.LimitAmount,
		SpentAmount: c.SpentAmount,
		Category:    c.Category,
	}
}

func summaryToProto(s *application.BudgetSummaryDTO) *budgetv1.BudgetSummary {
	proto := &budgetv1.BudgetSummary{
		BudgetId:        s.BudgetID,
		TotalLimit:      s.TotalLimit,
		TotalSpent:      s.TotalSpent,
		Remaining:       s.Remaining,
		UsagePercentage: s.UsagePercent,
		CategoryCount:   s.CategoryCount,
	}
	for _, c := range s.Categories {
		proto.Categories = append(proto.Categories, &budgetv1.CategorySummary{
			CategoryId:      c.CategoryID,
			Name:            c.Name,
			Category:        c.Category,
			LimitAmount:     c.LimitAmount,
			SpentAmount:     c.SpentAmount,
			Remaining:       c.Remaining,
			UsagePercentage: c.UsagePercent,
		})
	}
	return proto
}

// ── Helpers ───────────────────────────────────────────────────────────────

func offsetFromToken(token string) int {
	if token == "" {
		return 0
	}
	var offset int
	_, _ = fmt.Sscanf(token, "%d", &offset)
	return offset
}

func timestampFromUnix(unix int64) time.Time {
	if unix == 0 {
		return time.Time{}
	}
	return time.Unix(unix, 0)
}

// ── Context / Auth ────────────────────────────────────────────────────────

type ctxKey string

const userIDKey ctxKey = "user_id"

func mustExtractUserID(ctx context.Context) string {
	uid, _ := ctx.Value(userIDKey).(string)
	return uid
}

// UserContext injects a user ID into the context for testing.
func UserContext(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// ── Error mapping ─────────────────────────────────────────────────────────

func mapError(err error) error {
	if grpcErr := pkgErr.MapToGRPC(err); status.Code(grpcErr) != codes.Unknown {
		return grpcErr
	}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrNegativeAmount):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInsufficientBudget):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidPeriod):
		return status.Error(codes.InvalidArgument, err.Error())
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
	case errors.Is(err, domain.ErrInvalidDate):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidDateRange):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrCategoryLimit):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
