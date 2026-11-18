package tms

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
)

const maxLogRequests = 10

var (
	ErrNoGrant = errors.New(
		"the staff address holds no live support access grant on this instance",
	)
	ErrLogRequestSettled = errors.New("the TMS has already moved the log request past that status")
)

type LogRequester struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type LogRequest struct {
	ID          string           `json:"id"`
	Since       time.Time        `json:"since"`
	Until       time.Time        `json:"until"`
	RequestedBy LogRequester     `json:"requestedBy"`
	RequestedAt time.Time        `json:"requestedAt"`
	Status      LogRequestStatus `json:"status"`
	BundleID    string           `json:"bundleId,omitempty"`
}

type LogRequestStatus string

const (
	LogRequestRequested  LogRequestStatus = "requested"
	LogRequestCollecting LogRequestStatus = "collecting"
	LogRequestSent       LogRequestStatus = "sent"
	LogRequestFailed     LogRequestStatus = "failed"
)

type LogRequestUpdate struct {
	Status   LogRequestStatus `json:"status"`
	BundleID string           `json:"bundleId,omitempty"`
	Message  string           `json:"message,omitempty"`
}

type LogPullKind string

const (
	LogPullLines  LogPullKind = "lines"
	LogPullBundle LogPullKind = "bundle"
)

func (c *Client) LogRequests(ctx context.Context) ([]LogRequest, error) {
	u := c.base.JoinPath("managed", "v1", "log-requests")
	out, err := c.do(ctx, http.MethodGet, u, nil, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("read the TMS's log requests: %w", err)
	}
	var list struct {
		Requests []LogRequest `json:"requests"`
	}
	if err = sonic.Unmarshal(out, &list); err != nil {
		return nil, fmt.Errorf("read the TMS's log requests: %w", err)
	}
	if len(list.Requests) > maxLogRequests {
		list.Requests = list.Requests[:maxLogRequests]
	}
	return list.Requests, nil
}

func (c *Client) UpdateLogRequest(ctx context.Context, id string, update LogRequestUpdate) error {
	body, err := sonic.Marshal(update)
	if err != nil {
		return fmt.Errorf("tell the TMS how a log request goes: %w", err)
	}
	u := c.base.JoinPath("managed", "v1", "log-requests", id)
	_, err = c.do(ctx, http.MethodPut, u, body, http.StatusNoContent)
	var status *StatusError
	if errors.As(err, &status) && status.Status == http.StatusConflict {
		return ErrLogRequestSettled
	}
	if err != nil {
		return fmt.Errorf("tell the TMS how a log request goes: %w", err)
	}
	return nil
}

func (c *Client) AuthorizeLogPull(ctx context.Context, email string, kind LogPullKind) error {
	body, err := sonic.Marshal(struct {
		Email string      `json:"email"`
		Kind  LogPullKind `json:"kind"`
	}{Email: email, Kind: kind})
	if err != nil {
		return fmt.Errorf("ask the TMS for a support access grant: %w", err)
	}
	u := c.base.JoinPath("managed", "v1", "support", "log-pulls")
	_, err = c.do(ctx, http.MethodPost, u, body, http.StatusOK, http.StatusNoContent)
	var status *StatusError
	if errors.As(err, &status) && status.Status == http.StatusForbidden {
		return ErrNoGrant
	}
	if err != nil {
		return fmt.Errorf("ask the TMS for a support access grant: %w", err)
	}
	return nil
}
