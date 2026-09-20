# Payment Gateway

A payment gateway API for the Checkout.com engineering challenge. A merchant submits a
card payment, the gateway validates it, forwards it to the acquiring bank simulator,
stores the outcome, and lets the merchant retrieve it later by id.

## Running it

```bash
# start the bank simulator
docker-compose up -d

# start the gateway
go run main.go
```

The gateway listens on `:8090`. Swagger UI is at
`http://localhost:8090/swagger/index.html`.

The bank simulator URL is configurable via the `BANK_SIMULATOR_URL` environment
variable (default `http://localhost:8080`).

## Running the tests

```bash
go vet ./...
go test ./...
go test ./... -race     # confirms the repository is safe under concurrent access
```

Coverage:

- **Validation** — table-driven, including the expiry-month boundary (a card expiring
  this month is valid, last month is not), card/CVV length limits, and currency rules.
- **Bank client** — tested against a local `httptest` server: success, declined, 503,
  400, timeout, unreachable host, malformed response body.
- **HTTP handlers** — Authorized, Declined, bank-unavailable, invalid-request
  (asserts the bank is never called), not-found.
- **Repository** — concurrent reads and writes under `go test -race`.

## API

### `POST /api/payments`

Request fields:

| Field          | Rules                                                          |
|----------------|-----------------------------------------------------------------|
| `card_number`  | required, 14–19 numeric characters                              |
| `expiry_month` | required, 1–12                                                   |
| `expiry_year`  | required, month + year combination must be in the future         |
| `currency`     | required, exactly 3 characters, one of `GBP`, `USD`, `INR`       |
| `amount`       | required, positive integer, minor units (e.g. `1050` = $10.50)   |
| `cvv`          | required, 3–4 numeric characters                                 |

Response by outcome:

| Outcome                                          | Status | Body                                             | Persisted? |
|---------------------------------------------------|--------|---------------------------------------------------|------------|
| Valid request, bank authorizes                     | `201`  | `payment_status: "Authorized"`                     | Yes        |
| Valid request, bank declines                       | `201`  | `payment_status: "Declined"`                       | Yes        |
| Invalid request (fails validation)                 | `400`  | `payment_status: "Rejected"`, all field errors listed | No — bank is never called |
| Bank unreachable, times out, or returns `503`      | `502`  | error message                                       | No         |
| Bank returns `400` (gateway sent a malformed request) | `502` | error message                                       | No         |

A successful response never includes the full card number or CVV — only the last four
digits of the card are returned.

### `GET /api/payments/{id}`

Returns `200` with the stored payment (masked to last-four card digits), or `404` if
the id is unknown.

## Design decisions and assumptions

**Rejected vs Declined are deliberately different outcomes.** Rejected means the
gateway refused the request before ever contacting the bank — nothing is created,
nothing is retrievable. Declined means the bank was called with a valid request and
said no — a payment record is created, stored, and retrievable, because a merchant
doing reconciliation needs to see declined attempts, not just successes.

**Validation collects every failure, not just the first.** A merchant integrating
against this API shouldn't need one round trip per broken field — the `400` response
lists every rule violated in a single call.

**Three supported currencies, hardcoded.** The spec caps validation at no more than
three currency codes; a fixed map (`GBP`, `USD`, `INR`) is simpler than pulling in a
full ISO-4217 list for a constraint this narrow.

**In-memory storage, not a real database.** The spec explicitly allows this. The
repository is a `map[string]PostPaymentResponse` behind an `RWMutex`, since every HTTP
request runs on its own goroutine and concurrent reads/writes need to be safe — this
is verified under `go test -race`.

**Card number and CVV are never stored.** The full card number is used only to build
the outbound request to the bank and to derive the last four digits for the response;
it never appears in the stored payment record. The CVV is used once, for the bank
call, and discarded.

**Bank failures return `502`, not `503`.** A `503` would imply the gateway itself is
unhealthy. A `502` correctly signals that a downstream dependency — the bank — is the
problem. Internally, a `400` from the bank (a gateway defect, since the gateway built
the malformed request) and an unavailable bank (a genuine outage: connection refused,
timeout, or a `503`) are distinguished via separate sentinel errors, even though both
currently map to the same `502` status for the merchant.

## Architecture

```
handlers/    HTTP boundary only — decode, validate, call bank, persist, respond
validation/  pure functions, no HTTP dependency, easy to table-test
bank/        HTTP client for the acquiring bank simulator; translates wire format
             and maps failures to sentinel errors (ErrBankUnavailable, ErrBadBankRequest)
repository/  in-memory store, map + RWMutex
models/      shared request/response/domain types
```

The `BankClient` interface is defined in `handlers`, where it's consumed, rather than
next to its implementation in `bank`. This is what lets the handler tests run against
a fake bank with no network calls at all, while the `bank` package's own tests exercise
the real client against a local `httptest` server.

# Instructions for candidates

This is the Go version of the Payment Gateway challenge. If you haven't already read
the [README.md](https://github.com/cko-recruitment/) in the root of this organisation,
please do so now.