package sim

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	maxWebhookNameLength        = 255
	maxWebhookURLLength         = 2047
	maxWebhookCustomHeaders     = 5
	maxWebhookHeaderKeyLength   = 100
	maxWebhookHeaderValueLength = 100
	defaultWebhookVersion       = "2018-01-01"
	fieldWebhookURL             = "url"
	fieldWebhookVersion         = "version"
	fieldWebhookEventTypes      = "eventTypes"
	fieldWebhookCustomHeaders   = "customHeaders"
	fieldWebhookSecretKey       = "secretKey"
	fieldHeaderKey              = "key"
	labelWebhook                = "webhook"
	httpTokenSymbols            = "!#$%&'*+-.^_`|~"
)

var webhookVersions = []string{"2018-01-01", "2021-06-09", "2022-09-13", "2024-02-27"}

type webhookCustomHeader struct {
	Key   string
	Value string
}

func (s *Server) registerWebhookRoutes() {
	s.mux.HandleFunc("GET /webhooks", s.handleWebhookList)
	s.mux.HandleFunc("POST /webhooks", s.handleWebhookCreate)
	s.mux.HandleFunc("GET /webhooks/{id}", s.handleWebhookGet)
	s.mux.HandleFunc("PATCH /webhooks/{id}", s.handleWebhookPatch)
	s.mux.HandleFunc("DELETE /webhooks/{id}", s.handleWebhookDelete)
}

func webhookBodyRules(mode bodyMode) []fieldRule {
	return []fieldRule{
		{
			Name:  fieldWebhookEventTypes,
			Kind:  kindStringList,
			Check: enumList(fieldWebhookEventTypes, webhookEventTypes),
		},
		{
			Name:     keyName,
			Kind:     kindString,
			Required: mode == bodyCreate,
			MinLen:   1,
			MaxLen:   maxWebhookNameLength,
			Check:    nonBlank(keyName),
		},
		{
			Name:     fieldWebhookURL,
			Kind:     kindString,
			Required: mode == bodyCreate,
			MinLen:   1,
			MaxLen:   maxWebhookURLLength,
			Check:    validateHTTPURL(fieldWebhookURL),
		},
		{Name: fieldWebhookVersion, Kind: kindString, Enum: webhookVersions},
	}
}

func sanitizeWebhookBody(body Record, mode bodyMode) (Record, error) {
	headers, hasHeaders := body[fieldWebhookCustomHeaders]
	withoutHeaders := make(Record, len(body))
	for key, value := range body {
		if key != fieldWebhookCustomHeaders {
			withoutHeaders[key] = value
		}
	}
	sanitized, err := sanitizeBody(withoutHeaders, webhookBodyRules(mode), mode)
	if err != nil {
		return nil, err
	}
	if !hasHeaders {
		return sanitized, nil
	}
	if headers == nil {
		if mode == bodyCreate {
			return nil, invalidField(fieldWebhookCustomHeaders, "cannot be null")
		}
		sanitized[fieldWebhookCustomHeaders] = nil
		return sanitized, nil
	}
	cleaned, err := sanitizeCustomHeaders(headers)
	if err != nil {
		return nil, err
	}
	sanitized[fieldWebhookCustomHeaders] = cleaned
	return sanitized, nil
}

func sanitizeCustomHeaders(raw any) ([]any, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, invalidField(
			fieldWebhookCustomHeaders,
			"must be an array of {key, value} objects",
		)
	}
	if len(items) > maxWebhookCustomHeaders {
		return nil, invalidField(
			fieldWebhookCustomHeaders,
			fmt.Sprintf("supports at most %d headers", maxWebhookCustomHeaders),
		)
	}
	out := make([]any, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for idx, item := range items {
		path := fmt.Sprintf("%s[%d]", fieldWebhookCustomHeaders, idx)
		header, isMap := anyAsMap(item)
		if !isMap {
			return nil, invalidField(path, "must be an object")
		}
		key, keyOK := header[fieldHeaderKey].(string)
		if !keyOK || key == "" {
			return nil, invalidField(path+".key", "is required")
		}
		if utf8.RuneCountInString(key) > maxWebhookHeaderKeyLength {
			return nil, invalidField(
				path+".key",
				fmt.Sprintf("must be at most %d characters", maxWebhookHeaderKeyLength),
			)
		}
		if !isHTTPToken(key) {
			return nil, invalidField(path+".key", "must be a valid HTTP header name")
		}
		value, valueOK := header[keyValue].(string)
		if !valueOK || value == "" {
			return nil, invalidField(path+".value", "is required")
		}
		if utf8.RuneCountInString(value) > maxWebhookHeaderValueLength {
			return nil, invalidField(
				path+".value",
				fmt.Sprintf("must be at most %d characters", maxWebhookHeaderValueLength),
			)
		}
		if strings.ContainsAny(value, "\r\n\x00") {
			return nil, invalidField(path+".value", "must not contain line breaks")
		}
		canonical := http.CanonicalHeaderKey(key)
		if _, dup := seen[canonical]; dup {
			return nil, invalidField(path+".key", fmt.Sprintf("duplicates header %q", key))
		}
		seen[canonical] = struct{}{}
		out = append(out, map[string]any{fieldHeaderKey: key, keyValue: value})
	}
	return out, nil
}

func isHTTPToken(value string) bool {
	for idx := 0; idx < len(value); idx++ {
		char := value[idx]
		if isASCIIAlphanumeric(char) || strings.IndexByte(httpTokenSymbols, char) >= 0 {
			continue
		}
		return false
	}
	return value != ""
}

