# Feature Specification: Transactions Service & GraphQL BFF

**Feature Branch**: `001-transactions-service`

**Created**: 2026-05-28

**Status**: Draft

**Input**: User description: "Implementar o serviço transactions com três tipos de lançamento: 1. entrada 2. fixa 3. variável. Implementar graphql-bff com leitura desse serviço (e de identity?)"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Record and Track Income (Priority: P1)

A user records their income entries (e.g., salary, freelance payments) with details about source, income type, received date, and amount. They can view, update, and track the status of each income record.

**Why this priority**: Income tracking is a fundamental personal finance operation. Without it, the service has no value.

**Independent Test**: Can be fully tested by creating an income record with valid fields and retrieving it back, verifying all fields are preserved.

**Acceptance Scenarios**:

1. **Given** a user wants to record income, **When** they submit a valid income record with description, source, income type, received date, received amount, and status, **Then** the record is created and returned with a unique identifier
2. **Given** a user submits an income record with missing required fields, **When** validation runs, **Then** an error is returned listing which fields are missing
3. **Given** an existing income record, **When** the user updates its status, **Then** the record reflects the new status
4. **Given** an existing income record, **When** the user queries for that record, **Then** all fields are returned correctly
5. **Given** multiple income records exist, **When** the user lists records with optional date range filters, **Then** only matching records are returned

---

### User Story 2 - Manage Fixed Expenses (Priority: P1)

A user registers their recurring fixed expenses (e.g., rent, subscriptions) specifying the day of month for payment, category, and payment method. These repeat monthly.

**Why this priority**: Fixed expense management is equally critical alongside income for financial tracking.

**Independent Test**: Can be fully tested by creating a fixed expense record with all fields and confirming it can be retrieved.

**Acceptance Scenarios**:

1. **Given** a user wants to register a fixed expense, **When** they submit valid fields (description, category, day of month, payment method, status), **Then** the record is created successfully
2. **Given** a user submits a fixed expense with day_of_month = 32, **When** validation runs, **Then** an error is returned for invalid day value
3. **Given** a user submits a fixed expense with day_of_month = 0, **When** validation runs, **Then** an error is returned for invalid day value
4. **Given** an existing fixed expense, **When** the user updates its payment method or category, **Then** changes are persisted and retrievable

---

### User Story 3 - Track Variable Expenses (Priority: P1)

A user records non-recurring expenses with details about destination, category, expense type, payment method, payment date, and amount paid.

**Why this priority**: Variable expense tracking completes the three core transaction types needed for a personal finance tool.

**Independent Test**: Can be fully tested by creating a variable expense record with all fields and verifying retrieval.

**Acceptance Scenarios**:

1. **Given** a user wants to record a variable expense, **When** they submit all required fields, **Then** the record is created with a unique identifier
2. **Given** a user submits a variable expense with a future payment date, **When** validation runs, **Then** the record is accepted (future-dated expenses are valid)
3. **Given** a user submits a variable expense with a negative amount, **When** validation runs, **Then** an error is returned for invalid amount
4. **Given** an existing variable expense record, **When** the user deletes it, **Then** the record is removed

---

### User Story 4 - Unified Financial View via GraphQL BFF (Priority: P2)

A user accesses a unified dashboard via GraphQL that aggregates their income, fixed expenses, and variable expenses. The BFF layer fetches data from the transactions service and optionally joins with identity data (user name, email).

**Why this priority**: The BFF adds value by providing a single endpoint for frontend consumers, but transaction CRUD is the foundation.

**Independent Test**: Can be tested by querying the GraphQL endpoint for a combined view of all transaction types for a given user.

**Acceptance Scenarios**:

1. **Given** a user has records across all three transaction types, **When** they query the unified view via GraphQL, **Then** all records are returned grouped by type
2. **Given** a query for a specific transaction by ID via GraphQL, **When** the record exists, **Then** the correct type-specific fields are returned
3. **Given** an unauthenticated request to the GraphQL endpoint, **When** no valid token is provided, **Then** an authentication error is returned

---

### Edge Cases

