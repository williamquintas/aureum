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

	"github.com/aureum/creditcard-svc/internal/application"
	"github.com/aureum/creditcard-svc/internal/domain"
	"github.com/aureum/pkg/telemetry"
	creditcardv1 "github.com/aureum/proto/gen/creditcard/creditcardv1"
)

type GRPCHandler struct {
	creditcardv1.UnimplementedCreditCardServiceServer
	svc application.CreditCardService
}

func NewGRPCHandler(svc application.CreditCardService) *GRPCHandler {
	return &GRPCHandler{svc: svc}
}

// ── CreditCard ───────────────────────────────────────────────────────────────

func (h *GRPCHandler) CreateCreditCard(ctx context.Context, req *creditcardv1.CreateCreditCardRequest) (*creditcardv1.CreditCard, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	resp, err := h.svc.CreateCreditCard(ctx, application.CreateCreditCardRequest{
		UserID:         userID,
		Name:           req.Name,
		Brand:          brandFromProto(req.Brand),
		CardType:       cardTypeFromProto(req.CardType),
		LastFourDigits: req.LastFourDigits,
		ClosingDay:     int(req.ClosingDay),
		DueDay:         int(req.DueDay),
		CreditLimit:    req.CreditLimit,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		telemetry.RecordRequest(ctx, "CreateCreditCard", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "CreateCreditCard", "ok", time.Since(start))
	return creditCardToProto(resp), nil
}

func (h *GRPCHandler) GetCreditCard(ctx context.Context, req *creditcardv1.GetCreditCardRequest) (*creditcardv1.CreditCard, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	resp, err := h.svc.GetCreditCard(ctx, req.Id, userID)
	if err != nil {
		telemetry.RecordRequest(ctx, "GetCreditCard", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "GetCreditCard", "ok", time.Since(start))
	return creditCardToProto(resp), nil
}

func (h *GRPCHandler) UpdateCreditCard(ctx context.Context, req *creditcardv1.UpdateCreditCardRequest) (*creditcardv1.CreditCard, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	appReq := application.UpdateCreditCardRequest{
		ID:             req.Id,
		UserID:         userID,
		IdempotencyKey: req.IdempotencyKey,
	}
	if req.Name != nil {
		appReq.Name = req.Name
	}
	if req.ClosingDay != nil {
		d := int(*req.ClosingDay)
		appReq.ClosingDay = &d
	}
	if req.DueDay != nil {
		d := int(*req.DueDay)
		appReq.DueDay = &d
	}
	if req.CreditLimit != nil {
		appReq.CreditLimit = req.CreditLimit
	}
	if req.Active != nil {
		appReq.Active = req.Active
	}
	resp, err := h.svc.UpdateCreditCard(ctx, appReq)
	if err != nil {
		telemetry.RecordRequest(ctx, "UpdateCreditCard", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "UpdateCreditCard", "ok", time.Since(start))
	return creditCardToProto(resp), nil
}

func (h *GRPCHandler) DeleteCreditCard(ctx context.Context, req *creditcardv1.DeleteCreditCardRequest) (*emptypb.Empty, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	if err := h.svc.DeleteCreditCard(ctx, req.Id, userID); err != nil {
		telemetry.RecordRequest(ctx, "DeleteCreditCard", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "DeleteCreditCard", "ok", time.Since(start))
	return &emptypb.Empty{}, nil
}

func (h *GRPCHandler) ListCreditCards(ctx context.Context, req *creditcardv1.ListCreditCardsRequest) (*creditcardv1.ListCreditCardsResponse, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	filter := domain.CreditCardFilter{
		Limit:  int(req.PageSize),
		Offset: offsetFromToken(req.PageToken),
	}
	if req.ActiveFilter != nil {
		filter.ActiveFilter = req.ActiveFilter
	}

	items, total, err := h.svc.ListCreditCards(ctx, userID, filter)
	if err != nil {
		telemetry.RecordRequest(ctx, "ListCreditCards", "error", time.Since(start))
		return nil, mapError(err)
	}

	protoItems := make([]*creditcardv1.CreditCard, len(items))
	for i, card := range items {
		protoItems[i] = creditCardToProto(card)
	}
	telemetry.RecordRequest(ctx, "ListCreditCards", "ok", time.Since(start))
	return &creditcardv1.ListCreditCardsResponse{
		CreditCards: protoItems,
		TotalCount:  int32(total),
	}, nil
}

// ── Invoice ──────────────────────────────────────────────────────────────────

func (h *GRPCHandler) CreateInvoice(ctx context.Context, req *creditcardv1.CreateInvoiceRequest) (*creditcardv1.Invoice, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	resp, err := h.svc.CreateInvoice(ctx, application.CreateInvoiceRequest{
		CreditCardID:   req.CreditCardId,
		UserID:         userID,
		ReferenceMonth: req.ReferenceMonth,
		ClosingDate:    req.ClosingDate,
		DueDate:        req.DueDate,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		telemetry.RecordRequest(ctx, "CreateInvoice", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "CreateInvoice", "ok", time.Since(start))
	return invoiceToProto(resp), nil
}

func (h *GRPCHandler) GetInvoice(ctx context.Context, req *creditcardv1.GetInvoiceRequest) (*creditcardv1.Invoice, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	resp, err := h.svc.GetInvoice(ctx, req.Id, userID)
	if err != nil {
		telemetry.RecordRequest(ctx, "GetInvoice", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "GetInvoice", "ok", time.Since(start))
	return invoiceToProto(resp), nil
}

func (h *GRPCHandler) ListInvoices(ctx context.Context, req *creditcardv1.ListInvoicesRequest) (*creditcardv1.ListInvoicesResponse, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	filter := domain.InvoiceFilter{
		Limit:  int(req.PageSize),
		Offset: offsetFromToken(req.PageToken),
	}
	if req.CreditCardId != "" {
		ccID := req.CreditCardId
		filter.CreditCardID = &ccID
	}
	if req.StatusFilter != nil {
		s := domain.InvoiceStatus(invoiceStatusFromProto(*req.StatusFilter))
		filter.StatusFilter = &s
	}
	if req.MonthFrom != nil {
		filter.MonthFrom = req.MonthFrom
	}
	if req.MonthTo != nil {
		filter.MonthTo = req.MonthTo
	}

	items, total, err := h.svc.ListInvoices(ctx, userID, filter)
	if err != nil {
		telemetry.RecordRequest(ctx, "ListInvoices", "error", time.Since(start))
		return nil, mapError(err)
	}

	protoItems := make([]*creditcardv1.Invoice, len(items))
	for i, inv := range items {
		protoItems[i] = invoiceToProto(inv)
	}
	telemetry.RecordRequest(ctx, "ListInvoices", "ok", time.Since(start))
	return &creditcardv1.ListInvoicesResponse{
		Invoices:   protoItems,
		TotalCount: int32(total),
	}, nil
}

func (h *GRPCHandler) PayInvoice(ctx context.Context, req *creditcardv1.PayInvoiceRequest) (*creditcardv1.Invoice, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	resp, err := h.svc.PayInvoice(ctx, application.PayInvoiceRequest{
		ID:             req.Id,
		UserID:         userID,
		Amount:         req.Amount,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		telemetry.RecordRequest(ctx, "PayInvoice", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "PayInvoice", "ok", time.Since(start))
	return invoiceToProto(resp), nil
}

// ── Transaction ──────────────────────────────────────────────────────────────

func (h *GRPCHandler) AddTransaction(ctx context.Context, req *creditcardv1.AddTransactionRequest) (*creditcardv1.InvoiceTransaction, error) {
	start := time.Now()

	userID := mustExtractUserID(ctx)
	resp, err := h.svc.AddTransaction(ctx, application.AddTransactionRequest{
		InvoiceID:       req.InvoiceId,
		UserID:          userID,
		Description:     req.Description,
		Amount:          req.Amount,
		Category:        req.Category,
		TransactionDate: req.TransactionDate,
		Installments:    req.Installments,
		IdempotencyKey:  req.IdempotencyKey,
	})
	if err != nil {
		telemetry.RecordRequest(ctx, "AddTransaction", "error", time.Since(start))
		return nil, mapError(err)
	}
	telemetry.RecordRequest(ctx, "AddTransaction", "ok", time.Since(start))
	return transactionToProto(resp), nil
}

func (h *GRPCHandler) ListTransactions(ctx context.Context, req *creditcardv1.ListTransactionsRequest) (*creditcardv1.ListTransactionsResponse, error) {
	start := time.Now()

	_ = mustExtractUserID(ctx) // user context is carried for tracing
	filter := domain.TransactionFilter{
		Limit:  int(req.PageSize),
		Offset: offsetFromToken(req.PageToken),
	}
	if req.CategoryFilter != nil {
		filter.CategoryFilter = req.CategoryFilter
	}

	items, total, err := h.svc.ListTransactions(ctx, req.InvoiceId, filter)
	if err != nil {
		telemetry.RecordRequest(ctx, "ListTransactions", "error", time.Since(start))
		return nil, mapError(err)
	}

	protoItems := make([]*creditcardv1.InvoiceTransaction, len(items))
	for i, t := range items {
		protoItems[i] = transactionToProto(t)
	}
	telemetry.RecordRequest(ctx, "ListTransactions", "ok", time.Since(start))
	return &creditcardv1.ListTransactionsResponse{
		Transactions: protoItems,
		TotalCount:   int32(total),
	}, nil
}

// ── Proto enum → domain string ────────────────────────────────────────────

func brandFromProto(b creditcardv1.CardBrand) string {
	switch b {
	case creditcardv1.CardBrand_VISA:
		return "visa"
	case creditcardv1.CardBrand_MASTERCARD:
		return "mastercard"
	case creditcardv1.CardBrand_AMEX:
		return "amex"
	case creditcardv1.CardBrand_ELO:
		return "elo"
	case creditcardv1.CardBrand_HIPERCARD:
		return "hipercard"
	case creditcardv1.CardBrand_DINERS:
		return "diners"
	case creditcardv1.CardBrand_OTHER_BRAND:
		return "other"
	default:
		return "other"
	}
}

func cardTypeFromProto(t creditcardv1.CardType) string {
	switch t {
	case creditcardv1.CardType_CREDIT:
		return "credit"
	case creditcardv1.CardType_DEBIT:
		return "debit"
	case creditcardv1.CardType_MULTIPLE:
		return "multiple"
	default:
		return "credit"
	}
}

func invoiceStatusFromProto(s creditcardv1.InvoiceStatus) string {
	switch s {
	case creditcardv1.InvoiceStatus_OPEN:
		return "open"
	case creditcardv1.InvoiceStatus_CLOSED:
		return "closed"
	case creditcardv1.InvoiceStatus_PAID:
		return "paid"
	case creditcardv1.InvoiceStatus_OVERDUE:
		return "overdue"
	default:
		return "open"
	}
}

// ── Domain string → Proto enum ────────────────────────────────────────────

func brandToProto(b string) creditcardv1.CardBrand {
	switch b {
	case "visa":
		return creditcardv1.CardBrand_VISA
	case "mastercard":
		return creditcardv1.CardBrand_MASTERCARD
	case "amex":
		return creditcardv1.CardBrand_AMEX
	case "elo":
		return creditcardv1.CardBrand_ELO
	case "hipercard":
		return creditcardv1.CardBrand_HIPERCARD
	case "diners":
		return creditcardv1.CardBrand_DINERS
	case "other":
		return creditcardv1.CardBrand_OTHER_BRAND
	default:
		return creditcardv1.CardBrand_CARD_BRAND_UNSPECIFIED
	}
}

func cardTypeToProto(t string) creditcardv1.CardType {
	switch t {
	case "credit":
		return creditcardv1.CardType_CREDIT
	case "debit":
		return creditcardv1.CardType_DEBIT
	case "multiple":
		return creditcardv1.CardType_MULTIPLE
	default:
		return creditcardv1.CardType_CARD_TYPE_UNSPECIFIED
	}
}

func invoiceStatusToProto(s string) creditcardv1.InvoiceStatus {
	switch s {
	case "open":
		return creditcardv1.InvoiceStatus_OPEN
	case "closed":
		return creditcardv1.InvoiceStatus_CLOSED
	case "paid":
		return creditcardv1.InvoiceStatus_PAID
	case "overdue":
		return creditcardv1.InvoiceStatus_OVERDUE
	default:
		return creditcardv1.InvoiceStatus_INVOICE_STATUS_UNSPECIFIED
	}
}

// ── Response → Proto ──────────────────────────────────────────────────────

func creditCardToProto(resp *application.CreditCardResponse) *creditcardv1.CreditCard {
	cc := &creditcardv1.CreditCard{
		Id:              resp.ID,
		UserId:          resp.UserID,
		Name:            resp.Name,
		Brand:           brandToProto(resp.Brand),
		CardType:        cardTypeToProto(resp.CardType),
		LastFourDigits:  resp.LastFourDigits,
		ClosingDay:      resp.ClosingDay,
		DueDay:          resp.DueDay,
		CreditLimit:     resp.CreditLimit,
		AvailableCredit: resp.AvailableCredit,
		Active:          resp.Active,
		CreatedAt:       timestamppb.New(time.Unix(resp.CreatedAt, 0)),
		UpdatedAt:       timestamppb.New(time.Unix(resp.UpdatedAt, 0)),
	}
	return cc
}

func invoiceToProto(resp *application.InvoiceResponse) *creditcardv1.Invoice {
	inv := &creditcardv1.Invoice{
		Id:             resp.ID,
		CreditCardId:   resp.CreditCardID,
		UserId:         resp.UserID,
		ReferenceMonth: resp.ReferenceMonth,
		TotalAmount:    resp.TotalAmount,
		PaidAmount:     resp.PaidAmount,
		Status:         invoiceStatusToProto(resp.Status),
		ClosingDate:    resp.ClosingDate,
		DueDate:        resp.DueDate,
		CreatedAt:      timestamppb.New(time.Unix(resp.CreatedAt, 0)),
		UpdatedAt:      timestamppb.New(time.Unix(resp.UpdatedAt, 0)),
	}
	return inv
}

func transactionToProto(resp *application.TransactionResponse) *creditcardv1.InvoiceTransaction {
	t := &creditcardv1.InvoiceTransaction{
		Id:              resp.ID,
		InvoiceId:       resp.InvoiceID,
		UserId:          resp.UserID,
		Description:     resp.Description,
		Amount:          resp.Amount,
		Category:        resp.Category,
		TransactionDate: resp.TransactionDate,
		Installments:    resp.Installments,
		CreatedAt:       timestamppb.New(time.Unix(resp.CreatedAt, 0)),
	}
	return t
}

// ── Utilities ─────────────────────────────────────────────────────────────

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

func UserContext(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrNegativeAmount):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidDay):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidCardBrand):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidCardType):
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
	case errors.Is(err, domain.ErrCreditExceeded):
		return status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, domain.ErrInvalidMonth):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidInvoiceStatus):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvoiceNotOpen):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrInvoiceAlreadyPaid):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrPaymentExceedsAmount):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrValidation):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
