package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aureum/report-svc/internal/application"
	"github.com/aureum/report-svc/internal/domain"
)

type EventHandler struct {
	monthlyProjector    *application.MonthlySummaryProjector
	categoryProjector   *application.CategorySummaryProjector
	budgetProjector     *application.BudgetVsActualProjector
	portfolioProjector  *application.PortfolioSnapshotProjector
	debtProjector       *application.DebtSummaryProjector
	creditCardProjector *application.CreditCardSummaryProjector
}

func NewEventHandler(
	monthly *application.MonthlySummaryProjector,
	category *application.CategorySummaryProjector,
	budget *application.BudgetVsActualProjector,
	portfolio *application.PortfolioSnapshotProjector,
	debt *application.DebtSummaryProjector,
	creditCard *application.CreditCardSummaryProjector,
) *EventHandler {
	return &EventHandler{
		monthlyProjector:    monthly,
		categoryProjector:   category,
		budgetProjector:     budget,
		portfolioProjector:  portfolio,
		debtProjector:       debt,
		creditCardProjector: creditCard,
	}
}

func (h *EventHandler) HandleMessage(ctx context.Context, msg []byte) error {
	var event domain.ReportEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	slog.Info("processing event", "type", event.Type, "entity_id", event.EntityID, "user_id", event.UserID)

	switch event.Type {
	case domain.EventIncomeCreated, domain.EventIncomeDeleted,
		domain.EventFixedExpenseCreated, domain.EventFixedExpenseDeleted,
		domain.EventVariableExpenseCreated, domain.EventVariableExpenseDeleted:
		if err := h.monthlyProjector.Handle(ctx, event); err != nil {
			return fmt.Errorf("monthly projector: %w", err)
		}
		if err := h.categoryProjector.Handle(ctx, event); err != nil {
			return fmt.Errorf("category projector: %w", err)
		}

	case domain.EventIncomeUpdated, domain.EventFixedExpenseUpdated, domain.EventVariableExpenseUpdated:
		if err := h.monthlyProjector.Handle(ctx, event); err != nil {
			return fmt.Errorf("monthly projector: %w", err)
		}

	case domain.EventBudgetCreated, domain.EventBudgetUpdated, domain.EventBudgetDeleted:
		if err := h.budgetProjector.Handle(ctx, event); err != nil {
			return fmt.Errorf("budget projector: %w", err)
		}

	case domain.EventInvestmentCreated, domain.EventInvestmentUpdated, domain.EventInvestmentDeleted,
		domain.EventPortfolioCreated, domain.EventPortfolioUpdated:
		if err := h.portfolioProjector.Handle(ctx, event); err != nil {
			return fmt.Errorf("portfolio projector: %w", err)
		}

	case domain.EventDebtCreated, domain.EventDebtUpdated, domain.EventDebtDeleted:
		if err := h.debtProjector.Handle(ctx, event); err != nil {
			return fmt.Errorf("debt projector: %w", err)
		}

	case domain.EventCreditCardCreated, domain.EventCreditCardUpdated, domain.EventCreditCardDeleted:
		if err := h.creditCardProjector.Handle(ctx, event); err != nil {
			return fmt.Errorf("credit card projector: %w", err)
		}

	default:
		slog.Warn("unknown event type", "type", event.Type)
	}

	return nil
}
