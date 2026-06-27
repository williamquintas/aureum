package domain

import (
	"math"
	"time"
)

// AmortizationEntry represents a single month in an amortization schedule.
type AmortizationEntry struct {
	Month        int32
	Principal    int64
	Interest     int64
	Balance      int64
	TotalPayment int64
}

// AmortizationSchedule represents the full amortization schedule for a debt.
type AmortizationSchedule struct {
	DebtID          string
	TotalAmount     int64
	MonthlyPayment  int64
	InterestRate    int64 // annual % * 100 (e.g. 1250 = 12.50%)
	RemainingMonths int32
	TotalInterest   int64
	TotalPaid       int64
	Entries         []AmortizationEntry
}

// CalculateAmortization computes a monthly amortization schedule.
// totalAmount and monthlyPayment are in cents.
// interestRate is annual percentage * 100 (e.g. 1250 = 12.50%).
// months is the number of months to calculate.
func CalculateAmortization(totalAmount, interestRate, monthlyPayment int64, months int) AmortizationSchedule {
	// Monthly interest rate as a fraction: (rate / 100) / 12
	// Since rate is stored as % * 100, divide by 10000 to get the decimal rate per year,
	// then by 12 for monthly.
	monthlyRate := float64(interestRate) / 10000.0 / 12.0

	entries := make([]AmortizationEntry, 0, months)
	balance := float64(totalAmount)
	var totalInterest float64

	for i := 1; i <= months && balance > 0.01; i++ {
		interest := balance * monthlyRate
		principal := float64(monthlyPayment) - interest

		// Round to nearest cent for consistent integer arithmetic
		principalCents := int64(math.Round(principal))
		interestCents := int64(math.Round(interest))

		if principalCents <= 0 {
			// Payment too small to cover interest; pay what we can
			principalCents = 0
			interestCents = int64(math.Round(balance))
			balance = 0
		} else if float64(principalCents) >= balance {
			// Final payment: use integer-rounded principal to match total amount
			principalCents = int64(math.Round(balance))
			interestCents = int64(math.Round(balance * monthlyRate))
			balance = 0
		} else {
			// Use rounded principal to keep balance consistent with entries
			balance -= float64(principalCents)
		}

		totalPayment := principalCents + interestCents
		totalInterest += float64(interestCents)

		entries = append(entries, AmortizationEntry{
			Month:        int32(i),
			Principal:    principalCents,
			Interest:     interestCents,
			Balance:      int64(math.Round(balance)),
			TotalPayment: totalPayment,
		})
	}

	totalInterestCents := int64(math.Round(totalInterest))
	totalPaid := totalAmount + totalInterestCents

	return AmortizationSchedule{
		TotalAmount:     totalAmount,
		MonthlyPayment:  monthlyPayment,
		InterestRate:    interestRate,
		RemainingMonths: int32(len(entries)),
		TotalInterest:   totalInterestCents,
		TotalPaid:       totalPaid,
		Entries:         entries,
	}
}

// ComputeMonthlyPayment computes the fixed monthly payment using the amortization formula.
// totalAmount in cents, annualRate is % * 100 (e.g. 750 = 7.50%), months is loan term.
func ComputeMonthlyPayment(totalAmount int64, annualRate int64, months int) int64 {
	if months <= 0 {
		return 0
	}
	if annualRate == 0 {
		return totalAmount / int64(months)
	}
	monthlyRate := float64(annualRate) / 10000.0 / 12.0
	pow := math.Pow(1+monthlyRate, float64(months))
	payment := float64(totalAmount) * (monthlyRate * pow) / (pow - 1)
	return int64(math.Round(payment))
}

// MonthsBetween computes the number of calendar months between two dates.
// Returns at least 1 month.
func MonthsBetween(start, end string) (int, error) {
	startTime, err := time.Parse("2006-01-02", start)
	if err != nil {
		return 0, ErrInvalidDate
	}
	endTime, err := time.Parse("2006-01-02", end)
	if err != nil {
		return 0, ErrInvalidDate
	}
	months := (endTime.Year()-startTime.Year())*12 + int(endTime.Month()-startTime.Month())
	if months < 1 {
		months = 1
	}
	return months, nil
}