func webhookCustomHeaders(record Record) []webhookCustomHeader {
	items := listOf(record[fieldWebhookCustomHeaders])
	out := make([]webhookCustomHeader, 0, len(items))
	for _, item := range items {
		header, ok := anyAsMap(item)
		if !ok {
			continue
		}
		key := stringOf(header[fieldHeaderKey])
		if key == "" {
			continue
		}
		out = append(out, webhookCustomHeader{Key: key, Value: stringOf(header[keyValue])})
	}
	return out
}

func webhookSubscribes(record Record, eventTypes []string, eventType string) bool {
	if _, declared := record[fieldWebhookEventTypes]; !declared {
		return true
	}
	return eventType == "" || slices.Contains(eventTypes, eventType)
}

func generatedWebhookSecret(webhookID string) string {
	digest := sha256.Sum256([]byte("samsara-sim-webhook-secret|" + strings.TrimSpace(webhookID)))
	return base64.StdEncoding.EncodeToString(digest[:])
}

func webhookView(record Record) Record {
	id := recordID(record)
	out := Record{
		keyID:                     id,
		keyName:                   stringValue(record, keyName),
		fieldWebhookURL:           stringValue(record, fieldWebhookURL),
		fieldWebhookVersion:       defaultWebhookVersion,
		fieldWebhookSecretKey:     stringValue(record, fieldWebhookSecretKey),
		fieldWebhookEventTypes:    stringsAsAny(stringListValues(record[fieldWebhookEventTypes])),
		fieldWebhookCustomHeaders: []any{},
	}
	if version := stringValue(record, fieldWebhookVersion); version != "" {
		out[fieldWebhookVersion] = version
	}
	if out[fieldWebhookSecretKey] == "" {
		out[fieldWebhookSecretKey] = generatedWebhookSecret(id)
	}
	headers := webhookCustomHeaders(record)
	if len(headers) > 0 {
		rendered := make([]any, 0, len(headers))
		for _, header := range headers {
			rendered = append(rendered, map[string]any{
				fieldHeaderKey: header.Key,
				keyValue:       header.Value,
			})
		}
		out[fieldWebhookCustomHeaders] = rendered
	}
	return out
}

func applyWebhookWrite(record, body Record) {
	for _, field := range []string{
		keyName,
		fieldWebhookURL,
		fieldWebhookVersion,
		fieldWebhookEventTypes,
		fieldWebhookCustomHeaders,
	} {
		raw, ok := body[field]
		if !ok {
			continue
		}
		if raw == nil {
			delete(record, field)
			continue
		}
		record[field] = cloneAny(raw)
	}
}

func (s *Server) handleWebhookList(writer http.ResponseWriter, request *http.Request) {
	records, err := s.store.List(ResourceWebhooks)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	ids := csvQueryValues(request.URL.Query(), "ids")
	wanted := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	views := make([]Record, 0, len(records))
	for _, record := range records {
		if len(wanted) > 0 {
			if _, ok := wanted[recordID(record)]; !ok {
				continue
			}
		}
		views = append(views, webhookView(record))
	}
	s.respondPage(writer, request, views, "|webhook-list")
}

func (s *Server) handleWebhookGet(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	record, err := s.store.Get(ResourceWebhooks, id)
	if err != nil {
		s.writeError(writer, webhookLookupError(err, id))
		return
	}
	s.respondJSON(writer, request, requestSignature(request)+"|webhook-get", webhookView(record))
}

func (s *Server) handleWebhookCreate(writer http.ResponseWriter, request *http.Request) {
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeWebhookBody(body, bodyCreate)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	record := Record{
		fieldWebhookVersion:       defaultWebhookVersion,
		fieldWebhookEventTypes:    []any{},
		fieldWebhookCustomHeaders: []any{},
	}
	applyWebhookWrite(record, sanitized)
	now := s.simNow()
	var created Record
	err = s.store.Transact(func(tx *storeTx) error {
		inserted, insertErr := tx.insert(ResourceWebhooks, record, now)
		if insertErr != nil {
			return insertErr
		}
		inserted[fieldWebhookSecretKey] = generatedWebhookSecret(recordID(inserted))
		created = inserted
		return nil
	})
	if err != nil {
		s.writeError(writer, err)
		return
	}
	s.respondJSON(
		writer,
		request,
		requestSignature(request)+"|webhook-create",
		webhookView(created),
	)
}

func (s *Server) handleWebhookPatch(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	body, err := readRecordBody(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	sanitized, err := sanitizeWebhookBody(body, bodyPatch)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	now := s.simNow()
	var updated Record
	err = s.store.Transact(func(tx *storeTx) error {
		var updateErr error
		updated, updateErr = tx.update(ResourceWebhooks, id, func(record Record) error {
			applyWebhookWrite(record, sanitized)
			return nil
		}, now)
		return updateErr
	})
	if err != nil {
		s.writeError(writer, webhookLookupError(err, id))
		return
	}
	s.respondJSON(writer, request, requestSignature(request)+"|webhook-patch", webhookView(updated))
}

func (s *Server) handleWebhookDelete(writer http.ResponseWriter, request *http.Request) {
	id, err := pathID(request)
	if err != nil {
		s.writeError(writer, err)
		return
	}
	err = s.store.Transact(func(tx *storeTx) error {
		return tx.remove(ResourceWebhooks, id)
	})
	if err != nil {
		s.writeError(writer, webhookLookupError(err, id))
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func webhookLookupError(err error, id string) error {
	if errors.Is(err, ErrRecordNotFound) {
		return notFound(labelWebhook, id)
	}
	return err
}
