package lading

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/jsonutils"
)

type File struct {
	Path   string `json:"path"`
	Source string `json:"source"`
}

type Sample struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type Project struct {
	Files   []File   `json:"files"`
	Samples []Sample `json:"samples"`
}

type Org struct {
	Name string `json:"name"`
	SCAC string `json:"scac"`
	Zone string `json:"zone"`
}

type CustomField struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

type Context struct {
	Org          Org                      `json:"org"`
	CustomFields map[string][]CustomField `json:"customFields,omitempty"`
}

// Encoded is a project and its context encoded once and passed to every call unchanged: the
// module keeps the programs it checked by these exact bytes, so repeated calls skip the check.
type Encoded struct {
	Project jsonutils.RawJSON `json:"project"`
	Context jsonutils.RawJSON `json:"context"`
}

func Encode(project *Project, context *Context) (*Encoded, error) {
	if len(project.Files) > MaxFiles {
		return nil, fmt.Errorf("lading: a project holds at most %d files", MaxFiles)
	}
	if len(project.Samples) > MaxSamples {
		return nil, fmt.Errorf("lading: a project holds at most %d samples", MaxSamples)
	}
	if project.Files == nil {
		project.Files = []File{}
	}
	if project.Samples == nil {
		project.Samples = []Sample{}
	}
	encodedProject, err := sonic.ConfigStd.Marshal(project)
	if err != nil {
		return nil, fmt.Errorf("lading: encoding the project: %w", err)
	}
	if context == nil {
		context = &Context{}
	}
	encodedContext, err := sonic.ConfigStd.Marshal(context)
	if err != nil {
		return nil, fmt.Errorf("lading: encoding the context: %w", err)
	}
	return &Encoded{Project: encodedProject, Context: encodedContext}, nil
}

func Digest(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
