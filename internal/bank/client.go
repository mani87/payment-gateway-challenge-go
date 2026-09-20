package bank

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
)

var (
	ErrBankUnavailable = errors.New("acquiring bank unavailable")
	ErrBadBankRequest  = errors.New("rejected because bad request")
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type BankPaymentResponse struct {
	Authorized        bool   `json:"authorized"`
	AuthorizationCode string `json:"authorization_code"`
}

// Expiry date is expected here instead of ExpiryMonth and ExpiryYear sep
type BankPaymentRequest struct {
	CardNumber string `json:"card_number"`
	ExpiryDate string `json:"expiry_date"`
	Currency   string `json:"currency"`
	Amount     int    `json:"amount"`
	Cvv        string `json:"cvv"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

func (c *Client) ProcessPayment(ctx context.Context, r models.PostPaymentRequest) (*BankPaymentResponse, error) {

	payload := &BankPaymentRequest{
		CardNumber: r.CardNumber,
		ExpiryDate: fmt.Sprintf("%02d/%d", r.ExpiryMonth, r.ExpiryYear),
		Currency:   r.Currency,
		Amount:     r.Amount,
		Cvv:        r.Cvv,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal bank request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/payments",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("construct bank request %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Network error, Timeout error or Bank was not available
		return nil, fmt.Errorf("%w: %v", ErrBankUnavailable, err)
	}

	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		var res BankPaymentResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, fmt.Errorf("decode bank response: %w", err)
		}
		return &res, nil
	case resp.StatusCode == http.StatusBadRequest:
		return nil, ErrBadBankRequest
	default:
		// Anything else we are treating as Bank unavailable
		return nil, fmt.Errorf("%w , status: %v", ErrBankUnavailable, resp.StatusCode)
	}
}
