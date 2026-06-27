// Package api provides the gRPC API handler for the debt service.
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

	"github.com/aureum/debt-svc/internal/application"
	"github.com/aureum/debt-svc/internal/domain"
	pkgErr "github.com/aureum/pkg/errors"
	"github.com/aureum/pkg/telemetry"
	debtv1 "github.com/aureum/proto/gen/debt/debtv1"
)

// GRPCHandler implements the debtv1.DebtServiceServer interface.
type GRPCHandler struct {
	debtv1.UnimplementedDebtServiceServer
	svc application.DebtService
}

// NewGRPCHandler creates a new gRPC handler for debt operations.
func NewGRPCHandler(svc application.DebtService) *GRPCHandler {
	return &GRPCHandler{svc: svc}
}

// ── Debt ─────────────────────────────────────────────────────────────────────

// CreateDebt handles gRPC requests for creating a debt.
func (h *GRPCHandler) CreateDebt(ctx context.Context, req *debtv1.CreateDebtRequest) (*debtv1.Debt, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	resp, err := h.svc.CreateDebt(ctx, application.CreateDebtRequest{
		UserID:          userID,
		Name:            req.Name,
		Description:     req.Description,
		DebtType:        debtTypeFromProto(req.DebtType),
		TotalAmount:     req.TotalAmount,
		InterestRate:    req.InterestRate,
		StartDate:       req.StartDate,
		ExpectedEndDate: req.ExpectedEndDate,
		Status:          statusFromProto(req.Status),
		Creditor:        req.Creditor,
		IdempotencyKey:  req.IdempotencyKey,
	})
	if err != nil {
		telemetry.RecordRequest(ctx, "CreateDebt", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "CreateDebt", "ok", time.Since(start))
	return debtToProto(resp), nil
}

// GetDebt handles gRPC requests for retrieving a debt.
func (h *GRPCHandler) GetDebt(ctx context.Context, req *debtv1.GetDebtRequest) (*debtv1.Debt, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	resp, err := h.svc.GetDebt(ctx, req.Id, userID)
	if err != nil {
		telemetry.RecordRequest(ctx, "GetDebt", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "GetDebt", "ok", time.Since(start))
	return debtToProto(resp), nil
}

// UpdateDebt handles gRPC requests for updating a debt.
func (h *GRPCHandler) UpdateDebt(ctx context.Context, req *debtv1.UpdateDebtRequest) (*debtv1.Debt, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	appReq := application.UpdateDebtRequest{
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
	if req.DebtType != nil {
		t := debtTypeFromProto(*req.DebtType)
		appReq.DebtType = &t
	}
	if req.TotalAmount != nil {
		appReq.TotalAmount = req.TotalAmount
	}
	if req.InterestRate != nil {
		appReq.InterestRate = req.InterestRate
	}
	if req.ExpectedEndDate != nil {
		appReq.ExpectedEndDate = req.ExpectedEndDate
	}
	if req.Status != nil {
		s := statusFromProto(*req.Status)
		appReq.Status = &s
	}
	if req.Creditor != nil {
		appReq.Creditor = req.Creditor
	}
	resp, err := h.svc.UpdateDebt(ctx, appReq)
	if err != nil {
		telemetry.RecordRequest(ctx, "UpdateDebt", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "UpdateDebt", "ok", time.Since(start))
	return debtToProto(resp), nil
}

// DeleteDebt handles gRPC requests for deleting a debt.
func (h *GRPCHandler) DeleteDebt(ctx context.Context, req *debtv1.DeleteDebtRequest) (*emptypb.Empty, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	if err := h.svc.DeleteDebt(ctx, req.Id, userID); err != nil {
		telemetry.RecordRequest(ctx, "DeleteDebt", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "DeleteDebt", "ok", time.Since(start))
	return &emptypb.Empty{}, nil
}

// ListDebts handles gRPC requests for listing debts.
func (h *GRPCHandler) ListDebts(ctx context.Context, req *debtv1.ListDebtsRequest) (*debtv1.ListDebtsResponse, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	filter := domain.DebtFilter{
		Limit:  int(req.PageSize),
		Offset: offsetFromToken(req.PageToken),
	}
	if req.StatusFilter != nil {
		s := domain.DebtStatus(statusFromProto(*req.StatusFilter))
		filter.Status = &s
	}
	if req.TypeFilter != nil {
		t := domain.DebtType(debtTypeFromProto(*req.TypeFilter))
		filter.DebtType = &t
	}

	items, total, err := h.svc.ListDebts(ctx, userID, filter)
	if err != nil {
		telemetry.RecordRequest(ctx, "ListDebts", "error", time.Since(start))
		return nil, mapError(err)
	}

	protoItems := make([]*debtv1.Debt, len(items))
	for i, d := range items {
		protoItems[i] = debtToProto(d)
	}
	nextToken := ""
	if len(items) == int(req.PageSize) {
		nextToken = fmt.Sprintf("%d", filter.Offset+len(items))
	}
	telemetry.RecordRequest(ctx, "ListDebts", "ok", time.Since(start))
	return &debtv1.ListDebtsResponse{
		Debts:         protoItems,
		NextPageToken: nextToken,
		TotalCount:    int32(total),
	}, nil
}

// ── Payment ──────────────────────────────────────────────────────────────────

// RegisterPayment handles gRPC requests for registering a payment.
func (h *GRPCHandler) RegisterPayment(ctx context.Context, req *debtv1.RegisterPaymentRequest) (*debtv1.Payment, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	resp, err := h.svc.RegisterPayment(ctx, application.RegisterPaymentRequest{
		DebtID:         req.DebtId,
		UserID:         userID,
		Amount:         req.Amount,
		PaymentDate:    req.PaymentDate,
		Notes:          req.Notes,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		telemetry.RecordRequest(ctx, "RegisterPayment", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "RegisterPayment", "ok", time.Since(start))
	return paymentToProto(resp), nil
}

// ListPayments handles gRPC requests for listing payments.
func (h *GRPCHandler) ListPayments(ctx context.Context, req *debtv1.ListPaymentsRequest) (*debtv1.ListPaymentsResponse, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	filter := domain.PaymentFilter{
		DebtID: req.DebtId,
		Limit:  int(req.PageSize),
		Offset: offsetFromToken(req.PageToken),
	}
	if req.DateFrom != nil {
		filter.DateFrom = req.DateFrom
	}
	if req.DateTo != nil {
		filter.DateTo = req.DateTo
	}

	// Verify user has access to this debt
	_, err := h.svc.GetDebt(ctx, req.DebtId, userID)
	if err != nil {
		telemetry.RecordRequest(ctx, "ListPayments", "error", time.Since(start))
		return nil, mapError(err)
	}

	items, total, err := h.svc.ListPayments(ctx, filter)
	if err != nil {
		telemetry.RecordRequest(ctx, "ListPayments", "error", time.Since(start))
		return nil, mapError(err)
	}

	protoItems := make([]*debtv1.Payment, len(items))
	for i, p := range items {
		protoItems[i] = paymentToProto(p)
	}
	nextToken := ""
	if len(items) == int(req.PageSize) {
		nextToken = fmt.Sprintf("%d", filter.Offset+len(items))
	}
	telemetry.RecordRequest(ctx, "ListPayments", "ok", time.Since(start))
	return &debtv1.ListPaymentsResponse{
		Payments:      protoItems,
		NextPageToken: nextToken,
		TotalCount:    int32(total),
	}, nil
}

// ── Proto enum → domain string ──────────────────────────────────────────────

func debtTypeFromProto(t debtv1.DebtType) string {
	switch t {
	case debtv1.DebtType_PERSONAL_LOAN:
		return "personal_loan"
	case debtv1.DebtType_STUDENT_LOAN:
		return "student_loan"
	case debtv1.DebtType_MORTGAGE:
		return "mortgage"
	case debtv1.DebtType_CAR_LOAN:
		return "car_loan"
	case debtv1.DebtType_CREDIT_CARD_DEBT:
		return "credit_card_debt"
	case debtv1.DebtType_MEDICAL_DEBT:
		return "medical_debt"
	case debtv1.DebtType_OTHER_DEBT:
		return "other"
	default:
		return "other"
	}
}

func statusFromProto(s debtv1.DebtStatus) string {
	switch s {
	case debtv1.DebtStatus_ACTIVE:
		return "active"
	case debtv1.DebtStatus_PAUSED:
		return "paused"
	case debtv1.DebtStatus_PAID_OFF:
		return "paid_off"
	case debtv1.DebtStatus_DEFAULTED:
		return "defaulted"
	case debtv1.DebtStatus_SETTLED:
		return "settled"
	default:
		return "active"
	}
}

// ── Domain string → Proto enum ──────────────────────────────────────────────

func debtTypeToProto(t string) debtv1.DebtType {
	switch t {
	case "personal_loan":
		return debtv1.DebtType_PERSONAL_LOAN
	case "student_loan":
		return debtv1.DebtType_STUDENT_LOAN
	case "mortgage":
		return debtv1.DebtType_MORTGAGE
	case "car_loan":
		return debtv1.DebtType_CAR_LOAN
	case "credit_card_debt":
		return debtv1.DebtType_CREDIT_CARD_DEBT
	case "medical_debt":
		return debtv1.DebtType_MEDICAL_DEBT
	case "other":
		return debtv1.DebtType_OTHER_DEBT
	default:
		return debtv1.DebtType_DEBT_TYPE_UNSPECIFIED
	}
}

func statusToProto(s string) debtv1.DebtStatus {
	switch s {
	case "active":
		return debtv1.DebtStatus_ACTIVE
	case "paused":
		return debtv1.DebtStatus_PAUSED
	case "paid_off":
		return debtv1.DebtStatus_PAID_OFF
	case "defaulted":
		return debtv1.DebtStatus_DEFAULTED
	case "settled":
		return debtv1.DebtStatus_SETTLED
	default:
		return debtv1.DebtStatus_DEBT_STATUS_UNSPECIFIED
	}
}

// ── Response → Proto ─────────────────────────────────────────────────────────

func debtToProto(resp *application.DebtResponse) *debtv1.Debt {
	d := &debtv1.Debt{
		Id:              resp.ID,
		UserId:          resp.UserID,
		Name:            resp.Name,
		Description:     resp.Description,
		DebtType:        debtTypeToProto(resp.DebtType),
		TotalAmount:     resp.TotalAmount,
		RemainingAmount: resp.RemainingAmount,
		InterestRate:    resp.InterestRate,
		StartDate:       resp.StartDate,
		ExpectedEndDate: resp.ExpectedEndDate,
		Status:          statusToProto(resp.Status),
		Creditor:        resp.Creditor,
		CreatedAt:       timestamppb.New(time.Unix(resp.CreatedAt, 0)),
		UpdatedAt:       timestamppb.New(time.Unix(resp.UpdatedAt, 0)),
	}
	return d
}

func paymentToProto(resp *application.PaymentResponse) *debtv1.Payment {
	return &debtv1.Payment{
		Id:          resp.ID,
		DebtId:      resp.DebtID,
		UserId:      resp.UserID,
		Amount:      resp.Amount,
		PaymentDate: resp.PaymentDate,
		Notes:       resp.Notes,
		CreatedAt:   timestamppb.New(time.Unix(resp.CreatedAt, 0)),
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

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

// UserContext injects a user ID into the context for testing.
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
	case errors.Is(err, domain.ErrInvalidDebtType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidStatus):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrStatusTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrMissingField):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrPaymentExceedsBalance):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrDebtAlreadyPaid):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrAccessDenied):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
