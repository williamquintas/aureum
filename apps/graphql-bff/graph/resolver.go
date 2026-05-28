package graph

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	identityv1 "github.com/aureum/proto/gen/identity/identityv1"
	transactionv1 "github.com/aureum/proto/gen/transaction/transactionv1"

	"github.com/aureum/graphql-bff/graph/model"
)

type Resolver struct {
	TxClient transactionv1.TransactionServiceClient
	IDClient identityv1.IdentityServiceClient
}

func NewResolver(txConn, idConn *grpc.ClientConn) *Resolver {
	return &Resolver{
		TxClient: transactionv1.NewTransactionServiceClient(txConn),
		IDClient: identityv1.NewIdentityServiceClient(idConn),
	}
}

func userIDFromCtx(ctx context.Context) string {
	uid, _ := ctx.Value("user_id").(string)
	return uid
}

// ── Income Resolvers ─────────────────────────────────────────────────────

func (r *queryResolver) Income(ctx context.Context, id string) (*model.Income, error) {
	pb, err := r.TxClient.GetIncome(ctx, &transactionv1.GetIncomeRequest{Id: id})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return incomeFromProto(pb), nil
}

func (r *queryResolver) Incomes(ctx context.Context, first *int, after *string, status *model.TransactionStatus, dateFrom *time.Time, dateTo *time.Time) (*model.IncomeConnection, error) {
	limit, offset := limitAndOffset(first, after)

	pb, err := r.TxClient.ListIncomes(ctx, &transactionv1.ListIncomesRequest{
		PageSize:     int32(limit),
		PageToken:    fmt.Sprintf("%d", offset),
		StatusFilter: statusToProto(status),
		DateFrom:     dateToStrPtr(dateFrom),
		DateTo:       dateToStrPtr(dateTo),
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	edges := make([]*model.IncomeEdge, len(pb.Incomes))
	for i, inc := range pb.Incomes {
		edges[i] = &model.IncomeEdge{
			Node:   incomeFromProto(inc),
			Cursor: fmt.Sprintf("%d", offset+i),
		}
	}
	hasNext := offset+limit < int(pb.TotalCount)
	return &model.IncomeConnection{
		Edges:      edges,
		TotalCount: int(pb.TotalCount),
		PageInfo: &model.PageInfo{
			HasNextPage:     hasNext,
			HasPreviousPage: offset > 0,
			StartCursor:     strPtr("0"),
			EndCursor:       strPtr(fmt.Sprintf("%d", offset+len(edges)-1)),
		},
	}, nil
}

// ── FixedExpense Resolvers ───────────────────────────────────────────────

func (r *queryResolver) FixedExpense(ctx context.Context, id string) (*model.FixedExpense, error) {
	pb, err := r.TxClient.GetFixedExpense(ctx, &transactionv1.GetFixedExpenseRequest{Id: id})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return fixedExpenseFromProto(pb), nil
}

func (r *queryResolver) FixedExpenses(ctx context.Context, first *int, after *string, status *model.TransactionStatus) (*model.FixedExpenseConnection, error) {
	limit, offset := limitAndOffset(first, after)

	pb, err := r.TxClient.ListFixedExpenses(ctx, &transactionv1.ListFixedExpensesRequest{
		PageSize:     int32(limit),
		PageToken:    fmt.Sprintf("%d", offset),
		StatusFilter: statusToProto(status),
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	edges := make([]*model.FixedExpenseEdge, len(pb.FixedExpenses))
	for i, fe := range pb.FixedExpenses {
		edges[i] = &model.FixedExpenseEdge{
			Node:   fixedExpenseFromProto(fe),
			Cursor: fmt.Sprintf("%d", offset+i),
		}
	}
	hasNext := offset+limit < int(pb.TotalCount)
	return &model.FixedExpenseConnection{
		Edges:      edges,
		TotalCount: int(pb.TotalCount),
		PageInfo: &model.PageInfo{
			HasNextPage:     hasNext,
			HasPreviousPage: offset > 0,
			StartCursor:     strPtr("0"),
			EndCursor:       strPtr(fmt.Sprintf("%d", offset+len(edges)-1)),
		},
	}, nil
}

// ── VariableExpense Resolvers ────────────────────────────────────────────

func (r *queryResolver) VariableExpense(ctx context.Context, id string) (*model.VariableExpense, error) {
	pb, err := r.TxClient.GetVariableExpense(ctx, &transactionv1.GetVariableExpenseRequest{Id: id})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return variableExpenseFromProto(pb), nil
}

func (r *queryResolver) VariableExpenses(ctx context.Context, first *int, after *string, status *model.TransactionStatus, dateFrom *time.Time, dateTo *time.Time, category *string) (*model.VariableExpenseConnection, error) {
	limit, offset := limitAndOffset(first, after)

	pb, err := r.TxClient.ListVariableExpenses(ctx, &transactionv1.ListVariableExpensesRequest{
		PageSize:     int32(limit),
		PageToken:    fmt.Sprintf("%d", offset),
		StatusFilter: statusToProto(status),
		DateFrom:     dateToStrPtr(dateFrom),
		DateTo:       dateToStrPtr(dateTo),
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	edges := make([]*model.VariableExpenseEdge, len(pb.VariableExpenses))
	for i, ve := range pb.VariableExpenses {
		edges[i] = &model.VariableExpenseEdge{
			Node:   variableExpenseFromProto(ve),
			Cursor: fmt.Sprintf("%d", offset+i),
		}
	}
	hasNext := offset+limit < int(pb.TotalCount)
	return &model.VariableExpenseConnection{
		Edges:      edges,
		TotalCount: int(pb.TotalCount),
		PageInfo: &model.PageInfo{
			HasNextPage:     hasNext,
			HasPreviousPage: offset > 0,
			StartCursor:     strPtr("0"),
			EndCursor:       strPtr(fmt.Sprintf("%d", offset+len(edges)-1)),
		},
	}, nil
}

// ── Unified Transactions ─────────────────────────────────────────────────

func (r *queryResolver) Transactions(ctx context.Context, first *int, after *string, typeArg *model.TransactionTypeFilter, dateFrom *time.Time, dateTo *time.Time) (*model.TransactionConnection, error) {
	limit, offset := limitAndOffset(first, after)

	var edges []*model.TransactionEdge
	totalCount := 0

	if typeArg == nil {
		incResp, incErr := r.TxClient.ListIncomes(ctx, &transactionv1.ListIncomesRequest{
			PageSize: int32(limit), PageToken: fmt.Sprintf("%d", offset),
			DateFrom: dateToStrPtr(dateFrom), DateTo: dateToStrPtr(dateTo),
		})
		if incErr != nil {
			return nil, mapGRPCError(incErr)
		}
		for i, inc := range incResp.Incomes {
			edges = append(edges, &model.TransactionEdge{Node: incomeFromProto(inc), Cursor: fmt.Sprintf("income-%d", i)})
		}
		totalCount += int(incResp.TotalCount)

		feResp, feErr := r.TxClient.ListFixedExpenses(ctx, &transactionv1.ListFixedExpensesRequest{
			PageSize: int32(limit), PageToken: fmt.Sprintf("%d", offset),
		})
		if feErr != nil {
			return nil, mapGRPCError(feErr)
		}
		for i, fe := range feResp.FixedExpenses {
			edges = append(edges, &model.TransactionEdge{Node: fixedExpenseFromProto(fe), Cursor: fmt.Sprintf("fixed-%d", i)})
		}
		totalCount += int(feResp.TotalCount)

		veResp, veErr := r.TxClient.ListVariableExpenses(ctx, &transactionv1.ListVariableExpensesRequest{
			PageSize: int32(limit), PageToken: fmt.Sprintf("%d", offset),
			DateFrom: dateToStrPtr(dateFrom), DateTo: dateToStrPtr(dateTo),
		})
		if veErr != nil {
			return nil, mapGRPCError(veErr)
		}
		for i, ve := range veResp.VariableExpenses {
			edges = append(edges, &model.TransactionEdge{Node: variableExpenseFromProto(ve), Cursor: fmt.Sprintf("variable-%d", i)})
		}
		totalCount += int(veResp.TotalCount)
	} else {
		switch *typeArg {
		case model.TransactionTypeFilterIncome:
			resp, err := r.TxClient.ListIncomes(ctx, &transactionv1.ListIncomesRequest{
				PageSize: int32(limit), PageToken: fmt.Sprintf("%d", offset),
				DateFrom: dateToStrPtr(dateFrom), DateTo: dateToStrPtr(dateTo),
			})
			if err != nil {
				return nil, mapGRPCError(err)
			}
			for i, inc := range resp.Incomes {
				edges = append(edges, &model.TransactionEdge{Node: incomeFromProto(inc), Cursor: fmt.Sprintf("income-%d", i)})
			}
			totalCount = int(resp.TotalCount)

		case model.TransactionTypeFilterFixedExpense:
			resp, err := r.TxClient.ListFixedExpenses(ctx, &transactionv1.ListFixedExpensesRequest{
				PageSize: int32(limit), PageToken: fmt.Sprintf("%d", offset),
			})
			if err != nil {
				return nil, mapGRPCError(err)
			}
			for i, fe := range resp.FixedExpenses {
				edges = append(edges, &model.TransactionEdge{Node: fixedExpenseFromProto(fe), Cursor: fmt.Sprintf("fixed-%d", i)})
			}
			totalCount = int(resp.TotalCount)

		case model.TransactionTypeFilterVariableExpense:
			resp, err := r.TxClient.ListVariableExpenses(ctx, &transactionv1.ListVariableExpensesRequest{
				PageSize: int32(limit), PageToken: fmt.Sprintf("%d", offset),
				DateFrom: dateToStrPtr(dateFrom), DateTo: dateToStrPtr(dateTo),
			})
			if err != nil {
				return nil, mapGRPCError(err)
			}
			for i, ve := range resp.VariableExpenses {
				edges = append(edges, &model.TransactionEdge{Node: variableExpenseFromProto(ve), Cursor: fmt.Sprintf("variable-%d", i)})
			}
			totalCount = int(resp.TotalCount)
		}
	}

	hasNext := offset+limit < totalCount
	return &model.TransactionConnection{
		Edges:      edges,
		TotalCount: totalCount,
		PageInfo: &model.PageInfo{
			HasNextPage:     hasNext,
			HasPreviousPage: offset > 0,
		},
	}, nil
}

// ── Me Resolver ──────────────────────────────────────────────────────────

func (r *queryResolver) Me(ctx context.Context) (*model.UserProfile, error) {
	userID := userIDFromCtx(ctx)
	if userID == "" {
		return nil, fmt.Errorf("user not authenticated")
	}

	pb, err := r.IDClient.GetUser(ctx, &identityv1.GetUserRequest{UserId: userID})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &model.UserProfile{ID: pb.UserId, Name: pb.Name, Email: pb.Email}, nil
}

// ── Proto → Model Converters ─────────────────────────────────────────────

func incomeFromProto(pb *transactionv1.Income) *model.Income {
	return &model.Income{
		ID:             pb.Id,
		UserID:         pb.UserId,
		Description:    pb.Description,
		Source:         pb.Source,
		IncomeType:     incomeTypeFromProto(pb.IncomeType),
		ReceivedDate:   parseDate(pb.ReceivedDate),
		ReceivedAmount: pb.ReceivedAmount,
		Status:         statusFromProto(pb.Status),
		CreatedAt:      pb.CreatedAt.AsTime(),
		UpdatedAt:      pb.UpdatedAt.AsTime(),
	}
}

func fixedExpenseFromProto(pb *transactionv1.FixedExpense) *model.FixedExpense {
	return &model.FixedExpense{
		ID:            pb.Id,
		UserID:        pb.UserId,
		Description:   pb.Description,
		Category:      pb.Category,
		DayOfMonth:    int(pb.DayOfMonth),
		PaymentMethod: paymentMethodFromProto(pb.PaymentMethod),
		Status:        statusFromProto(pb.Status),
		CreatedAt:     pb.CreatedAt.AsTime(),
		UpdatedAt:     pb.UpdatedAt.AsTime(),
	}
}

func variableExpenseFromProto(pb *transactionv1.VariableExpense) *model.VariableExpense {
	return &model.VariableExpense{
		ID:            pb.Id,
		UserID:        pb.UserId,
		Description:   pb.Description,
		Destination:   pb.Destination,
		Category:      pb.Category,
		ExpenseType:   expenseTypeFromProto(pb.ExpenseType),
		PaymentMethod: paymentMethodFromProto(pb.PaymentMethod),
		PaymentDate:   parseDate(pb.PaymentDate),
		PaidAmount:    pb.PaidAmount,
		Status:        statusFromProto(pb.Status),
		CreatedAt:     pb.CreatedAt.AsTime(),
		UpdatedAt:     pb.UpdatedAt.AsTime(),
	}
}

func statusFromProto(s transactionv1.TransactionStatus) model.TransactionStatus {
	switch s {
	case transactionv1.TransactionStatus_PENDING:
		return model.TransactionStatusPending
	case transactionv1.TransactionStatus_COMPLETED:
		return model.TransactionStatusCompleted
	case transactionv1.TransactionStatus_CANCELLED:
		return model.TransactionStatusCancelled
	default:
		return model.TransactionStatusPending
	}
}

func statusToProto(s *model.TransactionStatus) *transactionv1.TransactionStatus {
	if s == nil {
		return nil
	}
	switch *s {
	case model.TransactionStatusPending:
		return ptrOf(transactionv1.TransactionStatus_PENDING)
	case model.TransactionStatusCompleted:
		return ptrOf(transactionv1.TransactionStatus_COMPLETED)
	case model.TransactionStatusCancelled:
		return ptrOf(transactionv1.TransactionStatus_CANCELLED)
	default:
		return nil
	}
}

func incomeTypeFromProto(t transactionv1.IncomeType) model.IncomeType {
	switch t {
	case transactionv1.IncomeType_SALARY:
		return model.IncomeTypeSalary
	case transactionv1.IncomeType_FREELANCE:
		return model.IncomeTypeFreelance
	case transactionv1.IncomeType_INVESTMENT:
		return model.IncomeTypeInvestment
	case transactionv1.IncomeType_BUSINESS:
		return model.IncomeTypeBusiness
	case transactionv1.IncomeType_REFUND:
		return model.IncomeTypeRefund
	case transactionv1.IncomeType_INCOME_OTHER:
		return model.IncomeTypeOther
	default:
		return model.IncomeTypeOther
	}
}

func expenseTypeFromProto(t transactionv1.ExpenseType) model.ExpenseType {
	switch t {
	case transactionv1.ExpenseType_ESSENTIAL:
		return model.ExpenseTypeEssential
	case transactionv1.ExpenseType_DISCRETIONARY:
		return model.ExpenseTypeDiscretionary
	case transactionv1.ExpenseType_OCCASIONAL:
		return model.ExpenseTypeOccasional
	case transactionv1.ExpenseType_EMERGENCY:
		return model.ExpenseTypeEmergency
	case transactionv1.ExpenseType_EXPENSE_OTHER:
		return model.ExpenseTypeOther
	default:
		return model.ExpenseTypeOther
	}
}

func paymentMethodFromProto(pm transactionv1.PaymentMethod) model.PaymentMethod {
	switch pm {
	case transactionv1.PaymentMethod_CREDIT_CARD:
		return model.PaymentMethodCreditCard
	case transactionv1.PaymentMethod_DEBIT_CARD:
		return model.PaymentMethodDebitCard
	case transactionv1.PaymentMethod_CASH:
		return model.PaymentMethodCash
	case transactionv1.PaymentMethod_BANK_TRANSFER:
		return model.PaymentMethodBankTransfer
	case transactionv1.PaymentMethod_PIX:
		return model.PaymentMethodPix
	case transactionv1.PaymentMethod_OTHER:
		return model.PaymentMethodOther
	default:
		return model.PaymentMethodOther
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────

func limitAndOffset(first *int, after *string) (int, int) {
	limit := 20
	if first != nil && *first > 0 {
		limit = *first
	}
	offset := 0
	if after != nil && *after != "" {
		fmt.Sscanf(*after, "%d", &offset)
	}
	return limit, offset
}

func parseDate(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func dateToStrPtr(d *time.Time) *string {
	if d == nil {
		return nil
	}
	v := d.Format("2006-01-02")
	return &v
}

func strPtr(s string) *string {
	return &s
}

func ptrOf[T any](v T) *T {
	return &v
}

func mapGRPCError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	switch st.Code() {
	case codes.NotFound:
		return fmt.Errorf("not found: %s", st.Message())
	default:
		return fmt.Errorf(st.Message())
	}
}

func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
