# Aureum Domain Model

## Bounded Contexts

| Service | Domain | Aggregate Root | Event Store |
|---------|--------|---------------|-------------|
| identity-svc | User & Auth | User | user_events |
| transaction-svc | Financial Transactions | Transaction | transaction_events |
| budget-svc | Budget Planning | Budget | budget_events |
| creditcard-svc | Credit Card Management | CreditCard | creditcard_events |
| debt-svc | Debt Tracking | Debt | debt_events |
| investment-svc | Investment Portfolio | Investment | investment_events |
| report-svc | Reporting & Analytics | Report | report_events |
| graphql-bff | API Gateway | - | - |

## Core Domain Events

- `UserRegistered`, `UserLoggedIn`, `UserProfileUpdated`
- `TransactionCreated`, `TransactionCategorized`, `TransactionDeleted`
- `BudgetCreated`, `BudgetAllocationChanged`, `BudgetThresholdReached`
- `CreditCardLinked`, `StatementGenerated`, `PaymentScheduled`
- `DebtRecorded`, `DebtRepaymentMade`, `DebtSettled`
- `InvestmentAdded`, `InvestmentValueChanged`, `DividendReceived`

## Value Objects

- `Money{Amount, Currency}` — no negative amounts
- `CPF` — validated Brazilian taxpayer ID
- `Email` — validated email address
- `DateRange{Start, End}` — end must be after start
- `Percentage` — 0.0–100.0 range
- `Category` — user-defined or system default
