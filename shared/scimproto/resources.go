package scimproto

import (
	"strings"

	"github.com/bytedance/sonic"
)

const (
	MediaType = "application/scim+json"

	SchemaUser                  = "urn:ietf:params:scim:schemas:core:2.0:User"
	SchemaGroup                 = "urn:ietf:params:scim:schemas:core:2.0:Group"
	SchemaEnterpriseUser        = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
	SchemaListResponse          = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	SchemaPatchOp               = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
	SchemaError                 = "urn:ietf:params:scim:api:messages:2.0:Error"
	SchemaServiceProviderConfig = "urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"
	SchemaResourceType          = "urn:ietf:params:scim:schemas:core:2.0:ResourceType"
	SchemaSchema                = "urn:ietf:params:scim:schemas:core:2.0:Schema"

	ResourceTypeUser  = "User"
	ResourceTypeGroup = "Group"
)

type Meta struct {
	ResourceType string `json:"resourceType"`
	Created      string `json:"created,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
	Location     string `json:"location,omitempty"`
	Version      string `json:"version,omitempty"`
}

type Name struct {
	Formatted       string `json:"formatted,omitempty"`
	FamilyName      string `json:"familyName,omitempty"`
	GivenName       string `json:"givenName,omitempty"`
	MiddleName      string `json:"middleName,omitempty"`
	HonorificPrefix string `json:"honorificPrefix,omitempty"`
	HonorificSuffix string `json:"honorificSuffix,omitempty"`
}

func (n *Name) Full() string {
	if n == nil {
		return ""
	}
	if formatted := strings.TrimSpace(n.Formatted); formatted != "" {
		return formatted
	}

	parts := make([]string, 0, 3)
	for _, part := range []string{n.GivenName, n.MiddleName, n.FamilyName} {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}

	return strings.Join(parts, " ")
}

type MultiValue struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type,omitempty"`
	Primary Bool   `json:"primary,omitempty"`
}

type MemberRef struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Ref     string `json:"$ref,omitempty"`
	Type    string `json:"type,omitempty"`
}

type User struct {
	Schemas           []string     `json:"schemas"`
	ID                string       `json:"id,omitempty"`
	ExternalID        string       `json:"externalId,omitempty"`
	UserName          string       `json:"userName"`
	Name              *Name        `json:"name,omitempty"`
	DisplayName       string       `json:"displayName,omitempty"`
	Title             string       `json:"title,omitempty"`
	Timezone          string       `json:"timezone,omitempty"`
	Locale            string       `json:"locale,omitempty"`
	PreferredLanguage string       `json:"preferredLanguage,omitempty"`
	Active            *Bool        `json:"active,omitempty"`
	Emails            []MultiValue `json:"emails,omitempty"`
	PhoneNumbers      []MultiValue `json:"phoneNumbers,omitempty"`
	Meta              *Meta        `json:"meta,omitempty"`
}

func (u *User) PrimaryEmail() string {
	return preferredValue(u.Emails)
}

func (u *User) PrimaryPhone() string {
	return preferredValue(u.PhoneNumbers)
}

func (u *User) FullName() string {
	if full := u.Name.Full(); full != "" {
		return full
	}
	if display := strings.TrimSpace(u.DisplayName); display != "" {
		return display
	}

	return strings.TrimSpace(u.UserName)
}

func (u *User) IsActive() bool {
	return u.Active == nil || bool(*u.Active)
}

func preferredValue(values []MultiValue) string {
	for _, candidate := range values {
		if candidate.Primary && strings.TrimSpace(candidate.Value) != "" {
			return strings.TrimSpace(candidate.Value)
		}
	}
	for _, candidate := range values {
		if strings.EqualFold(candidate.Type, "work") && strings.TrimSpace(candidate.Value) != "" {
			return strings.TrimSpace(candidate.Value)
		}
	}
	for _, candidate := range values {
		if trimmed := strings.TrimSpace(candidate.Value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}

type Group struct {
	Schemas     []string    `json:"schemas"`
	ID          string      `json:"id,omitempty"`
	ExternalID  string      `json:"externalId,omitempty"`
	DisplayName string      `json:"displayName"`
	Members     []MemberRef `json:"members,omitempty"`
	Meta        *Meta       `json:"meta,omitempty"`
}

func (g *Group) MemberIDs() []string {
	ids := make([]string, 0, len(g.Members))
	for _, member := range g.Members {
		if trimmed := strings.TrimSpace(member.Value); trimmed != "" {
			ids = append(ids, trimmed)
		}
	}

	return ids
}

type ListResponse struct {
	Schemas      []string `json:"schemas"`
	TotalResults int      `json:"totalResults"`
	StartIndex   int      `json:"startIndex"`
	ItemsPerPage int      `json:"itemsPerPage"`
	Resources    []any    `json:"Resources"`
}

func NewListResponse(resources []any, total, startIndex int) *ListResponse {
	if resources == nil {
		resources = []any{}
	}

	return &ListResponse{
		Schemas:      []string{SchemaListResponse},
		TotalResults: total,
		StartIndex:   startIndex,
		ItemsPerPage: len(resources),
		Resources:    resources,
	}
}

func DecodeUser(body []byte) (*User, error) {
	user := new(User)
	if err := sonic.Unmarshal(body, user); err != nil {
		return nil, InvalidSyntax("The request body is not a valid SCIM User")
	}
	if !hasSchema(user.Schemas, SchemaUser) {
		return nil, InvalidValue("The request body must name the core User schema")
	}

	return user, nil
}

func DecodeGroup(body []byte) (*Group, error) {
	group := new(Group)
	if err := sonic.Unmarshal(body, group); err != nil {
		return nil, InvalidSyntax("The request body is not a valid SCIM Group")
	}
	if !hasSchema(group.Schemas, SchemaGroup) {
		return nil, InvalidValue("The request body must name the core Group schema")
	}

	return group, nil
}

func hasSchema(schemas []string, want string) bool {
	for _, schema := range schemas {
		if strings.EqualFold(strings.TrimSpace(schema), want) {
			return true
		}
	}

	return false
}
