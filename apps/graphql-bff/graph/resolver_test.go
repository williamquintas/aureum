package graph

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	budgetv1 "github.com/aureum/proto/gen/budget/budgetv1"
	creditcardv1 "github.com/aureum/proto/gen/creditcard/creditcardv1"
	debtv1 "github.com/aureum/proto/gen/debt/debtv1"
	identityv1 "github.com/aureum/proto/gen/identity/identityv1"
	investmentv1 "github.com/aureum/proto/gen/investment/investmentv1"
	transactionv1 "github.com/aureum/proto/gen/transaction/transactionv1"

	"github.com/aureum/graphql-bff/graph/model"
	"github.com/aureum/graphql-bff/internal/infrastructure/clients"
)

// ── Mock Transaction Service ──────────────────────────────────────────────

type mockTxService struct {
	transactionv1.UnimplementedTransactionServiceServer
	incomes          map[string]*transactionv1.Income
	fixedExpenses    map[string]*transactionv1.FixedExpense
	variableExpenses map[string]*transactionv1.VariableExpense
}

func newMockTxService() *mockTxService {
	return &mockTxService{
		incomes:          make(map[string]*transactionv1.Income),
		fixedExpenses:    make(map[string]*transactionv1.FixedExpense),
		variableExpenses: make(map[string]*transactionv1.VariableExpense),
	}
}

func (m *mockTxService) GetIncome(ctx context.Context, req *transactionv1.GetIncomeRequest) (*transactionv1.Income, error) {
	if inc, ok := m.incomes[req.Id]; ok {
		return inc, nil
	}
	return nil, status.Error(codes.NotFound, "income not found")
}

func (m *mockTxService) ListIncomes(ctx context.Context, req *transactionv1.ListIncomesRequest) (*transactionv1.ListIncomesResponse, error) {
	var list []*transactionv1.Income
	for _, inc := range m.incomes {
		list = append(list, inc)
	}
	return &transactionv1.ListIncomesResponse{
		Incomes:    list,
		TotalCount: int32(len(list)),
	}, nil
}

func (m *mockTxService) GetFixedExpense(ctx context.Context, req *transactionv1.GetFixedExpenseRequest) (*transactionv1.FixedExpense, error) {
	if fe, ok := m.fixedExpenses[req.Id]; ok {
		return fe, nil
	}
	return nil, status.Error(codes.NotFound, "fixed expense not found")
}

func (m *mockTxService) ListFixedExpenses(ctx context.Context, req *transactionv1.ListFixedExpensesRequest) (*transactionv1.ListFixedExpensesResponse, error) {
	var list []*transactionv1.FixedExpense
	for _, fe := range m.fixedExpenses {
		list = append(list, fe)
	}
	return &transactionv1.ListFixedExpensesResponse{
		FixedExpenses: list,
		TotalCount:    int32(len(list)),
	}, nil
}

func (m *mockTxService) GetVariableExpense(ctx context.Context, req *transactionv1.GetVariableExpenseRequest) (*transactionv1.VariableExpense, error) {
	if ve, ok := m.variableExpenses[req.Id]; ok {
		return ve, nil
	}
	return nil, status.Error(codes.NotFound, "variable expense not found")
}

func (m *mockTxService) ListVariableExpenses(ctx context.Context, req *transactionv1.ListVariableExpensesRequest) (*transactionv1.ListVariableExpensesResponse, error) {
	var list []*transactionv1.VariableExpense
	for _, ve := range m.variableExpenses {
		list = append(list, ve)
	}
	return &transactionv1.ListVariableExpensesResponse{
		VariableExpenses: list,
		TotalCount:       int32(len(list)),
	}, nil
}

