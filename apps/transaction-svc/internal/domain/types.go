package domain

// TransactionStatus represents the lifecycle state of a transaction record.
type TransactionStatus string

const (
	// StatusPending indicates a transaction that has not yet been completed.
	StatusPending TransactionStatus = "pending"
	// StatusCompleted indicates a transaction that has been completed.
	StatusCompleted TransactionStatus = "completed"
	// StatusCancelled indicates a transaction that has been cancelled.
	StatusCancelled TransactionStatus = "cancelled"
)

// Valid returns true if the status is a recognised value.
func (s TransactionStatus) Valid() bool {
	switch s {
	case StatusPending, StatusCompleted, StatusCancelled:
		return true
	}
	return false
}

// PaymentMethod represents the method used to make a payment.
type PaymentMethod string

const (
	// PaymentMethodCreditCard represents credit card payments.
	PaymentMethodCreditCard PaymentMethod = "credit_card"
	// PaymentMethodDebitCard represents debit card payments.
	PaymentMethodDebitCard PaymentMethod = "debit_card"
	// PaymentMethodCash represents cash payments.
	PaymentMethodCash PaymentMethod = "cash"
	// PaymentMethodBankTransfer represents bank transfer payments.
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	// PaymentMethodPix represents Pix (Brazilian instant payment) transfers.
	PaymentMethodPix PaymentMethod = "pix"
	// PaymentMethodOther represents any other payment method.
	PaymentMethodOther PaymentMethod = "other"
)

// Valid returns true if the payment method is a recognised value.
func (p PaymentMethod) Valid() bool {
	switch p {
	case PaymentMethodCreditCard, PaymentMethodDebitCard, PaymentMethodCash,
		PaymentMethodBankTransfer, PaymentMethodPix, PaymentMethodOther:
		return true
	}
	return false
}

// ExpenseType categorises the nature of a variable expense.
type ExpenseType string

const (
	// ExpenseTypeEssential represents necessary expenses (e.g. bills, groceries).
	ExpenseTypeEssential ExpenseType = "essential"
	// ExpenseTypeDiscretionary represents non-essential spending.
	ExpenseTypeDiscretionary ExpenseType = "discretionary"
	// ExpenseTypeOccasional represents infrequent or one-off expenses.
	ExpenseTypeOccasional ExpenseType = "occasional"
	// ExpenseTypeEmergency represents unexpected urgent expenses.
	ExpenseTypeEmergency ExpenseType = "emergency"
	// ExpenseTypeOther represents any other expense category.
	ExpenseTypeOther ExpenseType = "other"
)

// Valid returns true if the expense type is a recognised value.
func (e ExpenseType) Valid() bool {
	switch e {
	case ExpenseTypeEssential, ExpenseTypeDiscretionary, ExpenseTypeOccasional,
		ExpenseTypeEmergency, ExpenseTypeOther:
		return true
	}
	return false
}

// UserInfo holds basic identity information for the authenticated user.
type UserInfo struct {
	ID    string
	Email string
	Name  string
}
