package ediports

type CredentialKind string

const (
	CredentialKindSecret  = CredentialKind("Secret")
	CredentialKindSetting = CredentialKind("Setting")
)

type CredentialRequirement struct {
	Keys     []string       `json:"keys"`
	Label    string         `json:"label"`
	Kind     CredentialKind `json:"kind"`
	Required bool           `json:"required"`
	Missing  bool           `json:"missing"`
}
