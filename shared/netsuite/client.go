package netsuite

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

const (
	MaxExternalIDLength = 255
	maxInternalIDDigits = 20
	externalIDPrefix    = "eid:"
	expandParam         = "expandSubResources"
	fieldsParam         = "fields"
	replaceParam        = "replace"
	listSeparator       = ","
)

var (
	recordTypePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,99}$`)
	fieldNamePattern  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,99}$`)
	externalIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

type AccountConfig struct {
	AccountID   string
	AccessToken string
}

type Client struct {
	core      *core
	accountID string
}

type RecordWrite struct {
	Type       string
	ExternalID string
	Body       any
}

type RecordPatch struct {
	Type    string
	ID      string
	Body    any
	Replace []string
}

type RecordRead struct {
	Type               string
	ID                 string
	ExpandSubResources bool
	Fields             []string
}

func NewAccountClient(cfg AccountConfig, opts ...Option) (*Client, error) {
	token := strings.TrimSpace(cfg.AccessToken)
	if token == "" {
		return nil, ErrAccessTokenRequired
	}
	account, err := NormalizeAccountID(cfg.AccountID)
	if err != nil {
		return nil, err
	}
	settings := resolveOptions(opts)
	baseURL, err := settings.apiBaseURL(account)
	if err != nil {
		return nil, err
	}
	c, err := newCore(&coreParams{
		settings: &settings,
		baseURL:  baseURL,
		token:    token,
		bucket:   AccountBucket(settings.limiterKeyPrefix, account),
	})
	if err != nil {
		return nil, err
	}
	return &Client{core: c, accountID: account}, nil
}

func (c *Client) AccountID() string {
	return c.accountID
}

func (c *Client) Upsert(ctx context.Context, req RecordWrite) (string, error) {
	recordType, err := validRecordType(req.Type)
	if err != nil {
		return "", err
	}
	externalID, err := validExternalID(req.ExternalID)
	if err != nil {
		return "", err
	}
	if req.Body == nil {
		return "", ErrBodyRequired
	}
	resp, err := c.core.do(ctx, &call{
		endpoint: "record-upsert",
		method:   http.MethodPut,
		path:     recordPath + "/" + recordType + "/" + externalIDPrefix + externalID,
		body:     req.Body,
		expected: []int{http.StatusNoContent, http.StatusOK, http.StatusCreated},
	})
	if err != nil {
		return "", err
	}
	return c.core.locationID(resp, recordType)
}

func (c *Client) Create(ctx context.Context, recordType string, body any) (string, error) {
	kind, err := validRecordType(recordType)
	if err != nil {
		return "", err
	}
	if body == nil {
		return "", ErrBodyRequired
	}
	resp, err := c.core.do(ctx, &call{
		endpoint: "record-create",
		method:   http.MethodPost,
		path:     recordPath + "/" + kind,
		body:     body,
		expected: []int{http.StatusNoContent, http.StatusCreated},
	})
	if err != nil {
		return "", err
	}
	return c.core.locationID(resp, kind)
}

func (c *Client) Patch(ctx context.Context, req RecordPatch) error {
	path, err := recordItemPath(req.Type, req.ID)
	if err != nil {
		return err
	}
	if req.Body == nil {
		return ErrBodyRequired
	}
	var query url.Values
	if len(req.Replace) > 0 {
		replace, joinErr := joinFieldNames(req.Replace)
		if joinErr != nil {
			return joinErr
		}
		query = url.Values{replaceParam: {replace}}
	}
	_, err = c.core.do(ctx, &call{
		endpoint: "record-patch",
		method:   http.MethodPatch,
		path:     path,
		query:    query,
		body:     req.Body,
		expected: []int{http.StatusNoContent, http.StatusOK},
	})
	return err
}

func (c *Client) Delete(ctx context.Context, recordType, id string) error {
	path, err := recordItemPath(recordType, id)
	if err != nil {
		return err
	}
	_, err = c.core.do(ctx, &call{
		endpoint: "record-delete",
		method:   http.MethodDelete,
		path:     path,
		expected: []int{http.StatusNoContent, http.StatusOK},
	})
	return err
}

func (c *Client) Get(ctx context.Context, req RecordRead, out any) error {
	path, err := recordItemPath(req.Type, req.ID)
	if err != nil {
		return err
	}
	if out == nil {
		return ErrOutRequired
	}
	query := url.Values{}
	if req.ExpandSubResources {
		query.Set(expandParam, "true")
	}
	if len(req.Fields) > 0 {
		fields, joinErr := joinFieldNames(req.Fields)
		if joinErr != nil {
			return joinErr
		}
		query.Set(fieldsParam, fields)
	}
	resp, err := c.core.do(ctx, &call{
		endpoint: "record-get",
		method:   http.MethodGet,
		path:     path,
		query:    query,
		out:      out,
	})
	if err != nil {
		return err
	}
	if len(resp.Body) == 0 {
		return ErrUnexpectedPayload
	}
	return nil
}

func recordItemPath(recordType, id string) (string, error) {
	kind, err := validRecordType(recordType)
	if err != nil {
		return "", err
	}
	segment, err := recordIDSegment(id)
	if err != nil {
		return "", err
	}
	return recordPath + "/" + kind + "/" + segment, nil
}

func validRecordType(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if !recordTypePattern.MatchString(value) {
		return "", ErrInvalidRecordType
	}
	return value, nil
}

func validExternalID(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if len(value) > MaxExternalIDLength || !externalIDPattern.MatchString(value) {
		return "", ErrInvalidExternalID
	}
	return value, nil
}

func recordIDSegment(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if external, ok := strings.CutPrefix(value, externalIDPrefix); ok {
		id, err := validExternalID(external)
		if err != nil {
			return "", err
		}
		return externalIDPrefix + id, nil
	}
	if !isInternalID(value) {
		return "", ErrInvalidRecordID
	}
	return value, nil
}

func isInternalID(value string) bool {
	if value == "" || len(value) > maxInternalIDDigits {
		return false
	}
	for idx := range len(value) {
		if value[idx] < '0' || value[idx] > '9' {
			return false
		}
	}
	return true
}

func joinFieldNames(names []string) (string, error) {
	cleaned := make([]string, 0, len(names))
	for _, name := range names {
		value := strings.TrimSpace(name)
		if !fieldNamePattern.MatchString(value) {
			return "", ErrInvalidFieldName
		}
		cleaned = append(cleaned, value)
	}
	return strings.Join(cleaned, listSeparator), nil
}
