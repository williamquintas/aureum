package domain

type TransactionStatus string

const (
	StatusPending   TransactionStatus = "pending"
	StatusCompleted TransactionStatus = "completed"
	StatusCancelled TransactionStatus = "cancelled"
)

func (s TransactionStatus) Valid() bool {
	switch s {
	case StatusPending, StatusCompleted, StatusCancelled:
		return true
	}
	return false
}

type PaymentMethod string

const (
	PaymentMethodCreditCard   PaymentMethod = "credit_card"
	PaymentMethodDebitCard    PaymentMethod = "debit_card"
	PaymentMethodCash         PaymentMethod = "cash"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodPix          PaymentMethod = "pix"
	PaymentMethodOther        PaymentMethod = "other"
)

func (p PaymentMethod) Valid() bool {
	switch p {
	case PaymentMethodCreditCard, PaymentMethodDebitCard, PaymentMethodCash,
		PaymentMethodBankTransfer, PaymentMethodPix, PaymentMethodOther:
		return true
	}
	return false
}

type ExpenseType string

const (
	ExpenseTypeEssential     ExpenseType = "essential"
	ExpenseTypeDiscretionary ExpenseType = "discretionary"
	ExpenseTypeOccasional    ExpenseType = "occasional"
	ExpenseTypeEmergency     ExpenseType = "emergency"
	ExpenseTypeOther         ExpenseType = "other"
)

func (e ExpenseType) Valid() bool {
	switch e {
	case ExpenseTypeEssential, ExpenseTypeDiscretionary, ExpenseTypeOccasional,
		ExpenseTypeEmergency, ExpenseTypeOther:
		return true
	}
	return false
}

type UserInfo struct {
	ID    string
	Email string
	Name  string
}
