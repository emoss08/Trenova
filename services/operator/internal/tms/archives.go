package tms

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
)

const (
	CommandArchive = "archive"
	CommandRestore = "restore"
	CommandImport  = "import"
)

var (
	ErrCommandNotFound = errors.New("the TMS holds no such command for the tenant")
	ErrTenantBusy      = errors.New(
		"the tenant runs another archive, restore or import; the command waits for it",
	)
)

type ArchiveKey struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	Current bool   `json:"current"`
}

type ArchiveTarget struct {
	InstanceID      string       `json:"instanceId"`
	CommandID       string       `json:"commandId"`
	Kind            string       `json:"kind"`
	Access          string       `json:"access"`
	Endpoint        string       `json:"endpoint"`
	Region          string       `json:"region"`
	Bucket          string       `json:"bucket"`
	Prefix          string       `json:"prefix"`
	AccessKeyID     string       `json:"accessKeyId"`
	SecretAccessKey string       `json:"secretAccessKey"`
	SessionToken    string       `json:"sessionToken,omitempty"`
	ExpiresAt       *time.Time   `json:"expiresAt,omitempty"`
	ArchiveKeys     []ArchiveKey `json:"archiveKeys,omitempty"`
}

func (t ArchiveTarget) String() string {
	return "archive target " + t.Kind + " " + t.CommandID + " at s3://" + t.Bucket + "/" + t.Prefix
}

func (t ArchiveTarget) GoString() string { //nolint:gocritic // hugeParam: a value receiver, so %#v on a value prints this and never the secret fields
	return t.String()
}

type ArchiveStart struct {
	CommandID string        `json:"commandId"`
	Purpose   string        `json:"purpose"`
	Target    ArchiveTarget `json:"target"`
}

type RestoreStart struct {
	CommandID    string        `json:"commandId"`
	ExpectSHA256 string        `json:"expectSha256"`
	Target       ArchiveTarget `json:"target"`
}

type NightlyGrant struct {
	Endpoint        string       `json:"endpoint"`
	Region          string       `json:"region"`
	Bucket          string       `json:"bucket"`
	AccessKeyID     string       `json:"accessKeyId"`
	SecretAccessKey string       `json:"secretAccessKey"`
	SessionToken    string       `json:"sessionToken,omitempty"`
	ExpiresAt       *time.Time   `json:"expiresAt,omitempty"`
	ArchiveKeys     []ArchiveKey `json:"archiveKeys"`
}

func (g NightlyGrant) String() string {
	return "nightly archive grant for s3://" + g.Bucket
}

func (g NightlyGrant) GoString() string { //nolint:gocritic // hugeParam: a value receiver, so %#v on a value prints this and never the secret fields
	return g.String()
}

type CommandProgress struct {
	Phase          string `json:"phase"`
	TablesDone     int    `json:"tablesDone"`
	TablesTotal    int    `json:"tablesTotal"`
	DocumentsDone  int    `json:"documentsDone"`
	DocumentsTotal int    `json:"documentsTotal"`
	Bytes          int64  `json:"bytes"`
}

type TenantCommand struct {
	CommandID  string          `json:"commandId"`
	InstanceID string          `json:"instanceId"`
	Kind       string          `json:"kind"`
	Purpose    string          `json:"purpose,omitempty"`
	State      string          `json:"state"`
	Location   string          `json:"location"`
	Progress   CommandProgress `json:"progress"`
	SHA256     string          `json:"sha256,omitempty"`
	SizeBytes  int64           `json:"sizeBytes,omitempty"`
	Reason     string          `json:"reason,omitempty"`
}

func (c *TenantCommand) Settled() bool {
	return c.State == "done" || c.State == "failed"
}

func (c *Client) StartArchive(
	ctx context.Context,
	instanceID string,
	start *ArchiveStart,
) (TenantCommand, error) {
	return c.startCommand(ctx, instanceID, "archives", start)
}

func (c *Client) StartRestore(
	ctx context.Context,
	instanceID string,
	start *RestoreStart,
) (TenantCommand, error) {
	return c.startCommand(ctx, instanceID, "restores", start)
}

func (c *Client) startCommand(
	ctx context.Context,
	instanceID, collection string,
	start any,
) (TenantCommand, error) {
	body, err := sonic.Marshal(start)
	if err != nil {
		return TenantCommand{}, errors.New("start a tenant command on the TMS: cannot encode it")
	}
	u := c.base.JoinPath("managed", "v1", "tenants", instanceID, collection)
	out, err := c.do(ctx, http.MethodPost, u, body, http.StatusAccepted)
	if err != nil {
		return TenantCommand{}, fmt.Errorf("start a tenant command on the TMS: %w", commandErr(err))
	}
	var cmd TenantCommand
	if err = sonic.Unmarshal(out, &cmd); err != nil {
		return TenantCommand{}, fmt.Errorf("start a tenant command on the TMS: %w", err)
	}
	return cmd, nil
}

func (c *Client) TenantCommand(
	ctx context.Context,
	instanceID, kind, commandID string,
) (TenantCommand, error) {
	collection := map[string]string{
		CommandArchive: "archives",
		CommandRestore: "restores",
		CommandImport:  "imports",
	}[kind]
	if collection == "" {
		return TenantCommand{}, fmt.Errorf("read a tenant command: unknown kind %q", kind)
	}
	u := c.base.JoinPath("managed", "v1", "tenants", instanceID, collection, commandID)
	out, err := c.do(ctx, http.MethodGet, u, nil, http.StatusOK)
	if err != nil {
		return TenantCommand{}, fmt.Errorf("read a tenant command on the TMS: %w", commandErr(err))
	}
	var cmd TenantCommand
	if err = sonic.Unmarshal(out, &cmd); err != nil {
		return TenantCommand{}, fmt.Errorf("read a tenant command on the TMS: %w", err)
	}
	return cmd, nil
}

func (c *Client) PutNightlyGrant(ctx context.Context, grant *NightlyGrant) error {
	body, err := sonic.Marshal(grant)
	if err != nil {
		return errors.New("send the nightly archive grant to the TMS: cannot encode it")
	}
	u := c.base.JoinPath("managed", "v1", "tenants", "archive-storage")
	if _, err = c.do(ctx, http.MethodPut, u, body, http.StatusOK); err != nil {
		return fmt.Errorf("send the nightly archive grant to the TMS: %w", sharedOnly(err))
	}
	return nil
}

func commandErr(err error) error {
	var status *StatusError
	if !errors.As(err, &status) {
		return err
	}
	switch status.Status {
	case http.StatusNotFound:
		return fmt.Errorf("%w: %w", ErrCommandNotFound, err)
	case http.StatusLocked:
		return fmt.Errorf("%w: %w", ErrTenantBusy, err)
	default:
		return err
	}
}

func Refused(err error) (string, bool) {
	var status *StatusError
	if !errors.As(err, &status) {
		return "", false
	}
	if status.Status != http.StatusBadRequest && status.Status != http.StatusConflict {
		return "", false
	}
	if status.Detail != "" {
		return status.Detail, true
	}
	return status.Title, true
}