func (m *mockTxService) CreateIncome(ctx context.Context, req *transactionv1.CreateIncomeRequest) (*transactionv1.Income, error) {
	id := fmt.Sprintf("inc-%d", len(m.incomes)+1)
	now := timestamppb.Now()
	inc := &transactionv1.Income{
		Id:             id,
		UserId:         "user-123",
		Description:    req.Description,
		Source:         req.Source,
		IncomeType:     req.IncomeType,
		ReceivedDate:   req.ReceivedDate,
		ReceivedAmount: req.ReceivedAmount,
		Status:         req.Status,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	m.incomes[id] = inc
	return inc, nil
}

func (m *mockTxService) UpdateIncome(ctx context.Context, req *transactionv1.UpdateIncomeRequest) (*transactionv1.Income, error) {
	inc, ok := m.incomes[req.Id]
	if !ok {
		return nil, status.Error(codes.NotFound, "income not found")
	}
	if req.Description != nil {
		inc.Description = *req.Description
	}
	if req.Source != nil {
		inc.Source = *req.Source
	}
	if req.IncomeType != nil {
		inc.IncomeType = *req.IncomeType
	}
	if req.ReceivedDate != nil {
		inc.ReceivedDate = *req.ReceivedDate
	}
	if req.ReceivedAmount != nil {
		inc.ReceivedAmount = *req.ReceivedAmount
	}
	if req.Status != nil {
		inc.Status = *req.Status
	}
	inc.UpdatedAt = timestamppb.Now()
	return inc, nil
}

func (m *mockTxService) DeleteIncome(ctx context.Context, req *transactionv1.DeleteIncomeRequest) (*emptypb.Empty, error) {
	if _, ok := m.incomes[req.Id]; !ok {
		return nil, status.Error(codes.NotFound, "income not found")
	}
	delete(m.incomes, req.Id)
	return &emptypb.Empty{}, nil
}

func (m *mockTxService) CreateFixedExpense(ctx context.Context, req *transactionv1.CreateFixedExpenseRequest) (*transactionv1.FixedExpense, error) {
	id := fmt.Sprintf("fe-%d", len(m.fixedExpenses)+1)
	now := timestamppb.Now()
	fe := &transactionv1.FixedExpense{
		Id:            id,
		UserId:        "user-123",
		Description:   req.Description,
		Category:      req.Category,
		DayOfMonth:    req.DayOfMonth,
		PaymentMethod: req.PaymentMethod,
		Status:        req.Status,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	m.fixedExpenses[id] = fe
	return fe, nil
}

func (m *mockTxService) UpdateFixedExpense(ctx context.Context, req *transactionv1.UpdateFixedExpenseRequest) (*transactionv1.FixedExpense, error) {
	fe, ok := m.fixedExpenses[req.Id]
	if !ok {
		return nil, status.Error(codes.NotFound, "fixed expense not found")
	}
	if req.Description != nil {
		fe.Description = *req.Description
	}
	if req.Category != nil {
		fe.Category = *req.Category
	}
	if req.DayOfMonth != nil {
		fe.DayOfMonth = *req.DayOfMonth
	}
	if req.PaymentMethod != nil {
		fe.PaymentMethod = *req.PaymentMethod
	}
	if req.Status != nil {
		fe.Status = *req.Status
	}
	fe.UpdatedAt = timestamppb.Now()
	return fe, nil
}

func (m *mockTxService) DeleteFixedExpense(ctx context.Context, req *transactionv1.DeleteFixedExpenseRequest) (*emptypb.Empty, error) {
	if _, ok := m.fixedExpenses[req.Id]; !ok {
		return nil, status.Error(codes.NotFound, "fixed expense not found")
	}
	delete(m.fixedExpenses, req.Id)
	return &emptypb.Empty{}, nil
}

func (m *mockTxService) CreateVariableExpense(ctx context.Context, req *transactionv1.CreateVariableExpenseRequest) (*transactionv1.VariableExpense, error) {
	id := fmt.Sprintf("ve-%d", len(m.variableExpenses)+1)
	now := timestamppb.Now()
	ve := &transactionv1.VariableExpense{
		Id:            id,
		UserId:        "user-123",
		Description:   req.Description,
		Destination:   req.Destination,
		Category:      req.Category,
		ExpenseType:   req.ExpenseType,
		PaymentMethod: req.PaymentMethod,
		PaymentDate:   req.PaymentDate,
		PaidAmount:    req.PaidAmount,
		Status:        req.Status,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	m.variableExpenses[id] = ve
	return ve, nil
}

func (m *mockTxService) UpdateVariableExpense(ctx context.Context, req *transactionv1.UpdateVariableExpenseRequest) (*transactionv1.VariableExpense, error) {
	ve, ok := m.variableExpenses[req.Id]
	if !ok {
		return nil, status.Error(codes.NotFound, "variable expense not found")
	}
	if req.Description != nil {
		ve.Description = *req.Description
	}
	if req.Destination != nil {
		ve.Destination = *req.Destination
	}
	if req.Category != nil {
		ve.Category = *req.Category
	}
	if req.ExpenseType != nil {
		ve.ExpenseType = *req.ExpenseType
	}
	if req.PaymentMethod != nil {
		ve.PaymentMethod = *req.PaymentMethod
	}
	if req.PaymentDate != nil {
		ve.PaymentDate = *req.PaymentDate
	}
	if req.PaidAmount != nil {
		ve.PaidAmount = *req.PaidAmount
	}
	if req.Status != nil {
		ve.Status = *req.Status
	}
	ve.UpdatedAt = timestamppb.Now()
	return ve, nil
}

func (m *mockTxService) DeleteVariableExpense(ctx context.Context, req *transactionv1.DeleteVariableExpenseRequest) (*emptypb.Empty, error) {
	if _, ok := m.variableExpenses[req.Id]; !ok {
		return nil, status.Error(codes.NotFound, "variable expense not found")
	}
	delete(m.variableExpenses, req.Id)
	return &emptypb.Empty{}, nil
}

// ── Mock Identity Service ─────────────────────────────────────────────────

type mockIDService struct {
	identityv1.UnimplementedIdentityServiceServer
	users map[string]*identityv1.GetUserResponse
}

func newMockIDService() *mockIDService {
	return &mockIDService{
		users: make(map[string]*identityv1.GetUserResponse),
	}
}

func (m *mockIDService) ValidateToken(ctx context.Context, req *identityv1.ValidateTokenRequest) (*identityv1.ValidateTokenResponse, error) {
	if req.Token == "valid-token" {
		return &identityv1.ValidateTokenResponse{
			Valid:  true,
			UserId: "user-123",
		}, nil
	}
	return &identityv1.ValidateTokenResponse{Valid: false}, nil
}

func (m *mockIDService) GetUser(ctx context.Context, req *identityv1.GetUserRequest) (*identityv1.GetUserResponse, error) {
	if user, ok := m.users[req.UserId]; ok {
		return user, nil
	}
	return nil, status.Error(codes.NotFound, "user not found")
}

// ── Mock Budget Service ────────────────────────────────────────────────────

type mockBudgetService struct {
	budgetv1.UnimplementedBudgetServiceServer
	budgets   map[string]*budgetv1.Budget
	summaries map[string]*budgetv1.BudgetSummary
}

func newMockBudgetService() *mockBudgetService {
	return &mockBudgetService{
		budgets:   make(map[string]*budgetv1.Budget),
		summaries: make(map[string]*budgetv1.BudgetSummary),
	}
}

func (m *mockBudgetService) GetBudget(ctx context.Context, req *budgetv1.GetBudgetRequest) (*budgetv1.Budget, error) {
	if b, ok := m.budgets[req.Id]; ok {
		return b, nil
	}
	return nil, status.Error(codes.NotFound, "budget not found")
}

func (m *mockBudgetService) ListBudgets(ctx context.Context, req *budgetv1.ListBudgetsRequest) (*budgetv1.ListBudgetsResponse, error) {
	var list []*budgetv1.Budget
	for _, b := range m.budgets {
		list = append(list, b)
	}
	return &budgetv1.ListBudgetsResponse{
		Budgets:    list,
		TotalCount: int32(len(list)),
	}, nil
}

func (m *mockBudgetService) GetBudgetSummary(ctx context.Context, req *budgetv1.GetBudgetSummaryRequest) (*budgetv1.BudgetSummary, error) {
	if s, ok := m.summaries[req.Id]; ok {
		return s, nil
	}
	return nil, status.Error(codes.NotFound, "budget summary not found")
}

// ── Mock Credit Card Service ──────────────────────────────────────────────

type mockCreditCardService struct {
	creditcardv1.UnimplementedCreditCardServiceServer
	creditCards  map[string]*creditcardv1.CreditCard
	invoices     map[string]*creditcardv1.Invoice
	transactions map[string]*creditcardv1.InvoiceTransaction
}

func newMockCreditCardService() *mockCreditCardService {
	return &mockCreditCardService{
		creditCards:  make(map[string]*creditcardv1.CreditCard),
		invoices:     make(map[string]*creditcardv1.Invoice),
		transactions: make(map[string]*creditcardv1.InvoiceTransaction),
	}
}

func (m *mockCreditCardService) GetCreditCard(ctx context.Context, req *creditcardv1.GetCreditCardRequest) (*creditcardv1.CreditCard, error) {
	if cc, ok := m.creditCards[req.Id]; ok {
		return cc, nil
	}
	return nil, status.Error(codes.NotFound, "credit card not found")
}

func (m *mockCreditCardService) ListCreditCards(ctx context.Context, req *creditcardv1.ListCreditCardsRequest) (*creditcardv1.ListCreditCardsResponse, error) {
	var list []*creditcardv1.CreditCard
	for _, cc := range m.creditCards {
		list = append(list, cc)
	}
	return &creditcardv1.ListCreditCardsResponse{
		CreditCards: list,
		TotalCount:  int32(len(list)),
	}, nil
}

func (m *mockCreditCardService) GetInvoice(ctx context.Context, req *creditcardv1.GetInvoiceRequest) (*creditcardv1.Invoice, error) {
	if inv, ok := m.invoices[req.Id]; ok {
		return inv, nil
	}
	return nil, status.Error(codes.NotFound, "invoice not found")
}

func (m *mockCreditCardService) ListInvoices(ctx context.Context, req *creditcardv1.ListInvoicesRequest) (*creditcardv1.ListInvoicesResponse, error) {
	var list []*creditcardv1.Invoice
	for _, inv := range m.invoices {
		list = append(list, inv)
	}
	return &creditcardv1.ListInvoicesResponse{
		Invoices:   list,
		TotalCount: int32(len(list)),
	}, nil
}

func (m *mockCreditCardService) ListTransactions(ctx context.Context, req *creditcardv1.ListTransactionsRequest) (*creditcardv1.ListTransactionsResponse, error) {
	var list []*creditcardv1.InvoiceTransaction
	for _, t := range m.transactions {
		list = append(list, t)
	}
	return &creditcardv1.ListTransactionsResponse{
		Transactions: list,
		TotalCount:   int32(len(list)),
	}, nil
}

// ── Mock Debt Service ─────────────────────────────────────────────────────

type mockDebtService struct {
	debtv1.UnimplementedDebtServiceServer
	debts    map[string]*debtv1.Debt
	payments map[string]*debtv1.Payment
}

func newMockDebtService() *mockDebtService {
	return &mockDebtService{
		debts:    make(map[string]*debtv1.Debt),
		payments: make(map[string]*debtv1.Payment),
	}
}

func (m *mockDebtService) GetDebt(ctx context.Context, req *debtv1.GetDebtRequest) (*debtv1.Debt, error) {
	if d, ok := m.debts[req.Id]; ok {
		return d, nil
	}
	return nil, status.Error(codes.NotFound, "debt not found")
}

func (m *mockDebtService) ListDebts(ctx context.Context, req *debtv1.ListDebtsRequest) (*debtv1.ListDebtsResponse, error) {
	var list []*debtv1.Debt
	for _, d := range m.debts {
		list = append(list, d)
	}
	return &debtv1.ListDebtsResponse{
		Debts:      list,
		TotalCount: int32(len(list)),
	}, nil
}

func (m *mockDebtService) ListPayments(ctx context.Context, req *debtv1.ListPaymentsRequest) (*debtv1.ListPaymentsResponse, error) {
	var list []*debtv1.Payment
	for _, p := range m.payments {
		list = append(list, p)
	}
	return &debtv1.ListPaymentsResponse{
		Payments:   list,
		TotalCount: int32(len(list)),
	}, nil
}

// ── Mock Investment Service ───────────────────────────────────────────────

type mockInvestmentService struct {
	investmentv1.UnimplementedInvestmentServiceServer
	investments      map[string]*investmentv1.Investment
	transactions     map[string]*investmentv1.InvestmentTransaction
	portfolioSummary *investmentv1.PortfolioSummary
}

func newMockInvestmentService() *mockInvestmentService {
	return &mockInvestmentService{
		investments:  make(map[string]*investmentv1.Investment),
		transactions: make(map[string]*investmentv1.InvestmentTransaction),
	}
}

func (m *mockInvestmentService) GetInvestment(ctx context.Context, req *investmentv1.GetInvestmentRequest) (*investmentv1.Investment, error) {
	if inv, ok := m.investments[req.Id]; ok {
		return inv, nil
	}
	return nil, status.Error(codes.NotFound, "investment not found")
}

func (m *mockInvestmentService) ListInvestments(ctx context.Context, req *investmentv1.ListInvestmentsRequest) (*investmentv1.ListInvestmentsResponse, error) {
	var list []*investmentv1.Investment
	for _, inv := range m.investments {
		list = append(list, inv)
	}
	return &investmentv1.ListInvestmentsResponse{
		Investments: list,
		TotalCount:  int32(len(list)),
	}, nil
}

func (m *mockInvestmentService) ListTransactions(ctx context.Context, req *investmentv1.ListTransactionsRequest) (*investmentv1.ListTransactionsResponse, error) {
	var list []*investmentv1.InvestmentTransaction
	for _, t := range m.transactions {
		list = append(list, t)
	}
	return &investmentv1.ListTransactionsResponse{
		Transactions: list,
		TotalCount:   int32(len(list)),
	}, nil
}

func (m *mockInvestmentService) GetPortfolioSummary(ctx context.Context, req *investmentv1.GetPortfolioSummaryRequest) (*investmentv1.PortfolioSummary, error) {
	if m.portfolioSummary != nil {
		return m.portfolioSummary, nil
	}
	return nil, status.Error(codes.NotFound, "portfolio summary not found")
}

// ── Test Helpers ──────────────────────────────────────────────────────────

func startTestGRPCServer(t *testing.T, services ...func(s *grpc.Server)) net.Listener {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	s := grpc.NewServer()
	for _, register := range services {
		register(s)
	}
	go s.Serve(lis)
	t.Cleanup(func() {
		s.GracefulStop()
	})
	return lis
}

func dialListener(t *testing.T, lis net.Listener) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.Dial(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

func testIncomeProto(id string) *transactionv1.Income {
	now := timestamppb.New(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	return &transactionv1.Income{
		Id:             id,
		UserId:         "user-123",
		Description:    "Freelance project",
		Source:         "Upwork",
		IncomeType:     transactionv1.IncomeType_FREELANCE,
		ReceivedDate:   "2024-01-15",
		ReceivedAmount: 500000,
		Status:         transactionv1.TransactionStatus_COMPLETED,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func testFixedExpenseProto(id string) *transactionv1.FixedExpense {
	now := timestamppb.New(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	return &transactionv1.FixedExpense{
		Id:            id,
		UserId:        "user-123",
		Description:   "Rent",
		Category:      "Housing",
		DayOfMonth:    5,
		PaymentMethod: transactionv1.PaymentMethod_BANK_TRANSFER,
		Status:        transactionv1.TransactionStatus_PENDING,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func testVariableExpenseProto(id string) *transactionv1.VariableExpense {
	now := timestamppb.New(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	return &transactionv1.VariableExpense{
		Id:            id,
		UserId:        "user-123",
		Description:   "Groceries",
		Destination:   "Supermarket",
		Category:      "Food",
		ExpenseType:   transactionv1.ExpenseType_ESSENTIAL,
		PaymentMethod: transactionv1.PaymentMethod_DEBIT_CARD,
		PaymentDate:   "2024-01-15",
		PaidAmount:    15000,
		Status:        transactionv1.TransactionStatus_COMPLETED,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func setupTestResolver(t *testing.T) (*Resolver, *mockTxService, *mockIDService, *mockBudgetService, *mockCreditCardService, *mockDebtService, *mockInvestmentService) {
	t.Helper()

	mockTx := newMockTxService()
	mockID := newMockIDService()
	mockBgt := newMockBudgetService()
	mockCCC := newMockCreditCardService()
	mockDbt := newMockDebtService()
	mockInv := newMockInvestmentService()

	txLis := startTestGRPCServer(t, func(s *grpc.Server) {
		transactionv1.RegisterTransactionServiceServer(s, mockTx)
	})
	idLis := startTestGRPCServer(t, func(s *grpc.Server) {
		identityv1.RegisterIdentityServiceServer(s, mockID)
	})
	bgtLis := startTestGRPCServer(t, func(s *grpc.Server) {
		budgetv1.RegisterBudgetServiceServer(s, mockBgt)
	})
	cccLis := startTestGRPCServer(t, func(s *grpc.Server) {
		creditcardv1.RegisterCreditCardServiceServer(s, mockCCC)
	})
	dbtLis := startTestGRPCServer(t, func(s *grpc.Server) {
		debtv1.RegisterDebtServiceServer(s, mockDbt)
	})
	invLis := startTestGRPCServer(t, func(s *grpc.Server) {
		investmentv1.RegisterInvestmentServiceServer(s, mockInv)
	})

	txConn := dialListener(t, txLis)
	idConn := dialListener(t, idLis)
	bgtConn := dialListener(t, bgtLis)
	cccConn := dialListener(t, cccLis)
	dbtConn := dialListener(t, dbtLis)
	invConn := dialListener(t, invLis)

	txClient := clients.NewTransactionServiceClient(txConn)
	idClient := clients.NewIdentityServiceClient(idConn)
	bgtClient := clients.NewBudgetServiceClient(bgtConn)
	cccClient := clients.NewCreditCardServiceClient(cccConn)
	dbtClient := clients.NewDebtServiceClient(dbtConn)
	invClient := clients.NewInvestmentServiceClient(invConn)

	resolver := NewResolver(txClient, idClient, bgtClient, cccClient, dbtClient, invClient, nil, nil)
	return resolver, mockTx, mockID, mockBgt, mockCCC, mockDbt, mockInv
}

func ctxWithUser(userID string) context.Context {
	return context.WithValue(context.Background(), userIDKey, userID)
}

// ── Proto Builder Helpers ─────────────────────────────────────────────────

func testBudgetProto(id string) *budgetv1.Budget {
	now := timestamppb.New(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	return &budgetv1.Budget{
		Id:          id,
		UserId:      "user-123",
		Name:        "Monthly Budget",
		Description: "Main monthly budget",
		Period:      budgetv1.BudgetPeriod_MONTHLY,
		TotalLimit:  500000,
		SpentAmount: 250000,
		Status:      budgetv1.BudgetStatus_ACTIVE,
		StartDate:   "2024-01-01",
		EndDate:     "2024-01-31",
		Categories: []*budgetv1.BudgetCategory{
			{
				Id:          "cat-1",
				BudgetId:    id,
				Name:        "Food",
				LimitAmount: 200000,
				SpentAmount: 100000,
				Category:    "Food",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func testBudgetSummaryProto() *budgetv1.BudgetSummary {
	return &budgetv1.BudgetSummary{
		BudgetId:        "bgt-1",
		TotalLimit:      500000,
		TotalSpent:      250000,
		Remaining:       250000,
		UsagePercentage: 50.0,
		CategoryCount:   1,
		Categories: []*budgetv1.CategorySummary{
			{
				CategoryId:      "cat-1",
				Name:            "Food",
				Category:        "Food",
				LimitAmount:     200000,
				SpentAmount:     100000,
				Remaining:       100000,
				UsagePercentage: 50.0,
			},
		},
	}
}

func testCreditCardProto(id string) *creditcardv1.CreditCard {
	now := timestamppb.New(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	return &creditcardv1.CreditCard{
		Id:              id,
		UserId:          "user-123",
		Name:            "My Card",
		Brand:           creditcardv1.CardBrand_VISA,
		CardType:        creditcardv1.CardType_CREDIT,
		LastFourDigits:  "1234",
		ClosingDay:      15,
		DueDay:          5,
		CreditLimit:     1000000,
		AvailableCredit: 500000,
		Active:          true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func testInvoiceProto(id string) *creditcardv1.Invoice {
	now := timestamppb.New(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	return &creditcardv1.Invoice{
		Id:             id,
		CreditCardId:   "ccc-1",
		UserId:         "user-123",
		ReferenceMonth: "2024-01",
		TotalAmount:    500000,
		PaidAmount:     500000,
		Status:         creditcardv1.InvoiceStatus_PAID,
		ClosingDate:    "2024-01-10",
		DueDate:        "2024-02-05",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func testInvoiceTransactionProto(id string) *creditcardv1.InvoiceTransaction {
	now := timestamppb.New(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	return &creditcardv1.InvoiceTransaction{
		Id:              id,
		InvoiceId:       "inv-1",
		UserId:          "user-123",
		Description:     "Restaurant",
		Amount:          50000,
		Category:        "Food",
		TransactionDate: "2024-01-05",
		Installments:    1,
		CreatedAt:       now,
	}
}

func testDebtProto(id string) *debtv1.Debt {
	now := timestamppb.New(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	return &debtv1.Debt{
		Id:              id,
		UserId:          "user-123",
		Name:            "Student Loan",
		Description:     "University loan",
		DebtType:        debtv1.DebtType_STUDENT_LOAN,
		TotalAmount:     5000000,
		RemainingAmount: 3000000,
		InterestRate:    500,
		StartDate:       "2023-01-01",
		ExpectedEndDate: "2027-01-01",
		Status:          debtv1.DebtStatus_ACTIVE,
		Creditor:        "Bank",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func testPaymentProto(id string) *debtv1.Payment {
	now := timestamppb.New(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	return &debtv1.Payment{
		Id:          id,
		DebtId:      "debt-1",
		UserId:      "user-123",
		Amount:      100000,
		PaymentDate: "2024-01-15",
		Notes:       "Monthly payment",
		CreatedAt:   now,
	}
}

func testInvestmentProto(id string) *investmentv1.Investment {
	now := timestamppb.New(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	return &investmentv1.Investment{
		Id:            id,
		UserId:        "user-123",
		Name:          "Tech Stock",
		Ticker:        "TECH",
		AssetType:     investmentv1.AssetType_STOCK,
		Quantity:      100,
		AveragePrice:  5000,
		TotalInvested: 500000,
		Status:        investmentv1.InvestmentStatus_ACTIVE,
		Broker:        "BrokerX",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func testInvestmentTransactionProto(id string) *investmentv1.InvestmentTransaction {
	now := timestamppb.New(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	return &investmentv1.InvestmentTransaction{
		Id:              id,
		InvestmentId:    "inv-1",
		UserId:          "user-123",
		TransactionType: investmentv1.TransactionType_BUY,
		Quantity:        10,
		UnitPrice:       5000,
		TotalAmount:     50000,
		TransactionDate: "2024-01-15",
		Notes:           "Purchase",
		CreatedAt:       now,
	}
}

func testPortfolioSummaryProto() *investmentv1.PortfolioSummary {
	return &investmentv1.PortfolioSummary{
		TotalInvested:     1000000,
		CurrentValue:      1200000,
		TotalReturn:       200000,
		ReturnPercentage:  20.0,
		ActiveInvestments: 5,
		Allocation: []*investmentv1.AssetAllocation{
			{
				AssetType:    investmentv1.AssetType_STOCK,
				Invested:     500000,
				CurrentValue: 600000,
				Percentage:   50.0,
			},
			{
				AssetType:    investmentv1.AssetType_ETF,
				Invested:     500000,
				CurrentValue: 600000,
				Percentage:   50.0,
			},
		},
	}
}

// ── Helper Function Tests ─────────────────────────────────────────────────

func TestLimitAndOffset(t *testing.T) {
	tests := []struct {
		name       string
		first      *int
		after      *string
		wantLimit  int
		wantOffset int
	}{
		{
			name:       "default values",
			first:      nil,
			after:      nil,
			wantLimit:  20,
			wantOffset: 0,
		},
		{
			name:       "custom limit",
			first:      ptrOf(10),
			after:      nil,
			wantLimit:  10,
			wantOffset: 0,
		},
		{
			name:       "with cursor",
			first:      nil,
			after:      ptrOf("5"),
			wantLimit:  20,
			wantOffset: 5,
		},
		{
			name:       "custom limit and cursor",
			first:      ptrOf(50),
			after:      ptrOf("10"),
			wantLimit:  50,
			wantOffset: 10,
		},
		{
			name:       "zero first defaults",
			first:      ptrOf(0),
			after:      nil,
			wantLimit:  20,
			wantOffset: 0,
		},
		{
			name:       "empty after string",
			first:      nil,
			after:      ptrOf(""),
			wantLimit:  20,
			wantOffset: 0,
		},
		{
			name:       "negative first defaults",
			first:      ptrOf(-5),
			after:      nil,
			wantLimit:  20,
			wantOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, offset := limitAndOffset(tt.first, tt.after)
			assert.Equal(t, tt.wantLimit, limit)
			assert.Equal(t, tt.wantOffset, offset)
		})
	}
}

func TestMapGRPCError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{
			name:     "not found error",
			err:      status.Error(codes.NotFound, "income not found"),
			contains: "not found: income not found",
		},
		{
			name:     "unknown gRPC error",
			err:      status.Error(codes.Internal, "internal error"),
			contains: "identity-svc error: internal error",
		},
		{
			name:     "non-grpc error",
			err:      fmt.Errorf("plain error"),
			contains: "plain error",
		},
		{
			name:     "unknown gRPC error (InvalidArgument)",
			err:      status.Error(codes.InvalidArgument, "bad request"),
			contains: "identity-svc error: bad request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapGRPCError(tt.err)
			require.NotNil(t, result)
			assert.Contains(t, result.Error(), tt.contains)
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		input string
		want  time.Time
	}{
		{input: "2024-01-15", want: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)},
		{input: "2023-12-01", want: time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)},
		{input: "invalid", want: time.Time{}},
		{input: "", want: time.Time{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDate(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDateToStrPtr(t *testing.T) {
	t.Run("non-nil time", func(t *testing.T) {
		d := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		result := dateToStrPtr(&d)
		require.NotNil(t, result)
		assert.Equal(t, "2024-06-15", *result)
	})

	t.Run("nil time", func(t *testing.T) {
		result := dateToStrPtr(nil)
		assert.Nil(t, result)
	})
}

func TestStrPtr(t *testing.T) {
	result := strPtr("hello")
	require.NotNil(t, result)
	assert.Equal(t, "hello", *result)
}

func TestPtrOf(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		result := ptrOf(42)
		require.NotNil(t, result)
		assert.Equal(t, 42, *result)
	})

	t.Run("string", func(t *testing.T) {
		result := ptrOf("test")
		require.NotNil(t, result)
		assert.Equal(t, "test", *result)
	})

	t.Run("bool", func(t *testing.T) {
		result := ptrOf(true)
		require.NotNil(t, result)
		assert.True(t, *result)
	})
}

func TestUserIDFromCtx(t *testing.T) {
	t.Run("with user id", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userIDKey, "user-123")
		assert.Equal(t, "user-123", userIDFromCtx(ctx))
	})

	t.Run("without user id", func(t *testing.T) {
		assert.Equal(t, "", userIDFromCtx(context.Background()))
	})

	t.Run("wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userIDKey, 42)
		assert.Equal(t, "", userIDFromCtx(ctx))
	})
}

// ── Query Resolver Tests ──────────────────────────────────────────────────

func TestQueryResolver_Income(t *testing.T) {
	resolver, mockTx, _, _, _, _, _ := setupTestResolver(t)

	mockTx.incomes["inc-1"] = testIncomeProto("inc-1")

	t.Run("success", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().Income(ctx, "inc-1")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "inc-1", result.ID)
		assert.Equal(t, "Freelance project", result.Description)
		assert.Equal(t, model.IncomeTypeFreelance, result.IncomeType)
		assert.Equal(t, model.TransactionStatusCompleted, result.Status)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().Income(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, result)
		// Circuit breaker intercepts gRPC errors and returns fallback
		assert.Contains(t, err.Error(), "transaction-svc unavailable")
	})
}

func TestQueryResolver_FixedExpense(t *testing.T) {
	resolver, mockTx, _, _, _, _, _ := setupTestResolver(t)

	mockTx.fixedExpenses["fe-1"] = testFixedExpenseProto("fe-1")

	t.Run("success", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().FixedExpense(ctx, "fe-1")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "fe-1", result.ID)
		assert.Equal(t, "Rent", result.Description)
		assert.Equal(t, model.PaymentMethodBankTransfer, result.PaymentMethod)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().FixedExpense(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestQueryResolver_VariableExpense(t *testing.T) {
	resolver, mockTx, _, _, _, _, _ := setupTestResolver(t)

	mockTx.variableExpenses["ve-1"] = testVariableExpenseProto("ve-1")

	t.Run("success", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().VariableExpense(ctx, "ve-1")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "ve-1", result.ID)
		assert.Equal(t, "Groceries", result.Description)
		assert.Equal(t, model.ExpenseTypeEssential, result.ExpenseType)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().VariableExpense(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestQueryResolver_IncomesList(t *testing.T) {
	resolver, mockTx, _, _, _, _, _ := setupTestResolver(t)

	mockTx.incomes["inc-1"] = testIncomeProto("inc-1")
	mockTx.incomes["inc-2"] = testIncomeProto("inc-2")

	t.Run("returns paginated list", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		first := 20
		result, err := resolver.Query().Incomes(ctx, &first, nil, nil, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, result.TotalCount)
		assert.Len(t, result.Edges, 2)
		assert.NotNil(t, result.PageInfo)
	})
}

func TestQueryResolver_Transactions(t *testing.T) {
	resolver, mockTx, _, _, _, _, _ := setupTestResolver(t)

	mockTx.incomes["inc-1"] = testIncomeProto("inc-1")
	mockTx.fixedExpenses["fe-1"] = testFixedExpenseProto("fe-1")
	mockTx.variableExpenses["ve-1"] = testVariableExpenseProto("ve-1")

	t.Run("all types", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().Transactions(ctx, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 3, result.TotalCount)
		assert.Len(t, result.Edges, 3)
	})

	t.Run("filter by income type", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		typeFilter := model.TransactionTypeFilterIncome
		result, err := resolver.Query().Transactions(ctx, nil, nil, &typeFilter, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.TotalCount)
		assert.Len(t, result.Edges, 1)
	})

	t.Run("filter by fixed expense type", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		typeFilter := model.TransactionTypeFilterFixedExpense
		result, err := resolver.Query().Transactions(ctx, nil, nil, &typeFilter, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.TotalCount)
		assert.Len(t, result.Edges, 1)
	})

	t.Run("filter by variable expense type", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		typeFilter := model.TransactionTypeFilterVariableExpense
		result, err := resolver.Query().Transactions(ctx, nil, nil, &typeFilter, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.TotalCount)
		assert.Len(t, result.Edges, 1)
	})
}

func TestQueryResolver_Me(t *testing.T) {
	resolver, _, mockID, _, _, _, _ := setupTestResolver(t)

	mockID.users["user-123"] = &identityv1.GetUserResponse{
		UserId: "user-123",
		Name:   "Test User",
		Email:  "test@example.com",
	}

	t.Run("authenticated user", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().Me(ctx)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "user-123", result.ID)
		assert.Equal(t, "Test User", result.Name)
		assert.Equal(t, "test@example.com", result.Email)
	})

	t.Run("unauthenticated returns error", func(t *testing.T) {
		result, err := resolver.Query().Me(context.Background())
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not authenticated")
	})

	t.Run("user not found", func(t *testing.T) {
		ctx := ctxWithUser("nonexistent")
		result, err := resolver.Query().Me(ctx)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestQueryResolver_Budget(t *testing.T) {
	resolver, _, _, mockBgt, _, _, _ := setupTestResolver(t)

	mockBgt.budgets["bgt-1"] = testBudgetProto("bgt-1")

	t.Run("success", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().Budget(ctx, "bgt-1")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "bgt-1", result.ID)
		assert.Equal(t, "Monthly Budget", result.Name)
		assert.Equal(t, model.BudgetPeriodMonthly, result.Period)
		assert.Equal(t, int64(500000), result.TotalLimit)
		assert.Equal(t, model.BudgetStatusActive, result.Status)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().Budget(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, result)
		// Circuit breaker intercepts gRPC errors and returns fallback
		assert.Contains(t, err.Error(), "budget-svc unavailable")
	})
}

func TestQueryResolver_BudgetsList(t *testing.T) {
	resolver, _, _, mockBgt, _, _, _ := setupTestResolver(t)

	mockBgt.budgets["bgt-1"] = testBudgetProto("bgt-1")
	mockBgt.budgets["bgt-2"] = testBudgetProto("bgt-2")

	t.Run("returns paginated list", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		first := 20
		result, err := resolver.Query().Budgets(ctx, &first, nil, nil, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, result.TotalCount)
		assert.Len(t, result.Edges, 2)
		assert.NotNil(t, result.PageInfo)
	})
}

func TestQueryResolver_BudgetSummary(t *testing.T) {
	resolver, _, _, mockBgt, _, _, _ := setupTestResolver(t)

	mockBgt.summaries["bgt-1"] = testBudgetSummaryProto()

	t.Run("success", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().BudgetSummary(ctx, "bgt-1")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "bgt-1", result.BudgetID)
		assert.Equal(t, int64(500000), result.TotalLimit)
		assert.Equal(t, int64(250000), result.TotalSpent)
		assert.Equal(t, 50.0, result.UsagePercentage)
		assert.Equal(t, 1, result.CategoryCount)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().BudgetSummary(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestQueryResolver_CreditCard(t *testing.T) {
	resolver, _, _, _, mockCCC, _, _ := setupTestResolver(t)

	mockCCC.creditCards["ccc-1"] = testCreditCardProto("ccc-1")

	t.Run("success", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().CreditCard(ctx, "ccc-1")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "ccc-1", result.ID)
		assert.Equal(t, "My Card", result.Name)
		assert.Equal(t, model.CardBrandVisa, result.Brand)
		assert.Equal(t, model.CardTypeCredit, result.CardType)
		assert.Equal(t, "1234", result.LastFourDigits)
		assert.True(t, result.Active)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().CreditCard(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, result)
		// Circuit breaker intercepts gRPC errors and returns fallback
		assert.Contains(t, err.Error(), "creditcard-svc unavailable")
	})
}

func TestQueryResolver_CreditCardsList(t *testing.T) {
	resolver, _, _, _, mockCCC, _, _ := setupTestResolver(t)

	mockCCC.creditCards["ccc-1"] = testCreditCardProto("ccc-1")
	mockCCC.creditCards["ccc-2"] = testCreditCardProto("ccc-2")

	t.Run("returns paginated list", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		first := 20
		result, err := resolver.Query().CreditCards(ctx, &first, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, result.TotalCount)
		assert.Len(t, result.Edges, 2)
		assert.NotNil(t, result.PageInfo)
	})
}

func TestQueryResolver_Invoice(t *testing.T) {
	resolver, _, _, _, mockCCC, _, _ := setupTestResolver(t)

	mockCCC.invoices["inv-1"] = testInvoiceProto("inv-1")

	t.Run("success", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().Invoice(ctx, "inv-1")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "inv-1", result.ID)
		assert.Equal(t, "ccc-1", result.CreditCardID)
		assert.Equal(t, "2024-01", result.ReferenceMonth)
		assert.Equal(t, model.InvoiceStatusPaid, result.Status)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().Invoice(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, result)
		// Circuit breaker intercepts gRPC errors and returns fallback
		assert.Contains(t, err.Error(), "creditcard-svc unavailable")
	})
}

func TestQueryResolver_InvoicesList(t *testing.T) {
	resolver, _, _, _, mockCCC, _, _ := setupTestResolver(t)

	mockCCC.invoices["inv-1"] = testInvoiceProto("inv-1")
	mockCCC.invoices["inv-2"] = testInvoiceProto("inv-2")
	creditCardID := "ccc-1"

	t.Run("returns paginated list", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		first := 20
		result, err := resolver.Query().Invoices(ctx, &first, nil, creditCardID, nil, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, result.TotalCount)
		assert.Len(t, result.Edges, 2)
		assert.NotNil(t, result.PageInfo)
	})
}

func TestQueryResolver_InvoiceTransactions(t *testing.T) {
	resolver, _, _, _, mockCCC, _, _ := setupTestResolver(t)

	mockCCC.transactions["tx-1"] = testInvoiceTransactionProto("tx-1")
	mockCCC.transactions["tx-2"] = testInvoiceTransactionProto("tx-2")

	t.Run("returns paginated list", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		first := 20
		result, err := resolver.Query().InvoiceTransactions(ctx, &first, nil, "inv-1", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, result.TotalCount)
		assert.Len(t, result.Edges, 2)
		assert.NotNil(t, result.PageInfo)
	})
}

func TestQueryResolver_Debt(t *testing.T) {
	resolver, _, _, _, _, mockDbt, _ := setupTestResolver(t)

	mockDbt.debts["debt-1"] = testDebtProto("debt-1")

	t.Run("success", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().Debt(ctx, "debt-1")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "debt-1", result.ID)
		assert.Equal(t, "Student Loan", result.Name)
		assert.Equal(t, model.DebtTypeStudentLoan, result.DebtType)
		assert.Equal(t, int64(5000000), result.TotalAmount)
		assert.Equal(t, model.DebtStatusActive, result.Status)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().Debt(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, result)
		// Circuit breaker intercepts gRPC errors and returns fallback
		assert.Contains(t, err.Error(), "debt-svc unavailable")
	})
}

func TestQueryResolver_DebtsList(t *testing.T) {
	resolver, _, _, _, _, mockDbt, _ := setupTestResolver(t)

	mockDbt.debts["debt-1"] = testDebtProto("debt-1")
	mockDbt.debts["debt-2"] = testDebtProto("debt-2")

	t.Run("returns paginated list", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		first := 20
		result, err := resolver.Query().Debts(ctx, &first, nil, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, result.TotalCount)
		assert.Len(t, result.Edges, 2)
		assert.NotNil(t, result.PageInfo)
	})
}

func TestQueryResolver_Payments(t *testing.T) {
	resolver, _, _, _, _, mockDbt, _ := setupTestResolver(t)

	mockDbt.payments["pmt-1"] = testPaymentProto("pmt-1")
	mockDbt.payments["pmt-2"] = testPaymentProto("pmt-2")

	t.Run("returns paginated list", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		first := 20
		result, err := resolver.Query().Payments(ctx, &first, nil, "debt-1", nil, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, result.TotalCount)
		assert.Len(t, result.Edges, 2)
		assert.NotNil(t, result.PageInfo)
	})
}

func TestQueryResolver_Investment(t *testing.T) {
	resolver, _, _, _, _, _, mockInv := setupTestResolver(t)

	mockInv.investments["inv-1"] = testInvestmentProto("inv-1")

	t.Run("success", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().Investment(ctx, "inv-1")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "inv-1", result.ID)
		assert.Equal(t, "Tech Stock", result.Name)
		assert.Equal(t, "TECH", result.Ticker)
		assert.Equal(t, model.AssetTypeStock, result.AssetType)
		assert.Equal(t, int64(100), result.Quantity)
		assert.Equal(t, model.InvestmentStatusActive, result.Status)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().Investment(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, result)
		// Circuit breaker intercepts gRPC errors and returns fallback
		assert.Contains(t, err.Error(), "investment-svc unavailable")
	})
}

func TestQueryResolver_InvestmentsList(t *testing.T) {
	resolver, _, _, _, _, _, mockInv := setupTestResolver(t)

	mockInv.investments["inv-1"] = testInvestmentProto("inv-1")
	mockInv.investments["inv-2"] = testInvestmentProto("inv-2")

	t.Run("returns paginated list", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		first := 20
		result, err := resolver.Query().Investments(ctx, &first, nil, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, result.TotalCount)
		assert.Len(t, result.Edges, 2)
		assert.NotNil(t, result.PageInfo)
	})
}

func TestQueryResolver_InvestmentTransactions(t *testing.T) {
	resolver, _, _, _, _, _, mockInv := setupTestResolver(t)

	mockInv.transactions["tx-1"] = testInvestmentTransactionProto("tx-1")
	mockInv.transactions["tx-2"] = testInvestmentTransactionProto("tx-2")

	t.Run("returns paginated list", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		first := 20
		result, err := resolver.Query().InvestmentTransactions(ctx, &first, nil, "inv-1", nil, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, result.TotalCount)
		assert.Len(t, result.Edges, 2)
		assert.NotNil(t, result.PageInfo)
	})
}

func TestQueryResolver_PortfolioSummary(t *testing.T) {
	resolver, _, _, _, _, _, mockInv := setupTestResolver(t)

	mockInv.portfolioSummary = testPortfolioSummaryProto()

	t.Run("success", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().PortfolioSummary(ctx)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(1000000), result.TotalInvested)
		assert.Equal(t, int64(1200000), result.CurrentValue)
		assert.Equal(t, int64(200000), result.TotalReturn)
		assert.Equal(t, 20.0, result.ReturnPercentage)
		assert.Equal(t, 5, result.ActiveInvestments)
		assert.Len(t, result.Allocation, 2)
	})
}

// ── Enum Converter Tests ──────────────────────────────────────────────────

func TestStatusFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input transactionv1.TransactionStatus
		want  model.TransactionStatus
	}{
		{name: "pending", input: transactionv1.TransactionStatus_PENDING, want: model.TransactionStatusPending},
		{name: "completed", input: transactionv1.TransactionStatus_COMPLETED, want: model.TransactionStatusCompleted},
		{name: "cancelled", input: transactionv1.TransactionStatus_CANCELLED, want: model.TransactionStatusCancelled},
		{name: "unknown defaults to pending", input: transactionv1.TransactionStatus(999), want: model.TransactionStatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, statusFromProto(tt.input))
		})
	}
}

func TestStatusToProto(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, statusToProto(nil))
	})

	tests := []struct {
		name    string
		input   model.TransactionStatus
		want    transactionv1.TransactionStatus
		wantNil bool
	}{
		{name: "pending", input: model.TransactionStatusPending, want: transactionv1.TransactionStatus_PENDING},
		{name: "completed", input: model.TransactionStatusCompleted, want: transactionv1.TransactionStatus_COMPLETED},
		{name: "cancelled", input: model.TransactionStatusCancelled, want: transactionv1.TransactionStatus_CANCELLED},
		{name: "unknown returns nil", input: model.TransactionStatus("unknown"), wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.input
			result := statusToProto(&s)
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Equal(t, tt.want, *result)
		})
	}
}

func TestIncomeTypeFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input transactionv1.IncomeType
		want  model.IncomeType
	}{
		{name: "salary", input: transactionv1.IncomeType_SALARY, want: model.IncomeTypeSalary},
		{name: "freelance", input: transactionv1.IncomeType_FREELANCE, want: model.IncomeTypeFreelance},
		{name: "investment", input: transactionv1.IncomeType_INVESTMENT, want: model.IncomeTypeInvestment},
		{name: "business", input: transactionv1.IncomeType_BUSINESS, want: model.IncomeTypeBusiness},
		{name: "refund", input: transactionv1.IncomeType_REFUND, want: model.IncomeTypeRefund},
		{name: "other", input: transactionv1.IncomeType_INCOME_OTHER, want: model.IncomeTypeOther},
		{name: "unknown defaults to other", input: transactionv1.IncomeType(999), want: model.IncomeTypeOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, incomeTypeFromProto(tt.input))
		})
	}
}

func TestExpenseTypeFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input transactionv1.ExpenseType
		want  model.ExpenseType
	}{
		{name: "essential", input: transactionv1.ExpenseType_ESSENTIAL, want: model.ExpenseTypeEssential},
		{name: "discretionary", input: transactionv1.ExpenseType_DISCRETIONARY, want: model.ExpenseTypeDiscretionary},
		{name: "occasional", input: transactionv1.ExpenseType_OCCASIONAL, want: model.ExpenseTypeOccasional},
		{name: "emergency", input: transactionv1.ExpenseType_EMERGENCY, want: model.ExpenseTypeEmergency},
		{name: "other", input: transactionv1.ExpenseType_EXPENSE_OTHER, want: model.ExpenseTypeOther},
		{name: "unknown defaults to other", input: transactionv1.ExpenseType(999), want: model.ExpenseTypeOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, expenseTypeFromProto(tt.input))
		})
	}
}

func TestPaymentMethodFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input transactionv1.PaymentMethod
		want  model.PaymentMethod
	}{
		{name: "credit card", input: transactionv1.PaymentMethod_CREDIT_CARD, want: model.PaymentMethodCreditCard},
		{name: "debit card", input: transactionv1.PaymentMethod_DEBIT_CARD, want: model.PaymentMethodDebitCard},
		{name: "cash", input: transactionv1.PaymentMethod_CASH, want: model.PaymentMethodCash},
		{name: "bank transfer", input: transactionv1.PaymentMethod_BANK_TRANSFER, want: model.PaymentMethodBankTransfer},
		{name: "pix", input: transactionv1.PaymentMethod_PIX, want: model.PaymentMethodPix},
		{name: "other", input: transactionv1.PaymentMethod_OTHER, want: model.PaymentMethodOther},
		{name: "unknown defaults to other", input: transactionv1.PaymentMethod(999), want: model.PaymentMethodOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, paymentMethodFromProto(tt.input))
		})
	}
}

func TestBudgetPeriodFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input budgetv1.BudgetPeriod
		want  model.BudgetPeriod
	}{
		{name: "monthly", input: budgetv1.BudgetPeriod_MONTHLY, want: model.BudgetPeriodMonthly},
		{name: "bimonthly", input: budgetv1.BudgetPeriod_BIMONTHLY, want: model.BudgetPeriodBimonthly},
		{name: "quarterly", input: budgetv1.BudgetPeriod_QUARTERLY, want: model.BudgetPeriodQuarterly},
		{name: "semestral", input: budgetv1.BudgetPeriod_SEMESTRAL, want: model.BudgetPeriodSemestral},
		{name: "yearly", input: budgetv1.BudgetPeriod_YEARLY, want: model.BudgetPeriodYearly},
		{name: "custom", input: budgetv1.BudgetPeriod_CUSTOM, want: model.BudgetPeriodCustom},
		{name: "unknown defaults to monthly", input: budgetv1.BudgetPeriod(999), want: model.BudgetPeriodMonthly},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, budgetPeriodFromProto(tt.input))
		})
	}
}

func TestBudgetStatusFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input budgetv1.BudgetStatus
		want  model.BudgetStatus
	}{
		{name: "active", input: budgetv1.BudgetStatus_ACTIVE, want: model.BudgetStatusActive},
		{name: "paused", input: budgetv1.BudgetStatus_PAUSED, want: model.BudgetStatusPaused},
		{name: "completed", input: budgetv1.BudgetStatus_COMPLETED, want: model.BudgetStatusCompleted},
		{name: "cancelled", input: budgetv1.BudgetStatus_CANCELLED, want: model.BudgetStatusCancelled},
		{name: "unknown defaults to active", input: budgetv1.BudgetStatus(999), want: model.BudgetStatusActive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, budgetStatusFromProto(tt.input))
		})
	}
}

