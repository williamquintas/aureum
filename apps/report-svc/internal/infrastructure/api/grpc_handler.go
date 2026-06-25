package api

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pkgErr "github.com/aureum/pkg/errors"
	reportv1 "github.com/aureum/proto/gen/report/reportv1"
	"github.com/aureum/report-svc/internal/application"
	"github.com/aureum/report-svc/internal/domain"
)

type GRPCHandler struct {
	reportv1.UnimplementedReportServiceServer
	svc *application.Service
}

func NewGRPCHandler(svc *application.Service) *GRPCHandler {
	return &GRPCHandler{svc: svc}
}

func (h *GRPCHandler) GetIncomeStatement(ctx context.Context, req *reportv1.IncomeStatementRequest) (*reportv1.IncomeStatementResponse, error) {
	userID := mustExtractUserID(ctx)

	dateFrom := protoTimeToString(req.DateFrom)
	dateTo := protoTimeToString(req.DateTo)

	resp, err := h.svc.GetIncomeStatement(ctx, application.IncomeStatementRequest{
		UserID:   userID,
		DateFrom: dateFrom,
		DateTo:   dateTo,
		GroupBy:  req.GroupBy,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return incomeStatementToProto(resp), nil
}

func (h *GRPCHandler) GetExpenseSummary(ctx context.Context, req *reportv1.ExpenseSummaryRequest) (*reportv1.ExpenseSummaryResponse, error) {
	userID := mustExtractUserID(ctx)

	dateFrom := protoTimeToString(req.DateFrom)
	dateTo := protoTimeToString(req.DateTo)

	resp, err := h.svc.GetExpenseSummary(ctx, application.ExpenseSummaryRequest{
		UserID:   userID,
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Type:     req.Type,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return expenseSummaryToProto(resp), nil
}

func (h *GRPCHandler) GetBudgetVsActual(ctx context.Context, req *reportv1.BudgetVsActualRequest) (*reportv1.BudgetVsActualResponse, error) {
	userID := mustExtractUserID(ctx)

	dateFrom := protoTimeToString(req.DateFrom)
	dateTo := protoTimeToString(req.DateTo)

	resp, err := h.svc.GetBudgetVsActual(ctx, application.BudgetVsActualRequest{
		UserID:   userID,
		BudgetID: req.BudgetId,
		DateFrom: dateFrom,
		DateTo:   dateTo,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return budgetVsActualToProto(resp), nil
}

func (h *GRPCHandler) GetSpendingTrends(ctx context.Context, req *reportv1.SpendingTrendsRequest) (*reportv1.SpendingTrendsResponse, error) {
	userID := mustExtractUserID(ctx)

	resp, err := h.svc.GetSpendingTrends(ctx, application.SpendingTrendsRequest{
		UserID:   userID,
		Months:   req.Months,
		Category: req.Category,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return spendingTrendsToProto(resp), nil
}

func (h *GRPCHandler) GetPortfolioPerformance(ctx context.Context, req *reportv1.PortfolioPerformanceRequest) (*reportv1.PortfolioPerformanceResponse, error) {
	userID := mustExtractUserID(ctx)

	dateFrom := protoTimeToString(req.DateFrom)
	dateTo := protoTimeToString(req.DateTo)

	resp, err := h.svc.GetPortfolioPerformance(ctx, application.PortfolioPerformanceRequest{
		UserID:       userID,
		InvestmentID: req.InvestmentId,
		DateFrom:     dateFrom,
		DateTo:       dateTo,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return portfolioPerformanceToProto(resp), nil
}

func (h *GRPCHandler) GetFinancialOverview(ctx context.Context, req *reportv1.FinancialOverviewRequest) (*reportv1.FinancialOverviewResponse, error) {
	userID := mustExtractUserID(ctx)

	resp, err := h.svc.GetFinancialOverview(ctx, application.FinancialOverviewRequest{
		UserID: userID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return financialOverviewToProto(resp), nil
}

// ── Proto → DTO converters ─────────────────────────────────────────────────

func protoTimeToString(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().Format("2006-01-02")
}

// ── DTO → Proto converters ─────────────────────────────────────────────────

func moneyToProto(m application.MoneyDTO) *reportv1.Money {
	return &reportv1.Money{
		Cents:    m.Cents,
		Currency: m.Currency,
	}
}

func categoryAmountToProto(ca application.CategoryAmountDTO) *reportv1.CategoryAmount {
	return &reportv1.CategoryAmount{
		Category: ca.Category,
		Amount: &reportv1.Money{
			Cents:    ca.Amount,
			Currency: "USD",
		},
		TransactionCount: ca.TransactionCount,
	}
}

func incomeStatementToProto(resp *application.IncomeStatementResponse) *reportv1.IncomeStatementResponse {
	periods := make([]*reportv1.IncomePeriod, len(resp.Periods))
	for i, p := range resp.Periods {
		cats := make([]*reportv1.CategoryAmount, len(p.Categories))
		for j, c := range p.Categories {
			cats[j] = categoryAmountToProto(c)
		}
		periods[i] = &reportv1.IncomePeriod{
			Period:      p.Period,
			Categories:  cats,
			PeriodTotal: moneyToProto(p.PeriodTotal),
		}
	}
	return &reportv1.IncomeStatementResponse{
		Periods:     periods,
		TotalIncome: moneyToProto(resp.TotalIncome),
	}
}

func expenseSummaryToProto(resp *application.ExpenseSummaryResponse) *reportv1.ExpenseSummaryResponse {
	fixedCats := make([]*reportv1.CategoryAmount, len(resp.FixedCategories))
	for i, c := range resp.FixedCategories {
		fixedCats[i] = categoryAmountToProto(c)
	}
	varCats := make([]*reportv1.CategoryAmount, len(resp.VariableCategories))
	for i, c := range resp.VariableCategories {
		varCats[i] = categoryAmountToProto(c)
	}
	return &reportv1.ExpenseSummaryResponse{
		TotalFixed:         moneyToProto(resp.TotalFixed),
		TotalVariable:      moneyToProto(resp.TotalVariable),
		TotalAll:           moneyToProto(resp.TotalAll),
		FixedCategories:    fixedCats,
		VariableCategories: varCats,
	}
}

func budgetVsActualToProto(resp *application.BudgetVsActualResponse) *reportv1.BudgetVsActualResponse {
	cats := make([]*reportv1.BudgetCategoryComparison, len(resp.Categories))
	for i, c := range resp.Categories {
		cats[i] = &reportv1.BudgetCategoryComparison{
			Category:           c.Category,
			Budgeted:           moneyToProto(c.Budgeted),
			Actual:             moneyToProto(c.Actual),
			Variance:           moneyToProto(c.Variance),
			VariancePercentage: c.VariancePct,
		}
	}
	return &reportv1.BudgetVsActualResponse{
		Categories:         cats,
		TotalBudgeted:      moneyToProto(resp.TotalBudgeted),
		TotalActual:        moneyToProto(resp.TotalActual),
		TotalVariance:      moneyToProto(resp.TotalVariance),
		VariancePercentage: resp.VariancePercentage,
	}
}

func spendingTrendsToProto(resp *application.SpendingTrendsResponse) *reportv1.SpendingTrendsResponse {
	trends := make([]*reportv1.MonthlyTrend, len(resp.Trends))
	for i, t := range resp.Trends {
		trends[i] = &reportv1.MonthlyTrend{
			Month:   t.Month,
			Amount:  moneyToProto(t.Amount),
			Average: moneyToProto(t.Average),
		}
	}
	return &reportv1.SpendingTrendsResponse{
		Trends:         trends,
		TrendDirection: resp.TrendDirection,
	}
}

func portfolioPerformanceToProto(resp *application.PortfolioPerformanceResponse) *reportv1.PortfolioPerformanceResponse {
	assets := make([]*reportv1.AssetPerformance, len(resp.Assets))
	for i, a := range resp.Assets {
		assets[i] = &reportv1.AssetPerformance{
			AssetType:            a.AssetType,
			Invested:             moneyToProto(a.Invested),
			CurrentValue:         moneyToProto(a.CurrentValue),
			ReturnPercentage:     a.ReturnPercentage,
			AllocationPercentage: a.AllocationPct,
		}
	}
	return &reportv1.PortfolioPerformanceResponse{
		TotalInvested:    moneyToProto(resp.TotalInvested),
		CurrentValue:     moneyToProto(resp.CurrentValue),
		TotalReturn:      moneyToProto(resp.TotalReturn),
		ReturnPercentage: resp.ReturnPercentage,
		Assets:           assets,
	}
}

func financialOverviewToProto(resp *application.FinancialOverviewResponse) *reportv1.FinancialOverviewResponse {
	return &reportv1.FinancialOverviewResponse{
		TotalMonthlyIncome:        moneyToProto(resp.TotalMonthlyIncome),
		TotalMonthlyExpenses:      moneyToProto(resp.TotalMonthlyExpenses),
		NetSavings:                moneyToProto(resp.NetSavings),
		TotalDebt:                 moneyToProto(resp.TotalDebt),
		TotalInvestments:          moneyToProto(resp.TotalInvestments),
		BudgetAdherencePercentage: resp.BudgetAdherencePct,
		ActiveBudgets:             resp.ActiveBudgets,
	}
}

// ── Error mapping ──────────────────────────────────────────────────────────

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
	if grpcErr := pkgErr.MapToGRPC(err); status.Code(grpcErr) != codes.Unknown {
		return grpcErr
	}
	switch {
	case errors.Is(err, domain.ErrNoData):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrMissingField):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidDateRange):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrAccessDenied):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
