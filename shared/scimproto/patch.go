package scimproto

import (
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
)

const (
	OpAdd     = "add"
	OpReplace = "replace"
	OpRemove  = "remove"
)

type PatchRequest struct {
	Schemas    []string         `json:"schemas"`
	Operations []PatchOperation `json:"Operations"`
}

type PatchOperation struct {
	Op    string                 `json:"op"`
	Path  string                 `json:"path,omitempty"`
	Value sonic.NoCopyRawMessage `json:"value,omitempty"`
}

func DecodePatch(body []byte, maxOperations int) (*PatchRequest, error) {
	req := new(PatchRequest)
	if err := sonic.Unmarshal(body, req); err != nil {
		return nil, InvalidSyntax("The request body is not a valid SCIM PatchOp")
	}
	if !hasSchema(req.Schemas, SchemaPatchOp) {
		return nil, InvalidValue("The request body must name the PatchOp schema")
	}
	if len(req.Operations) == 0 {
		return nil, InvalidValue("The patch has no operations")
	}
	if len(req.Operations) > maxOperations {
		return nil, TooMany("The patch has more operations than one request may carry")
	}

	for i := range req.Operations {
		op := strings.ToLower(strings.TrimSpace(req.Operations[i].Op))
		switch op {
		case OpAdd, OpReplace, OpRemove:
			req.Operations[i].Op = op
		default:
			return nil, InvalidSyntax(
				"The patch operation " + req.Operations[i].Op + " is not valid",
			)
		}
	}

	return req, nil
}

type patchPath struct {
	attribute string
	filter    *Filter
	sub       string
}

func parsePatchPath(raw string) (*patchPath, error) {
	stripped := stripSchemaPrefix(raw)
	if strings.HasPrefix(strings.ToLower(stripped), "urn:") {
		return &patchPath{attribute: strings.ToLower(stripped)}, nil
	}

	open := strings.IndexByte(stripped, '[')
	if open < 0 {
		attribute, sub, _ := strings.Cut(strings.ToLower(stripped), ".")
		return &patchPath{attribute: attribute, sub: sub}, nil
	}

	end := strings.LastIndexByte(stripped, ']')
	if end < open {
		return nil, InvalidPath("The path " + raw + " is not valid")
	}

	filter, err := ParseFilter(stripped[open+1:end], "type", "value", "display")
	if err != nil {
		return nil, InvalidPath("The path " + raw + " has a filter that is not supported")
	}

	return &patchPath{
		attribute: strings.ToLower(stripped[:open]),
		filter:    filter,
		sub:       strings.ToLower(strings.TrimPrefix(stripped[end+1:], ".")),
	}, nil
}

func decodeString(raw sonic.NoCopyRawMessage) (string, error) {
	var value any
	if err := sonic.Unmarshal(raw, &value); err != nil {
		return "", InvalidValue("The patch value is not valid JSON")
	}

	switch typed := value.(type) {
	case nil:
		return "", nil
	case string:
		return strings.TrimSpace(typed), nil
	case bool:
		return strconv.FormatBool(typed), nil
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), nil
	default:
		return "", InvalidValue("The patch value must be a single value")
	}
}

func decodeBool(raw sonic.NoCopyRawMessage) (bool, error) {
	var value Bool
	if err := sonic.Unmarshal(raw, &value); err != nil {
		return false, InvalidValue("The patch value must be true or false")
	}

	return bool(value), nil
}

func decodeObject(raw sonic.NoCopyRawMessage) (map[string]sonic.NoCopyRawMessage, error) {
	var object map[string]sonic.NoCopyRawMessage
	if err := sonic.Unmarshal(raw, &object); err != nil {
		return nil, InvalidValue("A patch without a path must carry an object of attributes")
	}

	return object, nil
}

func decodeMultiValues(raw sonic.NoCopyRawMessage) ([]MultiValue, error) {
	var values []MultiValue
	if err := sonic.Unmarshal(raw, &values); err == nil {
		return values, nil
	}

	var single MultiValue
	if err := sonic.Unmarshal(raw, &single); err != nil {
		return nil, InvalidValue("The patch value must be a list of values")
	}

	return []MultiValue{single}, nil
}

func decodeMembers(raw sonic.NoCopyRawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var members []MemberRef
	if err := sonic.Unmarshal(raw, &members); err != nil {
		var single MemberRef
		if singleErr := sonic.Unmarshal(raw, &single); singleErr != nil {
			return nil, InvalidValue("The members value must be a list of members")
		}
		members = []MemberRef{single}
	}

	ids := make([]string, 0, len(members))
	for _, member := range members {
		if trimmed := strings.TrimSpace(member.Value); trimmed != "" {
			ids = append(ids, trimmed)
		}
	}

	return ids, nil
}