- What happens when the user has zero records of a given type? The list endpoint should return an empty list, not an error.
- How does the system handle concurrent updates to the same record? The last valid update is accepted; conflicting changes are rejected with a notification to the user.
- What happens when monetary amounts exceed expected precision? Amounts stored with up to 2 decimal places, extra precision rejected.
- How are deleted records handled? Records are hidden from normal views after deletion, preserving an audit trail for recovery if needed.
- What happens when the identity service is unavailable? The BFF should still return transaction data, omitting user details gracefully.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Users MUST be able to create, read, update, and delete income records with fields: description, source, income_type, received_date, received_amount, and status.
- **FR-002**: Users MUST be able to create, read, update, and delete fixed expense records with fields: description, category, day_of_month, payment_method, and status.
- **FR-003**: Users MUST be able to create, read, update, and delete variable expense records with fields: description, destination, category, expense_type, payment_method, payment_date, paid_amount, and status.
- **FR-004**: The system MUST validate that all required fields are present before persisting any record.
- **FR-005**: The system MUST validate that day_of_month is an integer between 1 and 31 inclusive.
- **FR-006**: The system MUST validate that monetary amounts (received_amount, paid_amount) are positive numbers with at most 2 decimal places.
- **FR-007**: The system MUST validate that dates are valid calendar dates.
- **FR-008**: The system MUST support listing records of each type with optional filters (date range, status, category).
- **FR-009**: The system MUST associate all records with the authenticated user.
- **FR-010**: The system MUST preserve deleted records for audit purposes (records are hidden from normal queries after deletion).
- **FR-011**: The GraphQL BFF MUST expose queries to read all three transaction types and return them in a unified schema.
- **FR-012**: The GraphQL BFF MUST require authentication for all queries.
- **FR-013**: The GraphQL BFF SHOULD integrate with the identity service to enrich transaction data with user profile information (name, email) when available.
- **FR-014**: The GraphQL BFF MUST gracefully degrade when the identity service is unavailable, returning transaction data without user details.
- **FR-015**: The system MUST support filtering transactions by date range for all three types.

### Key Entities *(include if feature involves data)*

- **Income**: Represents a received income entry. Attributes: id, user_id, description, source, income_type, received_date, received_amount, status, created_at, updated_at, deleted_at.
- **FixedExpense**: Represents a recurring fixed expense. Attributes: id, user_id, description, category, day_of_month, payment_method, status, created_at, updated_at, deleted_at.
- **VariableExpense**: Represents a non-recurring expense. Attributes: id, user_id, description, destination, category, expense_type, payment_method, payment_date, paid_amount, status, created_at, updated_at, deleted_at.
- **User** (from identity service): Represents the authenticated user. Attributes: id, name, email. Used for enrichment in the BFF layer.
- **TransactionUnion**: A combined view representing any of the three transaction types, enabling unified queries across all types.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can create any transaction type (income, fixed, variable) in under 3 seconds round-trip (including network, 95th percentile).
- **SC-002**: Users can retrieve their full transaction history (all types combined) via GraphQL in under 5 seconds for up to 1000 records.
- **SC-003**: All required field validations return clear error messages within 500ms of submission.
- **SC-004**: The BFF returns transaction data within 2 seconds even when the identity service is unavailable (graceful degradation).
- **SC-005**: 100% of transaction records are correctly associated with the authenticated user with no data leakage between users.

## Assumptions

- The identity service exists and provides user profile data via an internal API.
- Authentication is handled by the API layer before requests reach business logic.
- All monetary amounts are in a single currency (BRL), managed at the application level rather than the service level.
- Status values follow a standard lifecycle: pending → completed → cancelled (with reasonable transitions).
- Income types include: salary, freelance, investment, business, refund, other.
- Expense types include: essential, discretionary, occasional, emergency, other.
- Payment methods include: credit_card, debit_card, cash, bank_transfer, pix, other.
- Categories are free-text strings defined by the user.
- The frontend consuming the GraphQL BFF is a web application (mobile is future scope).
- Records are scoped to the authenticated user — no admin-level cross-user access in v1.