func TestBudgetStatusToProtoPtr(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, budgetStatusToProtoPtr(nil))
	})

	tests := []struct {
		name    string
		input   model.BudgetStatus
		want    budgetv1.BudgetStatus
		wantNil bool
	}{
		{name: "active", input: model.BudgetStatusActive, want: budgetv1.BudgetStatus_ACTIVE},
		{name: "paused", input: model.BudgetStatusPaused, want: budgetv1.BudgetStatus_PAUSED},
		{name: "completed", input: model.BudgetStatusCompleted, want: budgetv1.BudgetStatus_COMPLETED},
		{name: "cancelled", input: model.BudgetStatusCancelled, want: budgetv1.BudgetStatus_CANCELLED},
		{name: "unknown returns nil", input: model.BudgetStatus("unknown"), wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.input
			result := budgetStatusToProtoPtr(&s)
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Equal(t, tt.want, *result)
		})
	}
}

func TestCardBrandFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input creditcardv1.CardBrand
		want  model.CardBrand
	}{
		{name: "visa", input: creditcardv1.CardBrand_VISA, want: model.CardBrandVisa},
		{name: "mastercard", input: creditcardv1.CardBrand_MASTERCARD, want: model.CardBrandMastercard},
		{name: "amex", input: creditcardv1.CardBrand_AMEX, want: model.CardBrandAmex},
		{name: "elo", input: creditcardv1.CardBrand_ELO, want: model.CardBrandElo},
		{name: "hipercard", input: creditcardv1.CardBrand_HIPERCARD, want: model.CardBrandHipercard},
		{name: "diners", input: creditcardv1.CardBrand_DINERS, want: model.CardBrandDiners},
		{name: "other", input: creditcardv1.CardBrand_OTHER_BRAND, want: model.CardBrandOtherBrand},
		{name: "unknown defaults to other", input: creditcardv1.CardBrand(999), want: model.CardBrandOtherBrand},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cardBrandFromProto(tt.input))
		})
	}
}

func TestCardTypeFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input creditcardv1.CardType
		want  model.CardType
	}{
		{name: "credit", input: creditcardv1.CardType_CREDIT, want: model.CardTypeCredit},
		{name: "debit", input: creditcardv1.CardType_DEBIT, want: model.CardTypeDebit},
		{name: "multiple", input: creditcardv1.CardType_MULTIPLE, want: model.CardTypeMultiple},
		{name: "unknown defaults to credit", input: creditcardv1.CardType(999), want: model.CardTypeCredit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cardTypeFromProto(tt.input))
		})
	}
}

func TestInvoiceStatusFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input creditcardv1.InvoiceStatus
		want  model.InvoiceStatus
	}{
		{name: "open", input: creditcardv1.InvoiceStatus_OPEN, want: model.InvoiceStatusOpen},
		{name: "closed", input: creditcardv1.InvoiceStatus_CLOSED, want: model.InvoiceStatusClosed},
		{name: "paid", input: creditcardv1.InvoiceStatus_PAID, want: model.InvoiceStatusPaid},
		{name: "overdue", input: creditcardv1.InvoiceStatus_OVERDUE, want: model.InvoiceStatusOverdue},
		{name: "unknown defaults to open", input: creditcardv1.InvoiceStatus(999), want: model.InvoiceStatusOpen},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, invoiceStatusFromProto(tt.input))
		})
	}
}

func TestInvoiceStatusToProtoPtr(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, invoiceStatusToProtoPtr(nil))
	})

	tests := []struct {
		name    string
		input   model.InvoiceStatus
		want    creditcardv1.InvoiceStatus
		wantNil bool
	}{
		{name: "open", input: model.InvoiceStatusOpen, want: creditcardv1.InvoiceStatus_OPEN},
		{name: "closed", input: model.InvoiceStatusClosed, want: creditcardv1.InvoiceStatus_CLOSED},
		{name: "paid", input: model.InvoiceStatusPaid, want: creditcardv1.InvoiceStatus_PAID},
		{name: "overdue", input: model.InvoiceStatusOverdue, want: creditcardv1.InvoiceStatus_OVERDUE},
		{name: "unknown returns nil", input: model.InvoiceStatus("unknown"), wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.input
			result := invoiceStatusToProtoPtr(&s)
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Equal(t, tt.want, *result)
		})
	}
}

func TestDebtTypeFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input debtv1.DebtType
		want  model.DebtType
	}{
		{name: "personal loan", input: debtv1.DebtType_PERSONAL_LOAN, want: model.DebtTypePersonalLoan},
		{name: "student loan", input: debtv1.DebtType_STUDENT_LOAN, want: model.DebtTypeStudentLoan},
		{name: "mortgage", input: debtv1.DebtType_MORTGAGE, want: model.DebtTypeMortgage},
		{name: "car loan", input: debtv1.DebtType_CAR_LOAN, want: model.DebtTypeCarLoan},
		{name: "credit card debt", input: debtv1.DebtType_CREDIT_CARD_DEBT, want: model.DebtTypeCreditCardDebt},
		{name: "medical debt", input: debtv1.DebtType_MEDICAL_DEBT, want: model.DebtTypeMedicalDebt},
		{name: "other", input: debtv1.DebtType_OTHER_DEBT, want: model.DebtTypeOtherDebt},
		{name: "unknown defaults to other", input: debtv1.DebtType(999), want: model.DebtTypeOtherDebt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, debtTypeFromProto(tt.input))
		})
	}
}

func TestDebtStatusFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input debtv1.DebtStatus
		want  model.DebtStatus
	}{
		{name: "active", input: debtv1.DebtStatus_ACTIVE, want: model.DebtStatusActive},
		{name: "paused", input: debtv1.DebtStatus_PAUSED, want: model.DebtStatusPaused},
		{name: "paid off", input: debtv1.DebtStatus_PAID_OFF, want: model.DebtStatusPaidOff},
		{name: "defaulted", input: debtv1.DebtStatus_DEFAULTED, want: model.DebtStatusDefaulted},
		{name: "settled", input: debtv1.DebtStatus_SETTLED, want: model.DebtStatusSettled},
		{name: "unknown defaults to active", input: debtv1.DebtStatus(999), want: model.DebtStatusActive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, debtStatusFromProto(tt.input))
		})
	}
}

func TestDebtStatusToProtoPtr(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, debtStatusToProtoPtr(nil))
	})

	tests := []struct {
		name    string
		input   model.DebtStatus
		want    debtv1.DebtStatus
		wantNil bool
	}{
		{name: "active", input: model.DebtStatusActive, want: debtv1.DebtStatus_ACTIVE},
		{name: "paused", input: model.DebtStatusPaused, want: debtv1.DebtStatus_PAUSED},
		{name: "paid off", input: model.DebtStatusPaidOff, want: debtv1.DebtStatus_PAID_OFF},
		{name: "defaulted", input: model.DebtStatusDefaulted, want: debtv1.DebtStatus_DEFAULTED},
		{name: "settled", input: model.DebtStatusSettled, want: debtv1.DebtStatus_SETTLED},
		{name: "unknown returns nil", input: model.DebtStatus("unknown"), wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.input
			result := debtStatusToProtoPtr(&s)
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Equal(t, tt.want, *result)
		})
	}
}

func TestDebtTypeToProtoPtr(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, debtTypeToProtoPtr(nil))
	})

	tests := []struct {
		name    string
		input   model.DebtType
		want    debtv1.DebtType
		wantNil bool
	}{
		{name: "personal loan", input: model.DebtTypePersonalLoan, want: debtv1.DebtType_PERSONAL_LOAN},
		{name: "student loan", input: model.DebtTypeStudentLoan, want: debtv1.DebtType_STUDENT_LOAN},
		{name: "mortgage", input: model.DebtTypeMortgage, want: debtv1.DebtType_MORTGAGE},
		{name: "car loan", input: model.DebtTypeCarLoan, want: debtv1.DebtType_CAR_LOAN},
		{name: "credit card debt", input: model.DebtTypeCreditCardDebt, want: debtv1.DebtType_CREDIT_CARD_DEBT},
		{name: "medical debt", input: model.DebtTypeMedicalDebt, want: debtv1.DebtType_MEDICAL_DEBT},
		{name: "other", input: model.DebtTypeOtherDebt, want: debtv1.DebtType_OTHER_DEBT},
		{name: "unknown returns nil", input: model.DebtType("unknown"), wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp := tt.input
			result := debtTypeToProtoPtr(&tp)
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Equal(t, tt.want, *result)
		})
	}
}

func TestAssetTypeFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input investmentv1.AssetType
		want  model.AssetType
	}{
		{name: "stock", input: investmentv1.AssetType_STOCK, want: model.AssetTypeStock},
		{name: "etf", input: investmentv1.AssetType_ETF, want: model.AssetTypeEtf},
		{name: "real estate fund", input: investmentv1.AssetType_REAL_ESTATE_FUND, want: model.AssetTypeRealEstateFund},
		{name: "treasury", input: investmentv1.AssetType_TREASURY, want: model.AssetTypeTreasury},
		{name: "cdb", input: investmentv1.AssetType_CDB, want: model.AssetTypeCdb},
		{name: "lci", input: investmentv1.AssetType_LCI, want: model.AssetTypeLci},
		{name: "lca", input: investmentv1.AssetType_LCA, want: model.AssetTypeLca},
		{name: "crypto", input: investmentv1.AssetType_CRYPTO, want: model.AssetTypeCrypto},
		{name: "pension", input: investmentv1.AssetType_PENSION, want: model.AssetTypePension},
		{name: "fund", input: investmentv1.AssetType_FUND, want: model.AssetTypeFund},
		{name: "dollar", input: investmentv1.AssetType_DOLLAR, want: model.AssetTypeDollar},
		{name: "gold", input: investmentv1.AssetType_GOLD, want: model.AssetTypeGold},
		{name: "other", input: investmentv1.AssetType_OTHER_ASSET, want: model.AssetTypeOtherAsset},
		{name: "unknown defaults to other", input: investmentv1.AssetType(999), want: model.AssetTypeOtherAsset},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, assetTypeFromProto(tt.input))
		})
	}
}

func TestTransactionTypeFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input investmentv1.TransactionType
		want  model.InvestmentTransactionType
	}{
		{name: "buy", input: investmentv1.TransactionType_BUY, want: model.InvestmentTransactionTypeBuy},
		{name: "sell", input: investmentv1.TransactionType_SELL, want: model.InvestmentTransactionTypeSell},
		{name: "dividend", input: investmentv1.TransactionType_DIVIDEND, want: model.InvestmentTransactionTypeDividend},
		{name: "jcp", input: investmentv1.TransactionType_JCP, want: model.InvestmentTransactionTypeJcp},
		{name: "amortization", input: investmentv1.TransactionType_AMORTIZATION, want: model.InvestmentTransactionTypeAmortization},
		{name: "unknown defaults to buy", input: investmentv1.TransactionType(999), want: model.InvestmentTransactionTypeBuy},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, transactionTypeFromProto(tt.input))
		})
	}
}

func TestInvestmentStatusFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input investmentv1.InvestmentStatus
		want  model.InvestmentStatus
	}{
		{name: "active", input: investmentv1.InvestmentStatus_ACTIVE, want: model.InvestmentStatusActive},
		{name: "sold", input: investmentv1.InvestmentStatus_SOLD, want: model.InvestmentStatusSold},
		{name: "cancelled", input: investmentv1.InvestmentStatus_CANCELLED, want: model.InvestmentStatusCancelled},
		{name: "unknown defaults to active", input: investmentv1.InvestmentStatus(999), want: model.InvestmentStatusActive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, investmentStatusFromProto(tt.input))
		})
	}
}

func TestInvestmentStatusToProtoPtr(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, investmentStatusToProtoPtr(nil))
	})

	tests := []struct {
		name    string
		input   model.InvestmentStatus
		want    investmentv1.InvestmentStatus
		wantNil bool
	}{
		{name: "active", input: model.InvestmentStatusActive, want: investmentv1.InvestmentStatus_ACTIVE},
		{name: "sold", input: model.InvestmentStatusSold, want: investmentv1.InvestmentStatus_SOLD},
		{name: "cancelled", input: model.InvestmentStatusCancelled, want: investmentv1.InvestmentStatus_CANCELLED},
		{name: "unknown returns nil", input: model.InvestmentStatus("unknown"), wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.input
			result := investmentStatusToProtoPtr(&s)
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Equal(t, tt.want, *result)
		})
	}
}

func TestAssetTypeToProtoPtr(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, assetTypeToProtoPtr(nil))
	})

	tests := []struct {
		name    string
		input   model.AssetType
		want    investmentv1.AssetType
		wantNil bool
	}{
		{name: "stock", input: model.AssetTypeStock, want: investmentv1.AssetType_STOCK},
		{name: "etf", input: model.AssetTypeEtf, want: investmentv1.AssetType_ETF},
		{name: "other asset", input: model.AssetTypeOtherAsset, want: investmentv1.AssetType_OTHER_ASSET},
		{name: "unknown returns nil", input: model.AssetType("unknown"), wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp := tt.input
			result := assetTypeToProtoPtr(&tp)
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Equal(t, tt.want, *result)
		})
	}
}

func TestTransactionTypeToProtoPtr(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, transactionTypeToProtoPtr(nil))
	})

	tests := []struct {
		name    string
		input   model.InvestmentTransactionType
		want    investmentv1.TransactionType
		wantNil bool
	}{
		{name: "buy", input: model.InvestmentTransactionTypeBuy, want: investmentv1.TransactionType_BUY},
		{name: "sell", input: model.InvestmentTransactionTypeSell, want: investmentv1.TransactionType_SELL},
		{name: "dividend", input: model.InvestmentTransactionTypeDividend, want: investmentv1.TransactionType_DIVIDEND},
		{name: "jcp", input: model.InvestmentTransactionTypeJcp, want: investmentv1.TransactionType_JCP},
		{name: "amortization", input: model.InvestmentTransactionTypeAmortization, want: investmentv1.TransactionType_AMORTIZATION},
		{name: "unknown returns nil", input: model.InvestmentTransactionType("unknown"), wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp := tt.input
			result := transactionTypeToProtoPtr(&tp)
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Equal(t, tt.want, *result)
		})
	}
}

// ── Proto Converter Tests ─────────────────────────────────────────────────

func TestIncomeFromProto(t *testing.T) {
	pb := testIncomeProto("inc-1")
	result := incomeFromProto(pb)

	assert.Equal(t, "inc-1", result.ID)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, "Freelance project", result.Description)
	assert.Equal(t, "Upwork", result.Source)
	assert.Equal(t, model.IncomeTypeFreelance, result.IncomeType)
	assert.Equal(t, int64(500000), result.ReceivedAmount)
	assert.Equal(t, model.TransactionStatusCompleted, result.Status)
	assert.NotZero(t, result.CreatedAt)
	assert.NotZero(t, result.UpdatedAt)
}

func TestFixedExpenseFromProto(t *testing.T) {
	pb := testFixedExpenseProto("fe-1")
	result := fixedExpenseFromProto(pb)

	assert.Equal(t, "fe-1", result.ID)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, "Rent", result.Description)
	assert.Equal(t, "Housing", result.Category)
	assert.Equal(t, 5, result.DayOfMonth)
	assert.Equal(t, model.PaymentMethodBankTransfer, result.PaymentMethod)
	assert.Equal(t, model.TransactionStatusPending, result.Status)
}

func TestVariableExpenseFromProto(t *testing.T) {
	pb := testVariableExpenseProto("ve-1")
	result := variableExpenseFromProto(pb)

	assert.Equal(t, "ve-1", result.ID)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, "Groceries", result.Description)
	assert.Equal(t, "Food", result.Category)
	assert.Equal(t, model.ExpenseTypeEssential, result.ExpenseType)
	assert.Equal(t, int64(15000), result.PaidAmount)
	assert.Equal(t, model.TransactionStatusCompleted, result.Status)
}

func TestBudgetFromProto(t *testing.T) {
	pb := testBudgetProto("bgt-1")
	result := budgetFromProto(pb)

	assert.Equal(t, "bgt-1", result.ID)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, "Monthly Budget", result.Name)
	assert.Equal(t, "Main monthly budget", result.Description)
	assert.Equal(t, model.BudgetPeriodMonthly, result.Period)
	assert.Equal(t, int64(500000), result.TotalLimit)
	assert.Equal(t, int64(250000), result.SpentAmount)
	assert.Equal(t, model.BudgetStatusActive, result.Status)
	assert.NotZero(t, result.StartDate)
	assert.NotZero(t, result.EndDate)
	assert.Len(t, result.Categories, 1)
	assert.NotZero(t, result.CreatedAt)
	assert.NotZero(t, result.UpdatedAt)
}

func TestBudgetCategoryFromProto(t *testing.T) {
	pb := &budgetv1.BudgetCategory{
		Id:          "cat-1",
		BudgetId:    "bgt-1",
		Name:        "Food",
		LimitAmount: 200000,
		SpentAmount: 100000,
		Category:    "Food",
	}
	result := budgetCategoryFromProto(pb)

	assert.Equal(t, "cat-1", result.ID)
	assert.Equal(t, "bgt-1", result.BudgetID)
	assert.Equal(t, "Food", result.Name)
	assert.Equal(t, int64(200000), result.LimitAmount)
	assert.Equal(t, int64(100000), result.SpentAmount)
	assert.Equal(t, "Food", result.Category)
}

func TestBudgetSummaryFromProto(t *testing.T) {
	pb := testBudgetSummaryProto()
	result := budgetSummaryFromProto(pb)

	assert.Equal(t, "bgt-1", result.BudgetID)
	assert.Equal(t, int64(500000), result.TotalLimit)
	assert.Equal(t, int64(250000), result.TotalSpent)
	assert.Equal(t, int64(250000), result.Remaining)
	assert.Equal(t, 50.0, result.UsagePercentage)
	assert.Equal(t, 1, result.CategoryCount)
	assert.Len(t, result.Categories, 1)
}

func TestCategorySummaryFromProto(t *testing.T) {
	pb := &budgetv1.CategorySummary{
		CategoryId:      "cat-1",
		Name:            "Food",
		Category:        "Food",
		LimitAmount:     200000,
		SpentAmount:     100000,
		Remaining:       100000,
		UsagePercentage: 50.0,
	}
	result := categorySummaryFromProto(pb)

	assert.Equal(t, "cat-1", result.CategoryID)
	assert.Equal(t, "Food", result.Name)
	assert.Equal(t, "Food", result.Category)
	assert.Equal(t, int64(200000), result.LimitAmount)
	assert.Equal(t, int64(100000), result.SpentAmount)
	assert.Equal(t, int64(100000), result.Remaining)
	assert.Equal(t, 50.0, result.UsagePercentage)
}

