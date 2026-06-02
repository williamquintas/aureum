package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	budgetv1 "github.com/aureum/proto/gen/budget/budgetv1"
	creditcardv1 "github.com/aureum/proto/gen/creditcard/creditcardv1"
	debtv1 "github.com/aureum/proto/gen/debt/debtv1"
	identityv1 "github.com/aureum/proto/gen/identity/identityv1"
	investmentv1 "github.com/aureum/proto/gen/investment/investmentv1"
	transactionv1 "github.com/aureum/proto/gen/transaction/transactionv1"

	"github.com/aureum/graphql-bff/graph/model"
	"github.com/aureum/graphql-bff/internal/infrastructure/cache"
	"github.com/aureum/graphql-bff/internal/infrastructure/clients"
	"github.com/aureum/graphql-bff/internal/infrastructure/featureflag"
)

type Resolver struct {
	TxClient  *clients.TransactionServiceClient
	IDClient  *clients.IdentityServiceClient
	BgtClient *clients.BudgetServiceClient
	CCCClient *clients.CreditCardServiceClient
	DbtClient *clients.DebtServiceClient
	InvClient *clients.InvestmentServiceClient
	Cache     *cache.Cache
	FFClient  *featureflag.Client
}

func NewResolver(txClient *clients.TransactionServiceClient, idClient *clients.IdentityServiceClient, bgtClient *clients.BudgetServiceClient, cccClient *clients.CreditCardServiceClient, dbtClient *clients.DebtServiceClient, invClient *clients.InvestmentServiceClient, cacheStore *cache.Cache, ffClient *featureflag.Client) *Resolver {
	return &Resolver{
		TxClient:  txClient,
		IDClient:  idClient,
		BgtClient: bgtClient,
		CCCClient: cccClient,
		DbtClient: dbtClient,
		InvClient: invClient,
		Cache:     cacheStore,
		FFClient:  ffClient,
	}
}

func userIDFromCtx(ctx context.Context) string {
	uid, _ := ctx.Value("user_id").(string)
	return uid
}

func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }

