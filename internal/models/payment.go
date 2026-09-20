package models

/*
Changed CardNumber and Cvv to string
because its an identifier we don't perform arithmetic on this.
Secondly, to preserve leading zeroes '0012' doesn't become 12
*/
type PostPaymentRequest struct {
	CardNumber  string `json:"card_number"`
	ExpiryMonth int    `json:"expiry_month"`
	ExpiryYear  int    `json:"expiry_year"`
	Currency    string `json:"currency"`
	Amount      int    `json:"amount"`
	Cvv         string `json:"cvv"`
}

type PostPaymentResponse struct {
	Id                 string `json:"id"`
	PaymentStatus      string `json:"payment_status"`
	CardNumberLastFour string `json:"card_number_last_four"`
	ExpiryMonth        int    `json:"expiry_month"`
	ExpiryYear         int    `json:"expiry_year"`
	Currency           string `json:"currency"`
	Amount             int    `json:"amount"`
}

type GetPaymentResponse struct {
	Id                 string `json:"id"`
	PaymentStatus      string `json:"payment_status"`
	CardNumberLastFour string `json:"card_number_last_four"`
	ExpiryMonth        int    `json:"expiry_month"`
	ExpiryYear         int    `json:"expiry_year"`
	Currency           string `json:"currency"`
	Amount             int    `json:"amount"`
}

// Tip: Since validation errors are data and not behaviours it should be close to req and res
// To capture which field caused the error/rejection
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type RejectedResponse struct {
	PaymentStatus string       `json:"payment_status"`
	Errors        []FieldError `json:"errors"`
}
