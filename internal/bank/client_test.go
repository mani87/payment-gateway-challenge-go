package bank

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
	"github.com/stretchr/testify/assert"
)

func validPaymentRequest() models.PostPaymentRequest {
	return models.PostPaymentRequest{
		CardNumber:  "2222405343248877",
		ExpiryMonth: 6,
		ExpiryYear:  2030,
		Currency:    "GBP",
		Amount:      100,
		Cvv:         "123",
	}
}

func TestClient_ProcessPayment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/payments", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req BankPaymentRequest
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "2222405343248877", req.CardNumber)
		assert.Equal(t, "06/2030", req.ExpiryDate)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(BankPaymentResponse{
			Authorized:        true,
			AuthorizationCode: "auth-code-123",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.ProcessPayment(context.Background(), validPaymentRequest())

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Authorized)
	assert.Equal(t, "auth-code-123", resp.AuthorizationCode)
}

func TestClient_ProcessPayment_Declined(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(BankPaymentResponse{Authorized: false})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.ProcessPayment(context.Background(), validPaymentRequest())

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.False(t, resp.Authorized)
}

func TestClient_ProcessPayment_ServiceUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.ProcessPayment(context.Background(), validPaymentRequest())

	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrBankUnavailable), "expected ErrBankUnavailable, got %v", err)
}

func TestClient_ProcessPayment_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.ProcessPayment(context.Background(), validPaymentRequest())

	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrBadBankRequest), "expected ErrBadBankRequest, got %v", err)
}

func TestClient_ProcessPayment_UnreachableHost(t *testing.T) {
	client := NewClient("http://localhost:1")

	resp, err := client.ProcessPayment(context.Background(), validPaymentRequest())

	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrBankUnavailable), "expected ErrBankUnavailable, got %v", err)
}

func TestClient_ProcessPayment_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 50 * time.Millisecond},
	}

	resp, err := client.ProcessPayment(context.Background(), validPaymentRequest())

	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrBankUnavailable), "expected ErrBankUnavailable on timeout, got %v", err)
}

func TestClient_ProcessPayment_MalformedResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.ProcessPayment(context.Background(), validPaymentRequest())

	assert.Nil(t, resp)
	assert.Error(t, err)
}
