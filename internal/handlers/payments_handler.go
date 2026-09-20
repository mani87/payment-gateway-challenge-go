package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/bank"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/repository"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/validation"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Its easier to test it with interface without touching http
// Can be done better but...time
type BankClient interface {
	ProcessPayment(ctx context.Context, req models.PostPaymentRequest) (*bank.BankPaymentResponse, error)
}

type PaymentsHandler struct {
	storage    *repository.PaymentsRepository
	bankClient BankClient
}

func NewPaymentsHandler(storage *repository.PaymentsRepository, client BankClient) *PaymentsHandler {
	return &PaymentsHandler{
		storage:    storage,
		bankClient: client,
	}
}

// GetHandler returns an http.HandlerFunc that handles HTTP GET requests.
// It retrieves a payment record by its ID from the storage.
// The ID is expected to be part of the URL.
func (h *PaymentsHandler) GetHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		payment := h.storage.GetPayment(id)

		if payment != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(payment); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func (ph *PaymentsHandler) PostHandler() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		var request models.PostPaymentRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeRejected(w, []models.FieldError{{Field: "body", Message: "invalid json"}})
			return
		}

		// validation errors does not reach bank
		if errs := validation.ValidatePayment(request, time.Now()); len(errs) > 0 {
			writeRejected(w, errs)
			return
		}

		bankResponse, err := ph.bankClient.ProcessPayment(
			r.Context(),
			request,
		)

		if err != nil {
			switch {
			case errors.Is(err, bank.ErrBadBankRequest):
				http.Error(w, "malformed request", http.StatusBadGateway)
				return
			case errors.Is(err, bank.ErrBankUnavailable):
				http.Error(w, "bank unavailable", http.StatusBadGateway)
				return
			default:
				http.Error(w, "unexpected error", http.StatusInternalServerError)
				// nothing persists
				return
			}
		}

		status := "Declined"
		if bankResponse.Authorized {
			status = "Authorized"
		}

		payment := models.PostPaymentResponse{
			Id:                 uuid.NewString(),
			PaymentStatus:      status,
			CardNumberLastFour: request.CardNumber[len(request.CardNumber)-4:],
			ExpiryMonth:        request.ExpiryMonth,
			ExpiryYear:         request.ExpiryYear,
			Amount:             request.Amount,
			Currency:           request.Currency,
		}

		ph.storage.AddPayment(payment)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(payment)
	}
}

func writeRejected(w http.ResponseWriter, errs []models.FieldError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(models.RejectedResponse{
		PaymentStatus: "Rejected",
		Errors:        errs,
	})
}