func TestCreditCardFromProto(t *testing.T) {
	pb := testCreditCardProto("ccc-1")
	result := creditCardFromProto(pb)

	assert.Equal(t, "ccc-1", result.ID)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, "My Card", result.Name)
	assert.Equal(t, model.CardBrandVisa, result.Brand)
	assert.Equal(t, model.CardTypeCredit, result.CardType)
	assert.Equal(t, "1234", result.LastFourDigits)
	assert.Equal(t, 15, result.ClosingDay)
	assert.Equal(t, 5, result.DueDay)
	assert.Equal(t, int64(1000000), result.CreditLimit)
	assert.Equal(t, int64(500000), result.AvailableCredit)
	assert.True(t, result.Active)
	assert.NotZero(t, result.CreatedAt)
	assert.NotZero(t, result.UpdatedAt)
}

func TestInvoiceFromProto(t *testing.T) {
	pb := testInvoiceProto("inv-1")
	result := invoiceFromProto(pb)

	assert.Equal(t, "inv-1", result.ID)
	assert.Equal(t, "ccc-1", result.CreditCardID)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, "2024-01", result.ReferenceMonth)
	assert.Equal(t, int64(500000), result.TotalAmount)
	assert.Equal(t, int64(500000), result.PaidAmount)
	assert.Equal(t, model.InvoiceStatusPaid, result.Status)
	assert.NotZero(t, result.ClosingDate)
	assert.NotZero(t, result.DueDate)
	assert.NotZero(t, result.CreatedAt)
	assert.NotZero(t, result.UpdatedAt)
}

func TestInvoiceTransactionFromProto(t *testing.T) {
	pb := testInvoiceTransactionProto("tx-1")
	result := invoiceTransactionFromProto(pb)

	assert.Equal(t, "tx-1", result.ID)
	assert.Equal(t, "inv-1", result.InvoiceID)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, "Restaurant", result.Description)
	assert.Equal(t, int64(50000), result.Amount)
	assert.Equal(t, "Food", result.Category)
	assert.NotZero(t, result.TransactionDate)
	assert.Equal(t, 1, result.Installments)
	assert.NotZero(t, result.CreatedAt)
}

func TestDebtFromProto(t *testing.T) {
	pb := testDebtProto("debt-1")
	result := debtFromProto(pb)

	assert.Equal(t, "debt-1", result.ID)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, "Student Loan", result.Name)
	assert.Equal(t, "University loan", result.Description)
	assert.Equal(t, model.DebtTypeStudentLoan, result.DebtType)
	assert.Equal(t, int64(5000000), result.TotalAmount)
	assert.Equal(t, int64(3000000), result.RemainingAmount)
	assert.Equal(t, int64(500), result.InterestRate)
	assert.NotZero(t, result.StartDate)
	assert.NotZero(t, result.ExpectedEndDate)
	assert.Equal(t, model.DebtStatusActive, result.Status)
	assert.Equal(t, "Bank", result.Creditor)
	assert.NotZero(t, result.CreatedAt)
	assert.NotZero(t, result.UpdatedAt)
}

func TestPaymentFromProto(t *testing.T) {
	pb := testPaymentProto("pmt-1")
	result := paymentFromProto(pb)

	assert.Equal(t, "pmt-1", result.ID)
	assert.Equal(t, "debt-1", result.DebtID)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, int64(100000), result.Amount)
	assert.NotZero(t, result.PaymentDate)
	assert.Equal(t, "Monthly payment", result.Notes)
	assert.NotZero(t, result.CreatedAt)
}

func TestInvestmentFromProto(t *testing.T) {
	pb := testInvestmentProto("inv-1")
	result := investmentFromProto(pb)

	assert.Equal(t, "inv-1", result.ID)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, "Tech Stock", result.Name)
	assert.Equal(t, "TECH", result.Ticker)
	assert.Equal(t, model.AssetTypeStock, result.AssetType)
	assert.Equal(t, int64(100), result.Quantity)
	assert.Equal(t, int64(5000), result.AveragePrice)
	assert.Equal(t, int64(500000), result.TotalInvested)
	assert.Equal(t, model.InvestmentStatusActive, result.Status)
	assert.Equal(t, "BrokerX", result.Broker)
	assert.NotZero(t, result.CreatedAt)
	assert.NotZero(t, result.UpdatedAt)
}

func TestInvestmentTransactionFromProto(t *testing.T) {
	pb := testInvestmentTransactionProto("tx-1")
	result := investmentTransactionFromProto(pb)

	assert.Equal(t, "tx-1", result.ID)
	assert.Equal(t, "inv-1", result.InvestmentID)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, model.InvestmentTransactionTypeBuy, result.TransactionType)
	assert.Equal(t, int64(10), result.Quantity)
	assert.Equal(t, int64(5000), result.UnitPrice)
	assert.Equal(t, int64(50000), result.TotalAmount)
	assert.NotZero(t, result.TransactionDate)
	assert.Equal(t, "Purchase", result.Notes)
	assert.NotZero(t, result.CreatedAt)
}

func TestPortfolioSummaryFromProto(t *testing.T) {
	pb := testPortfolioSummaryProto()
	result := portfolioSummaryFromProto(pb)

	assert.Equal(t, int64(1000000), result.TotalInvested)
	assert.Equal(t, int64(1200000), result.CurrentValue)
	assert.Equal(t, int64(200000), result.TotalReturn)
	assert.Equal(t, 20.0, result.ReturnPercentage)
	assert.Equal(t, 5, result.ActiveInvestments)
	assert.Len(t, result.Allocation, 2)
}

func TestAssetAllocationFromProto(t *testing.T) {
	pb := &investmentv1.AssetAllocation{
		AssetType:    investmentv1.AssetType_STOCK,
		Invested:     500000,
		CurrentValue: 600000,
		Percentage:   50.0,
	}
	result := assetAllocationFromProto(pb)

	assert.Equal(t, model.AssetTypeStock, result.AssetType)
	assert.Equal(t, int64(500000), result.Invested)
	assert.Equal(t, int64(600000), result.CurrentValue)
	assert.Equal(t, 50.0, result.Percentage)
}

// ── isFeatureEnabled Tests ─────────────────────────────────────────────────

func TestIsFeatureEnabled(t *testing.T) {
	t.Run("nil feature flag client returns false", func(t *testing.T) {
		r := &Resolver{FFClient: nil}
		assert.False(t, r.isFeatureEnabled(context.Background(), "test-flag"))
	})
}

// ── cachedSingle (no cache) Tests ──────────────────────────────────────────

func TestCachedSingle_NoCache(t *testing.T) {
	resolver, mockTx, _, _, _, _, _ := setupTestResolver(t)
	mockTx.incomes["inc-1"] = testIncomeProto("inc-1")

	ctx := ctxWithUser("user-123")
	q := resolver.Query().(*queryResolver)

	var result model.Income
	err := q.cachedSingle(ctx, "income", "inc-1", &result, func() (interface{}, error) {
		pb, err := resolver.TxClient.GetIncome(ctx, &transactionv1.GetIncomeRequest{Id: "inc-1"})
		if err != nil {
			return nil, err
		}
		return incomeFromProto(pb), nil
	})

	require.NoError(t, err)
	assert.Equal(t, "inc-1", result.ID)
}

func TestCachedSingle_NoCache_FallbackError(t *testing.T) {
	resolver, _, _, _, _, _, _ := setupTestResolver(t)

	ctx := ctxWithUser("user-123")
	q := resolver.Query().(*queryResolver)

	var result model.Income
	err := q.cachedSingle(ctx, "income", "notfound", &result, func() (interface{}, error) {
		return nil, fmt.Errorf("fallback error")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fallback error")
}

func TestCachedSingle_NoCache_MarshalError(t *testing.T) {
	resolver, _, _, _, _, _, _ := setupTestResolver(t)

	ctx := ctxWithUser("user-123")
	q := resolver.Query().(*queryResolver)

	// A channel can't be marshaled to JSON
	var result model.Income
	err := q.cachedSingle(ctx, "income", "bad", &result, func() (interface{}, error) {
		return make(chan int), nil
	})

	assert.Error(t, err)
}

// ── cachedList (no cache) Tests ────────────────────────────────────────────

func TestCachedList_NoCache(t *testing.T) {
	resolver, mockTx, _, _, _, _, _ := setupTestResolver(t)
	mockTx.incomes["inc-1"] = testIncomeProto("inc-1")

	ctx := ctxWithUser("user-123")
	q := resolver.Query().(*queryResolver)

	var result model.IncomeConnection
	err := q.cachedList(ctx, "incomes", struct{}{}, &result, func() (interface{}, error) {
		pb, err := resolver.TxClient.ListIncomes(ctx, &transactionv1.ListIncomesRequest{})
		if err != nil {
			return nil, err
		}
		edges := make([]*model.IncomeEdge, len(pb.Incomes))
		for i, inc := range pb.Incomes {
			edges[i] = &model.IncomeEdge{
				Node:   incomeFromProto(inc),
				Cursor: fmt.Sprintf("%d", i),
			}
		}
		return &model.IncomeConnection{
			Edges:      edges,
			TotalCount: int(pb.TotalCount),
		}, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
}

func TestCachedList_NoCache_FallbackError(t *testing.T) {
	resolver, _, _, _, _, _, _ := setupTestResolver(t)

	ctx := ctxWithUser("user-123")
	q := resolver.Query().(*queryResolver)

	var result model.IncomeConnection
	err := q.cachedList(ctx, "incomes", struct{}{}, &result, func() (interface{}, error) {
		return nil, fmt.Errorf("fallback error")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fallback error")
}

func TestCachedList_NoCache_MarshalError(t *testing.T) {
	resolver, _, _, _, _, _, _ := setupTestResolver(t)

	ctx := ctxWithUser("user-123")
	q := resolver.Query().(*queryResolver)

	// A channel can't be marshaled to JSON
	var result model.IncomeConnection
	err := q.cachedList(ctx, "incomes", struct{}{}, &result, func() (interface{}, error) {
		return make(chan int), nil
	})

	assert.Error(t, err)
}
