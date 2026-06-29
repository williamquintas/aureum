// Package api implements the gRPC transport layer for the transaction service.
package api

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pkgErr "github.com/aureum/pkg/errors"
	transactionv1 "github.com/aureum/proto/gen/transaction/transactionv1"
	"github.com/aureum/transaction-svc/internal/application"
	"github.com/aureum/transaction-svc/internal/domain"
)

const (
	strPending = "pending"
	strOther   = "other"
)

// GRPCHandler implements the gRPC TransactionService server.
type GRPCHandler struct {
	transactionv1.UnimplementedTransactionServiceServer
	svc *application.Service
}

// NewGRPCHandler creates a new gRPC handler with the given application service.
func NewGRPCHandler(svc *application.Service) *GRPCHandler {
	return &GRPCHandler{svc: svc}
}

// CreateIncome handles the gRPC request for creating a new income record.
func (h *GRPCHandler) CreateIncome(ctx context.Context,
	req *transactionv1.CreateIncomeRequest) (*transactionv1.Income, error) {
	userID := mustExtractUserID(ctx)
	resp, err := h.svc.CreateIncome(ctx, application.CreateIncomeRequest{
		UserID:         userID,
		Description:    req.Description,
		Source:         req.Source,
		IncomeType:     incomeTypeFromProto(req.IncomeType),
		ReceivedDate:   req.ReceivedDate,
		ReceivedAmount: req.ReceivedAmount,
		Status:         statusFromProto(req.Status),
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return incomeToProto(resp.ID, resp.UserID, resp.Description, resp.Source,
		resp.IncomeType, resp.ReceivedDate, resp.ReceivedAmount, resp.Status), nil
}

// GetIncome handles the gRPC request for retrieving an income record.
func (h *GRPCHandler) GetIncome(ctx context.Context,
	req *transactionv1.GetIncomeRequest) (*transactionv1.Income, error) {
	userID := mustExtractUserID(ctx)
	resp, err := h.svc.GetIncome(ctx, req.Id, userID)
	if err != nil {
		return nil, mapError(err)
	}
	return incomeToProto(resp.ID, resp.UserID, resp.Description, resp.Source,
		resp.IncomeType, resp.ReceivedDate, resp.ReceivedAmount, resp.Status), nil
}

// UpdateIncome handles the gRPC request for updating an income record.
func (h *GRPCHandler) UpdateIncome(ctx context.Context,
	req *transactionv1.UpdateIncomeRequest) (*transactionv1.Income, error) {
	userID := mustExtractUserID(ctx)
	appReq := application.UpdateIncomeRequest{
		ID:             req.Id,
		UserID:         userID,
		Description:    req.Description,
		Source:         req.Source,
		ReceivedDate:   req.ReceivedDate,
		ReceivedAmount: req.ReceivedAmount,
		IdempotencyKey: req.IdempotencyKey,
	}
	if req.IncomeType != nil {
		t := incomeTypeFromProto(*req.IncomeType)
		appReq.IncomeType = &t
	}
	if req.Status != nil {
		s := statusFromProto(*req.Status)
		appReq.Status = &s
	}
	resp, err := h.svc.UpdateIncome(ctx, appReq)
	if err != nil {
		return nil, mapError(err)
	}
	return incomeToProto(resp.ID, resp.UserID, resp.Description, resp.Source,
		resp.IncomeType, resp.ReceivedDate, resp.ReceivedAmount, resp.Status), nil
}

// DeleteIncome handles the gRPC request for deleting an income record.
func (h *GRPCHandler) DeleteIncome(ctx context.Context,
	req *transactionv1.DeleteIncomeRequest) (*emptypb.Empty, error) {
	userID := mustExtractUserID(ctx)
	if err := h.svc.DeleteIncome(ctx, req.Id, userID); err != nil {
		return nil, mapError(err)
	}
	return &emptypb.Empty{}, nil
}

// ListIncomes handles the gRPC request for listing income records.
func (h *GRPCHandler) ListIncomes(ctx context.Context,
	req *transactionv1.ListIncomesRequest) (*transactionv1.ListIncomesResponse, error) {
	userID := mustExtractUserID(ctx)
	filter := domain.IncomeFilter{
		Limit:  int(req.PageSize),
		Offset: offsetFromToken(req.PageToken),
	}
	if req.StatusFilter != nil {
		s := domain.TransactionStatus(statusFromProto(*req.StatusFilter))
		filter.Status = &s
	}
	if req.DateFrom != nil {
		filter.DateFrom = req.DateFrom
	}
	if req.DateTo != nil {
		filter.DateTo = req.DateTo
	}

	items, total, err := h.svc.ListIncomes(ctx, userID, filter)
	if err != nil {
		return nil, mapError(err)
	}

	protoItems := make([]*transactionv1.Income, len(items))
	for i, inc := range items {
		protoItems[i] = incomeToProto(inc.ID, inc.UserID, inc.Description, inc.Source,
			inc.IncomeType, inc.ReceivedDate, inc.ReceivedAmount, inc.Status)
	}
	//nolint:gosec // G115: safe conversion within app bounds
	return &transactionv1.ListIncomesResponse{Incomes: protoItems, TotalCount: int32(total)}, nil
}

// CreateFixedExpense handles the gRPC request for creating a fixed expense.
func (h *GRPCHandler) CreateFixedExpense(ctx context.Context,
	req *transactionv1.CreateFixedExpenseRequest) (*transactionv1.FixedExpense, error) {
	userID := mustExtractUserID(ctx)
	resp, err := h.svc.CreateFixedExpense(ctx, application.CreateFixedExpenseRequest{
		UserID:         userID,
		Description:    req.Description,
		Category:       req.Category,
		DayOfMonth:     int(req.DayOfMonth),
		PaymentMethod:  paymentMethodFromProto(req.PaymentMethod),
		Status:         statusFromProto(req.Status),
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return fixedExpenseToProto(resp), nil
}

// GetFixedExpense handles the gRPC request for retrieving a fixed expense.
func (h *GRPCHandler) GetFixedExpense(ctx context.Context,
	req *transactionv1.GetFixedExpenseRequest) (*transactionv1.FixedExpense, error) {
	userID := mustExtractUserID(ctx)
	resp, err := h.svc.GetFixedExpense(ctx, req.Id, userID)
	if err != nil {
		return nil, mapError(err)
	}
	return fixedExpenseToProto(resp), nil
}

// UpdateFixedExpense handles the gRPC request for updating a fixed expense.
func (h *GRPCHandler) UpdateFixedExpense(ctx context.Context,
	req *transactionv1.UpdateFixedExpenseRequest) (*transactionv1.FixedExpense, error) {
	userID := mustExtractUserID(ctx)
	appReq := application.UpdateFixedExpenseRequest{
		ID:             req.Id,
		UserID:         userID,
		Description:    req.Description,
		Category:       req.Category,
		IdempotencyKey: req.IdempotencyKey,
	}
	if req.DayOfMonth != nil {
		d := int(*req.DayOfMonth)
		appReq.DayOfMonth = &d
	}
	if req.PaymentMethod != nil {
		pm := paymentMethodFromProto(*req.PaymentMethod)
		appReq.PaymentMethod = &pm
	}
	if req.Status != nil {
		s := statusFromProto(*req.Status)
		appReq.Status = &s
	}
	resp, err := h.svc.UpdateFixedExpense(ctx, appReq)
	if err != nil {
		return nil, mapError(err)
	}
	return fixedExpenseToProto(resp), nil
}

// DeleteFixedExpense handles the gRPC request for deleting a fixed expense.
func (h *GRPCHandler) DeleteFixedExpense(ctx context.Context,
	req *transactionv1.DeleteFixedExpenseRequest) (*emptypb.Empty, error) {
	userID := mustExtractUserID(ctx)
	if err := h.svc.DeleteFixedExpense(ctx, req.Id, userID); err != nil {
		return nil, mapError(err)
	}
	return &emptypb.Empty{}, nil
}

// ListFixedExpenses handles the gRPC request for listing fixed expenses.
func (h *GRPCHandler) ListFixedExpenses(ctx context.Context,
	req *transactionv1.ListFixedExpensesRequest) (*transactionv1.ListFixedExpensesResponse, error) {
	userID := mustExtractUserID(ctx)
	filter := domain.FixedExpenseFilter{
		Limit:  int(req.PageSize),
		Offset: offsetFromToken(req.PageToken),
	}
	if req.StatusFilter != nil {
		s := domain.TransactionStatus(statusFromProto(*req.StatusFilter))
		filter.Status = &s
	}
	if req.DayFrom != nil {
		d := int(*req.DayFrom)
		filter.DayFrom = &d
	}
	if req.DayTo != nil {
		d := int(*req.DayTo)
		filter.DayTo = &d
	}

	items, total, err := h.svc.ListFixedExpenses(ctx, userID, filter)
	if err != nil {
		return nil, mapError(err)
	}

	protoItems := make([]*transactionv1.FixedExpense, len(items))
	for i, fe := range items {
		protoItems[i] = fixedExpenseToProto(fe)
	}
	//nolint:gosec // G115: safe conversion
	return &transactionv1.ListFixedExpensesResponse{FixedExpenses: protoItems, TotalCount: int32(total)}, nil
}

// CreateVariableExpense handles the gRPC request for creating a variable expense.
func (h *GRPCHandler) CreateVariableExpense(ctx context.Context,
	req *transactionv1.CreateVariableExpenseRequest) (*transactionv1.VariableExpense, error) {
	userID := mustExtractUserID(ctx)
	resp, err := h.svc.CreateVariableExpense(ctx, application.CreateVariableExpenseRequest{
		UserID:         userID,
		Description:    req.Description,
		Destination:    req.Destination,
		Category:       req.Category,
		ExpenseType:    expenseTypeFromProto(req.ExpenseType),
		PaymentMethod:  paymentMethodFromProto(req.PaymentMethod),
		PaymentDate:    req.PaymentDate,
		PaidAmount:     req.PaidAmount,
		Status:         statusFromProto(req.Status),
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return variableExpenseToProto(resp), nil
}

// GetVariableExpense handles the gRPC request for retrieving a variable expense.
func (h *GRPCHandler) GetVariableExpense(ctx context.Context,
	req *transactionv1.GetVariableExpenseRequest) (*transactionv1.VariableExpense, error) {
	userID := mustExtractUserID(ctx)
	resp, err := h.svc.GetVariableExpense(ctx, req.Id, userID)
	if err != nil {
		return nil, mapError(err)
	}
	return variableExpenseToProto(resp), nil
}

// UpdateVariableExpense handles the gRPC request for updating a variable expense.
func (h *GRPCHandler) UpdateVariableExpense(ctx context.Context,
	req *transactionv1.UpdateVariableExpenseRequest) (*transactionv1.VariableExpense, error) {
	userID := mustExtractUserID(ctx)
	appReq := application.UpdateVariableExpenseRequest{
		ID:             req.Id,
		UserID:         userID,
		Description:    req.Description,
		Destination:    req.Destination,
		Category:       req.Category,
		PaymentDate:    req.PaymentDate,
		PaidAmount:     req.PaidAmount,
		IdempotencyKey: req.IdempotencyKey,
	}
	if req.ExpenseType != nil {
		et := expenseTypeFromProto(*req.ExpenseType)
		appReq.ExpenseType = &et
	}
	if req.PaymentMethod != nil {
		pm := paymentMethodFromProto(*req.PaymentMethod)
		appReq.PaymentMethod = &pm
	}
	if req.Status != nil {
		s := statusFromProto(*req.Status)
		appReq.Status = &s
	}
	resp, err := h.svc.UpdateVariableExpense(ctx, appReq)
	if err != nil {
		return nil, mapError(err)
	}
	return variableExpenseToProto(resp), nil
}

// DeleteVariableExpense handles the gRPC request for deleting a variable expense.
func (h *GRPCHandler) DeleteVariableExpense(ctx context.Context,
	req *transactionv1.DeleteVariableExpenseRequest) (*emptypb.Empty, error) {
	userID := mustExtractUserID(ctx)
	if err := h.svc.DeleteVariableExpense(ctx, req.Id, userID); err != nil {
		return nil, mapError(err)
	}
	return &emptypb.Empty{}, nil
}

// ListVariableExpenses handles the gRPC request for listing variable expenses.
func (h *GRPCHandler) ListVariableExpenses(ctx context.Context,
	req *transactionv1.ListVariableExpensesRequest) (*transactionv1.ListVariableExpensesResponse, error) {
	userID := mustExtractUserID(ctx)
	filter := domain.VariableExpenseFilter{
		Limit:  int(req.PageSize),
		Offset: offsetFromToken(req.PageToken),
	}
	if req.StatusFilter != nil {
		s := domain.TransactionStatus(statusFromProto(*req.StatusFilter))
		filter.Status = &s
	}
	if req.DateFrom != nil {
		filter.DateFrom = req.DateFrom
	}
	if req.DateTo != nil {
		filter.DateTo = req.DateTo
	}
	if req.CategoryFilter != nil {
		filter.Category = req.CategoryFilter
	}

	items, total, err := h.svc.ListVariableExpenses(ctx, userID, filter)
	if err != nil {
		return nil, mapError(err)
	}

	protoItems := make([]*transactionv1.VariableExpense, len(items))
	for i, ve := range items {
		protoItems[i] = variableExpenseToProto(ve)
	}
	//nolint:gosec // G115: safe conversion
	return &transactionv1.ListVariableExpensesResponse{VariableExpenses: protoItems, TotalCount: int32(total)}, nil
}

// ── Proto enum → domain string ────────────────────────────────────────────

func statusFromProto(s transactionv1.TransactionStatus) string {
	switch s {
	case transactionv1.TransactionStatus_PENDING:
		return strPending
	case transactionv1.TransactionStatus_COMPLETED:
		return "completed"
	case transactionv1.TransactionStatus_CANCELLED:
		return "cancelled"
	default:
		return strPending
	}
}

func incomeTypeFromProto(t transactionv1.IncomeType) string {
	switch t {
	case transactionv1.IncomeType_SALARY:
		return "salary"
	case transactionv1.IncomeType_FREELANCE:
		return "freelance"
	case transactionv1.IncomeType_INVESTMENT:
		return "investment"
	case transactionv1.IncomeType_BUSINESS:
		return "business"
	case transactionv1.IncomeType_REFUND:
		return "refund"
	case transactionv1.IncomeType_INCOME_OTHER:
		return strOther
	default:
		return strOther
	}
}

func paymentMethodFromProto(pm transactionv1.PaymentMethod) string {
	switch pm {
	case transactionv1.PaymentMethod_CREDIT_CARD:
		return "credit_card"
	case transactionv1.PaymentMethod_DEBIT_CARD:
		return "debit_card"
	case transactionv1.PaymentMethod_CASH:
		return "cash"
	case transactionv1.PaymentMethod_BANK_TRANSFER:
		return "bank_transfer"
	case transactionv1.PaymentMethod_PIX:
		return "pix"
	case transactionv1.PaymentMethod_OTHER:
		return strOther
	default:
		return strOther
	}
}

func expenseTypeFromProto(et transactionv1.ExpenseType) string {
	switch et {
	case transactionv1.ExpenseType_ESSENTIAL:
		return "essential"
	case transactionv1.ExpenseType_DISCRETIONARY:
		return "discretionary"
	case transactionv1.ExpenseType_OCCASIONAL:
		return "occasional"
	case transactionv1.ExpenseType_EMERGENCY:
		return "emergency"
	case transactionv1.ExpenseType_EXPENSE_OTHER:
		return strOther
	default:
		return strOther
	}
}

// ── Domain string → Proto enum ────────────────────────────────────────────

func statusToProto(s string) transactionv1.TransactionStatus {
	switch s {
	case strPending:
		return transactionv1.TransactionStatus_PENDING
	case "completed":
		return transactionv1.TransactionStatus_COMPLETED
	case "cancelled":
		return transactionv1.TransactionStatus_CANCELLED
	default:
		return transactionv1.TransactionStatus_TRANSACTION_STATUS_UNSPECIFIED
	}
}

func incomeTypeToProto(t string) transactionv1.IncomeType {
	switch t {
	case "salary":
		return transactionv1.IncomeType_SALARY
	case "freelance":
		return transactionv1.IncomeType_FREELANCE
	case "investment":
		return transactionv1.IncomeType_INVESTMENT
	case "business":
		return transactionv1.IncomeType_BUSINESS
	case "refund":
		return transactionv1.IncomeType_REFUND
	case strOther:
		return transactionv1.IncomeType_INCOME_OTHER
	default:
		return transactionv1.IncomeType_INCOME_TYPE_UNSPECIFIED
	}
}

func paymentMethodToProto(pm string) transactionv1.PaymentMethod {
	switch pm {
	case "credit_card":
		return transactionv1.PaymentMethod_CREDIT_CARD
	case "debit_card":
		return transactionv1.PaymentMethod_DEBIT_CARD
	case "cash":
		return transactionv1.PaymentMethod_CASH
	case "bank_transfer":
		return transactionv1.PaymentMethod_BANK_TRANSFER
	case "pix":
		return transactionv1.PaymentMethod_PIX
	case strOther:
		return transactionv1.PaymentMethod_OTHER
	default:
		return transactionv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED
	}
}

func expenseTypeToProto(et string) transactionv1.ExpenseType {
	switch et {
	case "essential":
		return transactionv1.ExpenseType_ESSENTIAL
	case "discretionary":
		return transactionv1.ExpenseType_DISCRETIONARY
	case "occasional":
		return transactionv1.ExpenseType_OCCASIONAL
	case "emergency":
		return transactionv1.ExpenseType_EMERGENCY
	case strOther:
		return transactionv1.ExpenseType_EXPENSE_OTHER
	default:
		return transactionv1.ExpenseType_EXPENSE_TYPE_UNSPECIFIED
	}
}

// ── Response → Proto ──────────────────────────────────────────────────────

func incomeToProto(id, userID, description, source, incomeType, receivedDate string,
	receivedAmount int64, status string) *transactionv1.Income {
	return &transactionv1.Income{
		Id:             id,
		UserId:         userID,
		Description:    description,
		Source:         source,
		IncomeType:     incomeTypeToProto(incomeType),
		ReceivedDate:   receivedDate,
		ReceivedAmount: receivedAmount,
		Status:         statusToProto(status),
	}
}

func fixedExpenseToProto(resp *application.CreateFixedExpenseResponse) *transactionv1.FixedExpense {
	return &transactionv1.FixedExpense{
		Id:            resp.ID,
		UserId:        resp.UserID,
		Description:   resp.Description,
		Category:      resp.Category,
		DayOfMonth:    int32(resp.DayOfMonth), //nolint:gosec // G115: safe conversion, domain validates 1-31
		PaymentMethod: paymentMethodToProto(resp.PaymentMethod),
		Status:        statusToProto(resp.Status),
	}
}

func variableExpenseToProto(resp *application.CreateVariableExpenseResponse) *transactionv1.VariableExpense {
	return &transactionv1.VariableExpense{
		Id:            resp.ID,
		UserId:        resp.UserID,
		Description:   resp.Description,
		Destination:   resp.Destination,
		Category:      resp.Category,
		ExpenseType:   expenseTypeToProto(resp.ExpenseType),
		PaymentMethod: paymentMethodToProto(resp.PaymentMethod),
		PaymentDate:   resp.PaymentDate,
		PaidAmount:    resp.PaidAmount,
		Status:        statusToProto(resp.Status),
	}
}

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

// UserContext embeds a user ID into the context for downstream propagation.
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
	case errors.Is(err, domain.ErrInvalidDay):
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
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