func (r *queryResolver) Income(ctx context.Context, id string) (*model.Income, error) {
	var out model.Income
	err := r.cachedSingle(ctx, "income", id, &out, func() (interface{}, error) {
		pb, err := r.TxClient.GetIncome(ctx, &transactionv1.GetIncomeRequest{Id: id})
		if err != nil {
			return nil, err
		}
		return incomeFromProto(pb), nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) Incomes(ctx context.Context, first *int, after *string, status *model.TransactionStatus, dateFrom *time.Time, dateTo *time.Time) (*model.IncomeConnection, error) {
	var out model.IncomeConnection
	err := r.cachedList(ctx, "incomes", struct {
		First    *int
		After    *string
		Status   *model.TransactionStatus
		DateFrom *time.Time
		DateTo   *time.Time
	}{first, after, status, dateFrom, dateTo}, &out, func() (interface{}, error) {
		limit, offset := limitAndOffset(first, after)

		pb, err := r.TxClient.ListIncomes(ctx, &transactionv1.ListIncomesRequest{
			PageSize:     int32(limit),
			PageToken:    fmt.Sprintf("%d", offset),
			StatusFilter: statusToProto(status),
			DateFrom:     dateToStrPtr(dateFrom),
			DateTo:       dateToStrPtr(dateTo),
		})
		if err != nil {
			return nil, err
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
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) FixedExpense(ctx context.Context, id string) (*model.FixedExpense, error) {
	var out model.FixedExpense
	err := r.cachedSingle(ctx, "fixed_expense", id, &out, func() (interface{}, error) {
		pb, err := r.TxClient.GetFixedExpense(ctx, &transactionv1.GetFixedExpenseRequest{Id: id})
		if err != nil {
			return nil, err
		}
		return fixedExpenseFromProto(pb), nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) FixedExpenses(ctx context.Context, first *int, after *string, status *model.TransactionStatus) (*model.FixedExpenseConnection, error) {
	var out model.FixedExpenseConnection
	err := r.cachedList(ctx, "fixed_expenses", struct {
		First  *int
		After  *string
		Status *model.TransactionStatus
	}{first, after, status}, &out, func() (interface{}, error) {
		limit, offset := limitAndOffset(first, after)

		pb, err := r.TxClient.ListFixedExpenses(ctx, &transactionv1.ListFixedExpensesRequest{
			PageSize:     int32(limit),
			PageToken:    fmt.Sprintf("%d", offset),
			StatusFilter: statusToProto(status),
		})
		if err != nil {
			return nil, err
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
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) VariableExpense(ctx context.Context, id string) (*model.VariableExpense, error) {
	var out model.VariableExpense
	err := r.cachedSingle(ctx, "variable_expense", id, &out, func() (interface{}, error) {
		pb, err := r.TxClient.GetVariableExpense(ctx, &transactionv1.GetVariableExpenseRequest{Id: id})
		if err != nil {
			return nil, err
		}
		return variableExpenseFromProto(pb), nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) VariableExpenses(ctx context.Context, first *int, after *string, status *model.TransactionStatus, dateFrom *time.Time, dateTo *time.Time, category *string) (*model.VariableExpenseConnection, error) {
	var out model.VariableExpenseConnection
	err := r.cachedList(ctx, "variable_expenses", struct {
		First    *int
		After    *string
		Status   *model.TransactionStatus
		DateFrom *time.Time
		DateTo   *time.Time
		Category *string
	}{first, after, status, dateFrom, dateTo, category}, &out, func() (interface{}, error) {
		limit, offset := limitAndOffset(first, after)

		pb, err := r.TxClient.ListVariableExpenses(ctx, &transactionv1.ListVariableExpensesRequest{
			PageSize:       int32(limit),
			PageToken:      fmt.Sprintf("%d", offset),
			StatusFilter:   statusToProto(status),
			DateFrom:       dateToStrPtr(dateFrom),
			DateTo:         dateToStrPtr(dateTo),
			CategoryFilter: category,
		})
		if err != nil {
			return nil, err
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
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

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

func (r *queryResolver) Me(ctx context.Context) (*model.UserProfile, error) {
	userID := userIDFromCtx(ctx)
	if userID == "" {
		return nil, fmt.Errorf("user not authenticated")
	}

	var profile model.UserProfile
	err := r.cachedSingle(ctx, "user", userID, &profile, func() (interface{}, error) {
		pb, err := r.IDClient.GetUser(ctx, &identityv1.GetUserRequest{UserId: userID})
		if err != nil {
			return nil, err
		}
		return &model.UserProfile{ID: pb.UserId, Name: pb.Name, Email: pb.Email}, nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &profile, nil
}

func (r *queryResolver) Budget(ctx context.Context, id string) (*model.Budget, error) {
	var out model.Budget
	err := r.cachedSingle(ctx, "budget", id, &out, func() (interface{}, error) {
		pb, err := r.BgtClient.GetBudget(ctx, &budgetv1.GetBudgetRequest{Id: id})
		if err != nil {
			return nil, err
		}
		return budgetFromProto(pb), nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) BudgetSummary(ctx context.Context, id string) (*model.BudgetSummary, error) {
	var out model.BudgetSummary
	err := r.cachedSingle(ctx, "budget_summary", id, &out, func() (interface{}, error) {
		pb, err := r.BgtClient.GetBudgetSummary(ctx, &budgetv1.GetBudgetSummaryRequest{Id: id})
		if err != nil {
			return nil, err
		}
		return budgetSummaryFromProto(pb), nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) Budgets(ctx context.Context, first *int, after *string, status *model.BudgetStatus, dateFrom *time.Time, dateTo *time.Time) (*model.BudgetConnection, error) {
	var out model.BudgetConnection
	err := r.cachedList(ctx, "budgets", struct {
		First    *int
		After    *string
		Status   *model.BudgetStatus
		DateFrom *time.Time
		DateTo   *time.Time
	}{first, after, status, dateFrom, dateTo}, &out, func() (interface{}, error) {
		limit, offset := limitAndOffset(first, after)

		req := &budgetv1.ListBudgetsRequest{
			PageSize:  int32(limit),
			PageToken: fmt.Sprintf("%d", offset),
		}
		if status != nil {
			req.StatusFilter = budgetStatusToProtoPtr(status)
		}
		if dateFrom != nil {
			req.DateFrom = dateToStrPtr(dateFrom)
		}
		if dateTo != nil {
			req.DateTo = dateToStrPtr(dateTo)
		}

		pb, err := r.BgtClient.ListBudgets(ctx, req)
		if err != nil {
			return nil, err
		}

		edges := make([]*model.BudgetEdge, len(pb.Budgets))
		for i, b := range pb.Budgets {
			edges[i] = &model.BudgetEdge{
				Node:   budgetFromProto(b),
				Cursor: fmt.Sprintf("%d", offset+i),
			}
		}
		hasNext := offset+limit < int(pb.TotalCount)
		return &model.BudgetConnection{
			Edges:      edges,
			TotalCount: int(pb.TotalCount),
			PageInfo: &model.PageInfo{
				HasNextPage:     hasNext,
				HasPreviousPage: offset > 0,
				StartCursor:     strPtr("0"),
				EndCursor:       strPtr(fmt.Sprintf("%d", offset+len(edges)-1)),
			},
		}, nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) CreditCard(ctx context.Context, id string) (*model.CreditCard, error) {
	var out model.CreditCard
	err := r.cachedSingle(ctx, "credit_card", id, &out, func() (interface{}, error) {
		pb, err := r.CCCClient.GetCreditCard(ctx, &creditcardv1.GetCreditCardRequest{Id: id})
		if err != nil {
			return nil, err
		}
		return creditCardFromProto(pb), nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) CreditCards(ctx context.Context, first *int, after *string, active *bool) (*model.CreditCardConnection, error) {
	var out model.CreditCardConnection
	err := r.cachedList(ctx, "credit_cards", struct {
		First  *int
		After  *string
		Active *bool
	}{first, after, active}, &out, func() (interface{}, error) {
		limit, offset := limitAndOffset(first, after)

		req := &creditcardv1.ListCreditCardsRequest{
			PageSize:  int32(limit),
			PageToken: fmt.Sprintf("%d", offset),
		}
		if active != nil {
			req.ActiveFilter = active
		}

		pb, err := r.CCCClient.ListCreditCards(ctx, req)
		if err != nil {
			return nil, err
		}

		edges := make([]*model.CreditCardEdge, len(pb.CreditCards))
		for i, cc := range pb.CreditCards {
			edges[i] = &model.CreditCardEdge{
				Node:   creditCardFromProto(cc),
				Cursor: fmt.Sprintf("%d", offset+i),
			}
		}
		hasNext := offset+limit < int(pb.TotalCount)
		return &model.CreditCardConnection{
			Edges:      edges,
			TotalCount: int(pb.TotalCount),
			PageInfo: &model.PageInfo{
				HasNextPage:     hasNext,
				HasPreviousPage: offset > 0,
				StartCursor:     strPtr("0"),
				EndCursor:       strPtr(fmt.Sprintf("%d", offset+len(edges)-1)),
			},
		}, nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) Invoice(ctx context.Context, id string) (*model.Invoice, error) {
	var out model.Invoice
	err := r.cachedSingle(ctx, "invoice", id, &out, func() (interface{}, error) {
		pb, err := r.CCCClient.GetInvoice(ctx, &creditcardv1.GetInvoiceRequest{Id: id})
		if err != nil {
			return nil, err
		}
		return invoiceFromProto(pb), nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) Invoices(ctx context.Context, first *int, after *string, creditCardID string, status *model.InvoiceStatus, monthFrom *string, monthTo *string) (*model.InvoiceConnection, error) {
	var out model.InvoiceConnection
	err := r.cachedList(ctx, "invoices", struct {
		First        *int
		After        *string
		CreditCardID string
		Status       *model.InvoiceStatus
		MonthFrom    *string
		MonthTo      *string
	}{first, after, creditCardID, status, monthFrom, monthTo}, &out, func() (interface{}, error) {
		limit, offset := limitAndOffset(first, after)

		req := &creditcardv1.ListInvoicesRequest{
			PageSize:     int32(limit),
			PageToken:    fmt.Sprintf("%d", offset),
			CreditCardId: creditCardID,
			MonthFrom:    monthFrom,
			MonthTo:      monthTo,
		}
		if status != nil {
			req.StatusFilter = invoiceStatusToProtoPtr(status)
		}

		pb, err := r.CCCClient.ListInvoices(ctx, req)
		if err != nil {
			return nil, err
		}

		edges := make([]*model.InvoiceEdge, len(pb.Invoices))
		for i, inv := range pb.Invoices {
			edges[i] = &model.InvoiceEdge{
				Node:   invoiceFromProto(inv),
				Cursor: fmt.Sprintf("%d", offset+i),
			}
		}
		hasNext := offset+limit < int(pb.TotalCount)
		return &model.InvoiceConnection{
			Edges:      edges,
			TotalCount: int(pb.TotalCount),
			PageInfo: &model.PageInfo{
				HasNextPage:     hasNext,
				HasPreviousPage: offset > 0,
				StartCursor:     strPtr("0"),
				EndCursor:       strPtr(fmt.Sprintf("%d", offset+len(edges)-1)),
			},
		}, nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) InvoiceTransactions(ctx context.Context, first *int, after *string, invoiceID string, category *string) (*model.InvoiceTransactionConnection, error) {
	var out model.InvoiceTransactionConnection
	err := r.cachedList(ctx, "invoice_transactions", struct {
		First     *int
		After     *string
		InvoiceID string
		Category  *string
	}{first, after, invoiceID, category}, &out, func() (interface{}, error) {
		limit, offset := limitAndOffset(first, after)

		req := &creditcardv1.ListTransactionsRequest{
			PageSize:       int32(limit),
			PageToken:      fmt.Sprintf("%d", offset),
			InvoiceId:      invoiceID,
			CategoryFilter: category,
		}

		pb, err := r.CCCClient.ListTransactions(ctx, req)
		if err != nil {
			return nil, err
		}

		edges := make([]*model.InvoiceTransactionEdge, len(pb.Transactions))
		for i, t := range pb.Transactions {
			edges[i] = &model.InvoiceTransactionEdge{
				Node:   invoiceTransactionFromProto(t),
				Cursor: fmt.Sprintf("%d", offset+i),
			}
		}
		hasNext := offset+limit < int(pb.TotalCount)
		return &model.InvoiceTransactionConnection{
			Edges:      edges,
			TotalCount: int(pb.TotalCount),
			PageInfo: &model.PageInfo{
				HasNextPage:     hasNext,
				HasPreviousPage: offset > 0,
				StartCursor:     strPtr("0"),
				EndCursor:       strPtr(fmt.Sprintf("%d", offset+len(edges)-1)),
			},
		}, nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) Debt(ctx context.Context, id string) (*model.Debt, error) {
	var out model.Debt
	err := r.cachedSingle(ctx, "debt", id, &out, func() (interface{}, error) {
		pb, err := r.DbtClient.GetDebt(ctx, &debtv1.GetDebtRequest{Id: id})
		if err != nil {
			return nil, err
		}
		return debtFromProto(pb), nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) Debts(ctx context.Context, first *int, after *string, status *model.DebtStatus, typeArg *model.DebtType) (*model.DebtConnection, error) {
	var out model.DebtConnection
	err := r.cachedList(ctx, "debts", struct {
		First   *int
		After   *string
		Status  *model.DebtStatus
		TypeArg *model.DebtType
	}{first, after, status, typeArg}, &out, func() (interface{}, error) {
		limit, offset := limitAndOffset(first, after)

		req := &debtv1.ListDebtsRequest{
			PageSize:  int32(limit),
			PageToken: fmt.Sprintf("%d", offset),
		}
		if status != nil {
			req.StatusFilter = debtStatusToProtoPtr(status)
		}
		if typeArg != nil {
			req.TypeFilter = debtTypeToProtoPtr(typeArg)
		}

		pb, err := r.DbtClient.ListDebts(ctx, req)
		if err != nil {
			return nil, err
		}

		edges := make([]*model.DebtEdge, len(pb.Debts))
		for i, d := range pb.Debts {
			edges[i] = &model.DebtEdge{
				Node:   debtFromProto(d),
				Cursor: fmt.Sprintf("%d", offset+i),
			}
		}
		hasNext := offset+limit < int(pb.TotalCount)
		return &model.DebtConnection{
			Edges:      edges,
			TotalCount: int(pb.TotalCount),
			PageInfo: &model.PageInfo{
				HasNextPage:     hasNext,
				HasPreviousPage: offset > 0,
				StartCursor:     strPtr("0"),
				EndCursor:       strPtr(fmt.Sprintf("%d", offset+len(edges)-1)),
			},
		}, nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) Payments(ctx context.Context, first *int, after *string, debtID string, dateFrom *time.Time, dateTo *time.Time) (*model.PaymentConnection, error) {
	var out model.PaymentConnection
	err := r.cachedList(ctx, "payments", struct {
		First    *int
		After    *string
		DebtID   string
		DateFrom *time.Time
		DateTo   *time.Time
	}{first, after, debtID, dateFrom, dateTo}, &out, func() (interface{}, error) {
		limit, offset := limitAndOffset(first, after)

		pb, err := r.DbtClient.ListPayments(ctx, &debtv1.ListPaymentsRequest{
			PageSize:  int32(limit),
			PageToken: fmt.Sprintf("%d", offset),
			DebtId:    debtID,
			DateFrom:  dateToStrPtr(dateFrom),
			DateTo:    dateToStrPtr(dateTo),
		})
		if err != nil {
			return nil, err
		}

		edges := make([]*model.PaymentEdge, len(pb.Payments))
		for i, p := range pb.Payments {
			edges[i] = &model.PaymentEdge{
				Node:   paymentFromProto(p),
				Cursor: fmt.Sprintf("%d", offset+i),
			}
		}
		hasNext := offset+limit < int(pb.TotalCount)
		return &model.PaymentConnection{
			Edges:      edges,
			TotalCount: int(pb.TotalCount),
			PageInfo: &model.PageInfo{
				HasNextPage:     hasNext,
				HasPreviousPage: offset > 0,
				StartCursor:     strPtr("0"),
				EndCursor:       strPtr(fmt.Sprintf("%d", offset+len(edges)-1)),
			},
		}, nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) Investment(ctx context.Context, id string) (*model.Investment, error) {
	var out model.Investment
	err := r.cachedSingle(ctx, "investment", id, &out, func() (interface{}, error) {
		pb, err := r.InvClient.GetInvestment(ctx, &investmentv1.GetInvestmentRequest{Id: id})
		if err != nil {
			return nil, err
		}
		return investmentFromProto(pb), nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) Investments(ctx context.Context, first *int, after *string, assetType *model.AssetType, status *model.InvestmentStatus) (*model.InvestmentConnection, error) {
	var out model.InvestmentConnection
	err := r.cachedList(ctx, "investments", struct {
		First     *int
		After     *string
		AssetType *model.AssetType
		Status    *model.InvestmentStatus
	}{first, after, assetType, status}, &out, func() (interface{}, error) {
		limit, offset := limitAndOffset(first, after)

		req := &investmentv1.ListInvestmentsRequest{
			PageSize:  int32(limit),
			PageToken: fmt.Sprintf("%d", offset),
		}
		if assetType != nil {
			req.TypeFilter = assetTypeToProtoPtr(assetType)
		}
		if status != nil {
			req.StatusFilter = investmentStatusToProtoPtr(status)
		}

		pb, err := r.InvClient.ListInvestments(ctx, req)
		if err != nil {
			return nil, err
		}

		edges := make([]*model.InvestmentEdge, len(pb.Investments))
		for i, inv := range pb.Investments {
			edges[i] = &model.InvestmentEdge{
				Node:   investmentFromProto(inv),
				Cursor: fmt.Sprintf("%d", offset+i),
			}
		}
		hasNext := offset+limit < int(pb.TotalCount)
		return &model.InvestmentConnection{
			Edges:      edges,
			TotalCount: int(pb.TotalCount),
			PageInfo: &model.PageInfo{
				HasNextPage:     hasNext,
				HasPreviousPage: offset > 0,
				StartCursor:     strPtr("0"),
				EndCursor:       strPtr(fmt.Sprintf("%d", offset+len(edges)-1)),
			},
		}, nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) InvestmentTransactions(ctx context.Context, first *int, after *string, investmentID string, typeArg *model.InvestmentTransactionType, dateFrom *time.Time, dateTo *time.Time) (*model.InvestmentTransactionConnection, error) {
	var out model.InvestmentTransactionConnection
	err := r.cachedList(ctx, "investment_transactions", struct {
		First        *int
		After        *string
		InvestmentID string
		TypeArg      *model.InvestmentTransactionType
		DateFrom     *time.Time
		DateTo       *time.Time
	}{first, after, investmentID, typeArg, dateFrom, dateTo}, &out, func() (interface{}, error) {
		limit, offset := limitAndOffset(first, after)

		req := &investmentv1.ListTransactionsRequest{
			PageSize:     int32(limit),
			PageToken:    fmt.Sprintf("%d", offset),
			InvestmentId: investmentID,
			DateFrom:     dateToStrPtr(dateFrom),
			DateTo:       dateToStrPtr(dateTo),
		}
		if typeArg != nil {
			req.TypeFilter = transactionTypeToProtoPtr(typeArg)
		}

		pb, err := r.InvClient.ListTransactions(ctx, req)
		if err != nil {
			return nil, err
		}

		edges := make([]*model.InvestmentTransactionEdge, len(pb.Transactions))
		for i, t := range pb.Transactions {
			edges[i] = &model.InvestmentTransactionEdge{
				Node:   investmentTransactionFromProto(t),
				Cursor: fmt.Sprintf("%d", offset+i),
			}
		}
		hasNext := offset+limit < int(pb.TotalCount)
		return &model.InvestmentTransactionConnection{
			Edges:      edges,
			TotalCount: int(pb.TotalCount),
			PageInfo: &model.PageInfo{
				HasNextPage:     hasNext,
				HasPreviousPage: offset > 0,
				StartCursor:     strPtr("0"),
				EndCursor:       strPtr(fmt.Sprintf("%d", offset+len(edges)-1)),
			},
		}, nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
}

func (r *queryResolver) PortfolioSummary(ctx context.Context) (*model.PortfolioSummary, error) {
	var out model.PortfolioSummary
	err := r.cachedSingle(ctx, "portfolio_summary", "default", &out, func() (interface{}, error) {
		pb, err := r.InvClient.GetPortfolioSummary(ctx, &investmentv1.GetPortfolioSummaryRequest{})
		if err != nil {
			return nil, err
		}
		return portfolioSummaryFromProto(pb), nil
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &out, nil
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

func budgetFromProto(pb *budgetv1.Budget) *model.Budget {
	cats := make([]*model.BudgetCategory, len(pb.Categories))
	for i, c := range pb.Categories {
		cats[i] = budgetCategoryFromProto(c)
	}
	return &model.Budget{
		ID:          pb.Id,
		UserID:      pb.UserId,
		Name:        pb.Name,
		Description: pb.Description,
		Period:      budgetPeriodFromProto(pb.Period),
		TotalLimit:  pb.TotalLimit,
		SpentAmount: pb.SpentAmount,
		Status:      budgetStatusFromProto(pb.Status),
		StartDate:   parseDate(pb.StartDate),
		EndDate:     parseDate(pb.EndDate),
		Categories:  cats,
		CreatedAt:   pb.CreatedAt.AsTime(),
		UpdatedAt:   pb.UpdatedAt.AsTime(),
	}
}

func budgetCategoryFromProto(pb *budgetv1.BudgetCategory) *model.BudgetCategory {
	return &model.BudgetCategory{
		ID:          pb.Id,
		BudgetID:    pb.BudgetId,
		Name:        pb.Name,
		LimitAmount: pb.LimitAmount,
		SpentAmount: pb.SpentAmount,
		Category:    pb.Category,
	}
}

func budgetSummaryFromProto(pb *budgetv1.BudgetSummary) *model.BudgetSummary {
	cats := make([]*model.CategorySummary, len(pb.Categories))
	for i, c := range pb.Categories {
		cats[i] = categorySummaryFromProto(c)
	}
	return &model.BudgetSummary{
		BudgetID:        pb.BudgetId,
		TotalLimit:      pb.TotalLimit,
		TotalSpent:      pb.TotalSpent,
		Remaining:       pb.Remaining,
		UsagePercentage: pb.UsagePercentage,
		CategoryCount:   int(pb.CategoryCount),
		Categories:      cats,
	}
}

func categorySummaryFromProto(pb *budgetv1.CategorySummary) *model.CategorySummary {
	return &model.CategorySummary{
		CategoryID:      pb.CategoryId,
		Name:            pb.Name,
		Category:        pb.Category,
		LimitAmount:     pb.LimitAmount,
		SpentAmount:     pb.SpentAmount,
		Remaining:       pb.Remaining,
		UsagePercentage: pb.UsagePercentage,
	}
}

func creditCardFromProto(pb *creditcardv1.CreditCard) *model.CreditCard {
	return &model.CreditCard{
		ID:              pb.Id,
		UserID:          pb.UserId,
		Name:            pb.Name,
		Brand:           cardBrandFromProto(pb.Brand),
		CardType:        cardTypeFromProto(pb.CardType),
		LastFourDigits:  pb.LastFourDigits,
		ClosingDay:      int(pb.ClosingDay),
		DueDay:          int(pb.DueDay),
		CreditLimit:     pb.CreditLimit,
		AvailableCredit: pb.AvailableCredit,
		Active:          pb.Active,
		CreatedAt:       pb.CreatedAt.AsTime(),
		UpdatedAt:       pb.UpdatedAt.AsTime(),
	}
}

func invoiceFromProto(pb *creditcardv1.Invoice) *model.Invoice {
	return &model.Invoice{
		ID:             pb.Id,
		CreditCardID:   pb.CreditCardId,
		UserID:         pb.UserId,
		ReferenceMonth: pb.ReferenceMonth,
		TotalAmount:    pb.TotalAmount,
		PaidAmount:     pb.PaidAmount,
		Status:         invoiceStatusFromProto(pb.Status),
		ClosingDate:    parseDate(pb.ClosingDate),
		DueDate:        parseDate(pb.DueDate),
		CreatedAt:      pb.CreatedAt.AsTime(),
		UpdatedAt:      pb.UpdatedAt.AsTime(),
	}
}

func invoiceTransactionFromProto(pb *creditcardv1.InvoiceTransaction) *model.InvoiceTransaction {
	return &model.InvoiceTransaction{
		ID:              pb.Id,
		InvoiceID:       pb.InvoiceId,
		UserID:          pb.UserId,
		Description:     pb.Description,
		Amount:          pb.Amount,
		Category:        pb.Category,
		TransactionDate: parseDate(pb.TransactionDate),
		Installments:    int(pb.Installments),
		CreatedAt:       pb.CreatedAt.AsTime(),
	}
}

func debtFromProto(pb *debtv1.Debt) *model.Debt {
	return &model.Debt{
		ID:              pb.Id,
		UserID:          pb.UserId,
		Name:            pb.Name,
		Description:     pb.Description,
		DebtType:        debtTypeFromProto(pb.DebtType),
		TotalAmount:     pb.TotalAmount,
		RemainingAmount: pb.RemainingAmount,
		InterestRate:    pb.InterestRate,
		StartDate:       parseDate(pb.StartDate),
		ExpectedEndDate: parseDate(pb.ExpectedEndDate),
		Status:          debtStatusFromProto(pb.Status),
		Creditor:        pb.Creditor,
		CreatedAt:       pb.CreatedAt.AsTime(),
		UpdatedAt:       pb.UpdatedAt.AsTime(),
	}
}

func paymentFromProto(pb *debtv1.Payment) *model.Payment {
	return &model.Payment{
		ID:          pb.Id,
		DebtID:      pb.DebtId,
		UserID:      pb.UserId,
		Amount:      pb.Amount,
		PaymentDate: parseDate(pb.PaymentDate),
		Notes:       pb.Notes,
		CreatedAt:   pb.CreatedAt.AsTime(),
	}
}

func investmentFromProto(pb *investmentv1.Investment) *model.Investment {
	return &model.Investment{
		ID:            pb.Id,
		UserID:        pb.UserId,
		Name:          pb.Name,
		Ticker:        pb.Ticker,
		AssetType:     assetTypeFromProto(pb.AssetType),
		Quantity:      pb.Quantity,
		AveragePrice:  pb.AveragePrice,
		TotalInvested: pb.TotalInvested,
		Status:        investmentStatusFromProto(pb.Status),
		Broker:        pb.Broker,
		CreatedAt:     pb.CreatedAt.AsTime(),
		UpdatedAt:     pb.UpdatedAt.AsTime(),
	}
}

func investmentTransactionFromProto(pb *investmentv1.InvestmentTransaction) *model.InvestmentTransaction {
	return &model.InvestmentTransaction{
		ID:              pb.Id,
		InvestmentID:    pb.InvestmentId,
		UserID:          pb.UserId,
		TransactionType: transactionTypeFromProto(pb.TransactionType),
		Quantity:        pb.Quantity,
		UnitPrice:       pb.UnitPrice,
		TotalAmount:     pb.TotalAmount,
		TransactionDate: parseDate(pb.TransactionDate),
		Notes:           pb.Notes,
		CreatedAt:       pb.CreatedAt.AsTime(),
	}
}

func portfolioSummaryFromProto(pb *investmentv1.PortfolioSummary) *model.PortfolioSummary {
	allocs := make([]*model.AssetAllocation, len(pb.Allocation))
	for i, a := range pb.Allocation {
		allocs[i] = assetAllocationFromProto(a)
	}
	return &model.PortfolioSummary{
		TotalInvested:     pb.TotalInvested,
		CurrentValue:      pb.CurrentValue,
		TotalReturn:       pb.TotalReturn,
		ReturnPercentage:  pb.ReturnPercentage,
		ActiveInvestments: int(pb.ActiveInvestments),
		Allocation:        allocs,
	}
}

func assetAllocationFromProto(pb *investmentv1.AssetAllocation) *model.AssetAllocation {
	return &model.AssetAllocation{
		AssetType:    assetTypeFromProto(pb.AssetType),
		Invested:     pb.Invested,
		CurrentValue: pb.CurrentValue,
		Percentage:   pb.Percentage,
	}
}

// ── Transaction Enum Converters ──────────────────────────────────────────

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

// ── Budget Enum Converters ───────────────────────────────────────────────

func budgetPeriodFromProto(p budgetv1.BudgetPeriod) model.BudgetPeriod {
	switch p {
	case budgetv1.BudgetPeriod_MONTHLY:
		return model.BudgetPeriodMonthly
	case budgetv1.BudgetPeriod_BIMONTHLY:
		return model.BudgetPeriodBimonthly
	case budgetv1.BudgetPeriod_QUARTERLY:
		return model.BudgetPeriodQuarterly
	case budgetv1.BudgetPeriod_SEMESTRAL:
		return model.BudgetPeriodSemestral
	case budgetv1.BudgetPeriod_YEARLY:
		return model.BudgetPeriodYearly
	case budgetv1.BudgetPeriod_CUSTOM:
		return model.BudgetPeriodCustom
	default:
		return model.BudgetPeriodMonthly
	}
}

func budgetStatusFromProto(s budgetv1.BudgetStatus) model.BudgetStatus {
	switch s {
	case budgetv1.BudgetStatus_ACTIVE:
		return model.BudgetStatusActive
	case budgetv1.BudgetStatus_PAUSED:
		return model.BudgetStatusPaused
	case budgetv1.BudgetStatus_COMPLETED:
		return model.BudgetStatusCompleted
	case budgetv1.BudgetStatus_CANCELLED:
		return model.BudgetStatusCancelled
	default:
		return model.BudgetStatusActive
	}
}

func budgetStatusToProtoPtr(s *model.BudgetStatus) *budgetv1.BudgetStatus {
	if s == nil {
		return nil
	}
	switch *s {
	case model.BudgetStatusActive:
		return ptrOf(budgetv1.BudgetStatus_ACTIVE)
	case model.BudgetStatusPaused:
		return ptrOf(budgetv1.BudgetStatus_PAUSED)
	case model.BudgetStatusCompleted:
		return ptrOf(budgetv1.BudgetStatus_COMPLETED)
	case model.BudgetStatusCancelled:
		return ptrOf(budgetv1.BudgetStatus_CANCELLED)
	default:
		return nil
	}
}

// ── Credit Card Enum Converters ──────────────────────────────────────────

func cardBrandFromProto(b creditcardv1.CardBrand) model.CardBrand {
	switch b {
	case creditcardv1.CardBrand_VISA:
		return model.CardBrandVisa
	case creditcardv1.CardBrand_MASTERCARD:
		return model.CardBrandMastercard
	case creditcardv1.CardBrand_AMEX:
		return model.CardBrandAmex
	case creditcardv1.CardBrand_ELO:
		return model.CardBrandElo
	case creditcardv1.CardBrand_HIPERCARD:
		return model.CardBrandHipercard
	case creditcardv1.CardBrand_DINERS:
		return model.CardBrandDiners
	case creditcardv1.CardBrand_OTHER_BRAND:
		return model.CardBrandOtherBrand
	default:
		return model.CardBrandOtherBrand
	}
}

func cardTypeFromProto(t creditcardv1.CardType) model.CardType {
	switch t {
	case creditcardv1.CardType_CREDIT:
		return model.CardTypeCredit
	case creditcardv1.CardType_DEBIT:
		return model.CardTypeDebit
	case creditcardv1.CardType_MULTIPLE:
		return model.CardTypeMultiple
	default:
		return model.CardTypeCredit
	}
}

func invoiceStatusFromProto(s creditcardv1.InvoiceStatus) model.InvoiceStatus {
	switch s {
	case creditcardv1.InvoiceStatus_OPEN:
		return model.InvoiceStatusOpen
	case creditcardv1.InvoiceStatus_CLOSED:
		return model.InvoiceStatusClosed
	case creditcardv1.InvoiceStatus_PAID:
		return model.InvoiceStatusPaid
	case creditcardv1.InvoiceStatus_OVERDUE:
		return model.InvoiceStatusOverdue
	default:
		return model.InvoiceStatusOpen
	}
}

func invoiceStatusToProtoPtr(s *model.InvoiceStatus) *creditcardv1.InvoiceStatus {
	if s == nil {
		return nil
	}
	switch *s {
	case model.InvoiceStatusOpen:
		return ptrOf(creditcardv1.InvoiceStatus_OPEN)
	case model.InvoiceStatusClosed:
		return ptrOf(creditcardv1.InvoiceStatus_CLOSED)
	case model.InvoiceStatusPaid:
		return ptrOf(creditcardv1.InvoiceStatus_PAID)
	case model.InvoiceStatusOverdue:
		return ptrOf(creditcardv1.InvoiceStatus_OVERDUE)
	default:
		return nil
	}
}

// ── Debt Enum Converters ─────────────────────────────────────────────────

func debtTypeFromProto(t debtv1.DebtType) model.DebtType {
	switch t {
	case debtv1.DebtType_PERSONAL_LOAN:
		return model.DebtTypePersonalLoan
	case debtv1.DebtType_STUDENT_LOAN:
		return model.DebtTypeStudentLoan
	case debtv1.DebtType_MORTGAGE:
		return model.DebtTypeMortgage
	case debtv1.DebtType_CAR_LOAN:
		return model.DebtTypeCarLoan
	case debtv1.DebtType_CREDIT_CARD_DEBT:
		return model.DebtTypeCreditCardDebt
	case debtv1.DebtType_MEDICAL_DEBT:
		return model.DebtTypeMedicalDebt
	case debtv1.DebtType_OTHER_DEBT:
		return model.DebtTypeOtherDebt
	default:
		return model.DebtTypeOtherDebt
	}
}

func debtStatusFromProto(s debtv1.DebtStatus) model.DebtStatus {
	switch s {
	case debtv1.DebtStatus_ACTIVE:
		return model.DebtStatusActive
	case debtv1.DebtStatus_PAUSED:
		return model.DebtStatusPaused
	case debtv1.DebtStatus_PAID_OFF:
		return model.DebtStatusPaidOff
	case debtv1.DebtStatus_DEFAULTED:
		return model.DebtStatusDefaulted
	case debtv1.DebtStatus_SETTLED:
		return model.DebtStatusSettled
	default:
		return model.DebtStatusActive
	}
}

func debtStatusToProtoPtr(s *model.DebtStatus) *debtv1.DebtStatus {
	if s == nil {
		return nil
	}
	switch *s {
	case model.DebtStatusActive:
		return ptrOf(debtv1.DebtStatus_ACTIVE)
	case model.DebtStatusPaused:
		return ptrOf(debtv1.DebtStatus_PAUSED)
	case model.DebtStatusPaidOff:
		return ptrOf(debtv1.DebtStatus_PAID_OFF)
	case model.DebtStatusDefaulted:
		return ptrOf(debtv1.DebtStatus_DEFAULTED)
	case model.DebtStatusSettled:
		return ptrOf(debtv1.DebtStatus_SETTLED)
	default:
		return nil
	}
}

func debtTypeToProtoPtr(t *model.DebtType) *debtv1.DebtType {
	if t == nil {
		return nil
	}
	switch *t {
	case model.DebtTypePersonalLoan:
		return ptrOf(debtv1.DebtType_PERSONAL_LOAN)
	case model.DebtTypeStudentLoan:
		return ptrOf(debtv1.DebtType_STUDENT_LOAN)
	case model.DebtTypeMortgage:
		return ptrOf(debtv1.DebtType_MORTGAGE)
	case model.DebtTypeCarLoan:
		return ptrOf(debtv1.DebtType_CAR_LOAN)
	case model.DebtTypeCreditCardDebt:
		return ptrOf(debtv1.DebtType_CREDIT_CARD_DEBT)
	case model.DebtTypeMedicalDebt:
		return ptrOf(debtv1.DebtType_MEDICAL_DEBT)
	case model.DebtTypeOtherDebt:
		return ptrOf(debtv1.DebtType_OTHER_DEBT)
	default:
		return nil
	}
}

// ── Investment Enum Converters ───────────────────────────────────────────

func assetTypeFromProto(t investmentv1.AssetType) model.AssetType {
	switch t {
	case investmentv1.AssetType_STOCK:
		return model.AssetTypeStock
	case investmentv1.AssetType_ETF:
		return model.AssetTypeEtf
	case investmentv1.AssetType_REAL_ESTATE_FUND:
		return model.AssetTypeRealEstateFund
	case investmentv1.AssetType_TREASURY:
		return model.AssetTypeTreasury
	case investmentv1.AssetType_CDB:
		return model.AssetTypeCdb
	case investmentv1.AssetType_LCI:
		return model.AssetTypeLci
	case investmentv1.AssetType_LCA:
		return model.AssetTypeLca
	case investmentv1.AssetType_CRYPTO:
		return model.AssetTypeCrypto
	case investmentv1.AssetType_PENSION:
		return model.AssetTypePension
	case investmentv1.AssetType_FUND:
		return model.AssetTypeFund
	case investmentv1.AssetType_DOLLAR:
		return model.AssetTypeDollar
	case investmentv1.AssetType_GOLD:
		return model.AssetTypeGold
	case investmentv1.AssetType_OTHER_ASSET:
		return model.AssetTypeOtherAsset
	default:
		return model.AssetTypeOtherAsset
	}
}

func transactionTypeFromProto(t investmentv1.TransactionType) model.InvestmentTransactionType {
	switch t {
	case investmentv1.TransactionType_BUY:
		return model.InvestmentTransactionTypeBuy
	case investmentv1.TransactionType_SELL:
		return model.InvestmentTransactionTypeSell
	case investmentv1.TransactionType_DIVIDEND:
		return model.InvestmentTransactionTypeDividend
	case investmentv1.TransactionType_JCP:
		return model.InvestmentTransactionTypeJcp
	case investmentv1.TransactionType_AMORTIZATION:
		return model.InvestmentTransactionTypeAmortization
	default:
		return model.InvestmentTransactionTypeBuy
	}
}

func investmentStatusFromProto(s investmentv1.InvestmentStatus) model.InvestmentStatus {
	switch s {
	case investmentv1.InvestmentStatus_ACTIVE:
		return model.InvestmentStatusActive
	case investmentv1.InvestmentStatus_SOLD:
		return model.InvestmentStatusSold
	case investmentv1.InvestmentStatus_CANCELLED:
		return model.InvestmentStatusCancelled
	default:
		return model.InvestmentStatusActive
	}
}

func assetTypeToProtoPtr(t *model.AssetType) *investmentv1.AssetType {
	if t == nil {
		return nil
	}
	switch *t {
	case model.AssetTypeStock:
		return ptrOf(investmentv1.AssetType_STOCK)
	case model.AssetTypeEtf:
		return ptrOf(investmentv1.AssetType_ETF)
	case model.AssetTypeRealEstateFund:
		return ptrOf(investmentv1.AssetType_REAL_ESTATE_FUND)
	case model.AssetTypeTreasury:
		return ptrOf(investmentv1.AssetType_TREASURY)
	case model.AssetTypeCdb:
		return ptrOf(investmentv1.AssetType_CDB)
	case model.AssetTypeLci:
		return ptrOf(investmentv1.AssetType_LCI)
	case model.AssetTypeLca:
		return ptrOf(investmentv1.AssetType_LCA)
	case model.AssetTypeCrypto:
		return ptrOf(investmentv1.AssetType_CRYPTO)
	case model.AssetTypePension:
		return ptrOf(investmentv1.AssetType_PENSION)
	case model.AssetTypeFund:
		return ptrOf(investmentv1.AssetType_FUND)
	case model.AssetTypeDollar:
		return ptrOf(investmentv1.AssetType_DOLLAR)
	case model.AssetTypeGold:
		return ptrOf(investmentv1.AssetType_GOLD)
	case model.AssetTypeOtherAsset:
		return ptrOf(investmentv1.AssetType_OTHER_ASSET)
	default:
		return nil
	}
}

func investmentStatusToProtoPtr(s *model.InvestmentStatus) *investmentv1.InvestmentStatus {
	if s == nil {
		return nil
	}
	switch *s {
	case model.InvestmentStatusActive:
		return ptrOf(investmentv1.InvestmentStatus_ACTIVE)
	case model.InvestmentStatusSold:
		return ptrOf(investmentv1.InvestmentStatus_SOLD)
	case model.InvestmentStatusCancelled:
		return ptrOf(investmentv1.InvestmentStatus_CANCELLED)
	default:
		return nil
	}
}

func transactionTypeToProtoPtr(t *model.InvestmentTransactionType) *investmentv1.TransactionType {
	if t == nil {
		return nil
	}
	switch *t {
	case model.InvestmentTransactionTypeBuy:
		return ptrOf(investmentv1.TransactionType_BUY)
	case model.InvestmentTransactionTypeSell:
		return ptrOf(investmentv1.TransactionType_SELL)
	case model.InvestmentTransactionTypeDividend:
		return ptrOf(investmentv1.TransactionType_DIVIDEND)
	case model.InvestmentTransactionTypeJcp:
		return ptrOf(investmentv1.TransactionType_JCP)
	case model.InvestmentTransactionTypeAmortization:
		return ptrOf(investmentv1.TransactionType_AMORTIZATION)
	default:
		return nil
	}
}

func (r *Resolver) isFeatureEnabled(ctx context.Context, flag string) bool {
	if r.FFClient == nil {
		return false
	}
	return r.FFClient.IsEnabled(ctx, flag)
}

func (r *queryResolver) cachedSingle(ctx context.Context, entity, id string, dest interface{}, fetchFn func() (interface{}, error)) error {
	if r.Resolver.Cache == nil {
		val, err := fetchFn()
		if err != nil {
			return err
		}
		data, marshalErr := json.Marshal(val)
		if marshalErr != nil {
			return marshalErr
		}
		return json.Unmarshal(data, dest)
	}
	return r.Resolver.Cache.GetOrSet(ctx, cache.CacheKey(entity, id), 5*time.Minute, fetchFn, dest)
}

func (r *queryResolver) cachedList(ctx context.Context, entity string, args interface{}, dest interface{}, fetchFn func() (interface{}, error)) error {
	key := cache.CacheKeyList(entity, args)
	if r.Resolver.Cache == nil {
		val, err := fetchFn()
		if err != nil {
			return err
		}
		data, marshalErr := json.Marshal(val)
		if marshalErr != nil {
			return marshalErr
		}
		return json.Unmarshal(data, dest)
	}
	return r.Resolver.Cache.GetOrSet(ctx, key, 2*time.Minute, fetchFn, dest)
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
		return fmt.Errorf("identity-svc error: %s", st.Message())
	}
}
