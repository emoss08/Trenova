package rampconnector

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

// Ramp issues a bearer token from an app a workspace admin creates. Unlike the
// fleet networks, this needs no agreement with Trenova: the customer's own
// credentials are all that is required.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type merchantLocation struct {
	City  string `json:"city"`
	State string `json:"state"`
}

type cardHolder struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	CardID    string `json:"card_id"`
	LastFour  string `json:"card_last_four"`
}

type transaction struct {
	ID                  string            `json:"id"`
	Amount              float64           `json:"amount"`
	CurrencyCode        string            `json:"currency_code"`
	MerchantName        string            `json:"merchant_name"`
	MerchantDescriptor  string            `json:"merchant_descriptor"`
	UserTransactionTime string            `json:"user_transaction_time"`
	CardID              string            `json:"card_id"`
	CardLastFour        string            `json:"card_last_four"`
	MerchantLocation    *merchantLocation `json:"merchant_location"`
	CardHolder          *cardHolder       `json:"card_holder"`
}

type transactionPage struct {
	Data []transaction `json:"data"`
	Page struct {
		Next string `json:"next"`
	} `json:"page"`
}

func (t transaction) lastFour() string {
	if four := strings.TrimSpace(t.CardLastFour); four != "" {
		return four
	}
	if t.CardHolder != nil {
		return strings.TrimSpace(t.CardHolder.LastFour)
	}

	return ""
}

func (t transaction) merchantCity() string {
	if t.MerchantLocation == nil {
		return ""
	}

	return strings.TrimSpace(t.MerchantLocation.City)
}

func (t transaction) merchantState() string {
	if t.MerchantLocation == nil {
		return ""
	}

	return strings.TrimSpace(t.MerchantLocation.State)
}

func (t transaction) holderName() string {
	if t.CardHolder == nil {
		return ""
	}

	return strings.TrimSpace(t.CardHolder.FirstName + " " + t.CardHolder.LastName)
}

func (t transaction) merchant() string {
	if name := strings.TrimSpace(t.MerchantName); name != "" {
		return name
	}

	return strings.TrimSpace(t.MerchantDescriptor)
}

// cells lays a transaction out in the order sheetHeaders declares. A field Ramp
// did not send becomes an empty cell, which the staging pipeline treats exactly
// as it treats a blank column in an uploaded file.
func (t transaction) cells() []string {
	return []string{
		t.UserTransactionTime,
		t.merchant(),
		t.merchantCity(),
		t.merchantState(),
		strconv.FormatFloat(t.Amount, 'f', -1, 64),
		t.CurrencyCode,
		t.lastFour(),
		t.ID,
		t.holderName(),
		"",
	}
}

func (c *Connector) token(ctx context.Context, settings *settings) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("scope", "transactions:read cards:read")

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		settings.baseURL+"/token",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("build Ramp token request: %w", err)
	}
	req.SetBasicAuth(settings.clientID, settings.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	body, err := c.do(req, "authenticate with Ramp")
	if err != nil {
		return "", err
	}

	var parsed tokenResponse
	if err = sonic.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("read Ramp token response: %w", err)
	}
	if strings.TrimSpace(parsed.AccessToken) == "" {
		return "", fmt.Errorf("Ramp returned no access token")
	}

	return parsed.AccessToken, nil
}

// transactions walks the window a page at a time. The page cap is a guard against
// a cursor that never terminates rather than a limit anybody should reach: a
// window is at most a few days of one workspace's spending.
func (c *Connector) transactions(
	ctx context.Context,
	settings *settings,
	token string,
	since, until int64,
) ([]transaction, error) {
	next := settings.baseURL + "/transactions?" + url.Values{
		"from_date": {formatInstant(since)},
		"to_date":   {formatInstant(until)},
		"page_size": {strconv.Itoa(pageSize)},
	}.Encode()

	collected := make([]transaction, 0, pageSize)

	for page := 0; page < maxPages && next != ""; page++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, next, nil)
		if err != nil {
			return nil, fmt.Errorf("build Ramp transactions request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		body, err := c.do(req, "read Ramp transactions")
		if err != nil {
			return nil, err
		}

		var parsed transactionPage
		if err = sonic.Unmarshal(body, &parsed); err != nil {
			return nil, fmt.Errorf("read Ramp transactions response: %w", err)
		}

		collected = append(collected, parsed.Data...)
		next = strings.TrimSpace(parsed.Page.Next)
	}

	return collected, nil
}

func (c *Connector) do(req *http.Request, action string) ([]byte, error) {
	response, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", action, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", action, err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("%s: Ramp returned HTTP %d", action, response.StatusCode)
	}

	return body, nil
}

func formatInstant(seconds int64) string {
	return time.Unix(seconds, 0).UTC().Format(time.RFC3339)
}
