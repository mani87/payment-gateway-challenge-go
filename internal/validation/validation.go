package validation

import (
	"time"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
)

var supportedCurrencies = map[string]bool{
	"GBP": true,
	"INR": true,
	"USD": true,
}

func ValidatePayment(r models.PostPaymentRequest, now time.Time) []models.FieldError {

	var errs []models.FieldError

	if len(r.CardNumber) < 14 || len(r.CardNumber) > 19 || !isDigits(r.CardNumber) {
		errs = append(errs, models.FieldError{Field: "card_number", Message: "must be a 14-19 numeric value"})
	}

	if r.ExpiryMonth < 1 || r.ExpiryMonth > 12 {
		errs = append(errs, models.FieldError{Field: "expiry_month", Message: "invalid expiry month"})
	}

	if r.ExpiryMonth >= 1 && r.ExpiryMonth <= 12 {
		if now.Year() > r.ExpiryYear || (now.Year() == r.ExpiryYear && now.Month() > time.Month(r.ExpiryMonth)) {
			errs = append(errs, models.FieldError{Field: "expiry_year", Message: "expired card"})
		}
	}

	if !supportedCurrencies[r.Currency] {
		errs = append(errs, models.FieldError{Field: "currency", Message: "unsupported currency"})
	}

	if r.Amount <= 0 {
		errs = append(errs, models.FieldError{Field: "amount", Message: "amount must be a positive integer"})
	}

	if len(r.Cvv) < 3 || len(r.Cvv) > 4 || !isDigits(r.Cvv) {
		errs = append(errs, models.FieldError{Field: "cvv", Message: "invalid cvv"})
	}

	return errs
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
