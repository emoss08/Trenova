package reply

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/goccy/go-yaml"
)

const (
	samplesFile = "replies.yaml"
	ladingDir   = "lading"
	samplesDir  = "samples"
)

const (
	ReplyResponse = "response"
	ReplyStatus   = "status"
	ReplyStray    = "stray"
	ReplyInvoice  = "invoice"
)

var (
	replyKinds       = []string{ReplyResponse, ReplyStatus, ReplyStray, ReplyInvoice}
	errNoTender      = errors.New("reply: the tender sample holds no 204")
	errNoSuchStatus  = errors.New("reply: the status sequence has no such status")
	errUnknownReply  = errors.New("reply: unknown reply kind")
	errTenderRejects = errors.New("reply: the carrier's guide rejects the tender sample")
)

type SampleSpecs struct {
	Tender  TenderSample  `yaml:"tender"`
	Replies []ReplySample `yaml:"replies"`
}

type TenderSample struct {
	File    string `yaml:"file"`
	Seed    uint64 `yaml:"seed"`
	Control int64  `yaml:"control"`
	At      string `yaml:"at"`
}

type ReplySample struct {
	File     string   `yaml:"file"`
	Control  int64    `yaml:"control"`
	Reply    string   `yaml:"reply"`
	Status   string   `yaml:"status"`
	Last     bool     `yaml:"last"`
	Scenario Scenario `yaml:"scenario"`
}

func (c *Carrier) SampleSpecs() (*SampleSpecs, error) {
	path := filepath.Join(c.Definition.Dir, samplesFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	specs := &SampleSpecs{}
	if err := yaml.UnmarshalWithOptions(raw, specs, yaml.DisallowUnknownField()); err != nil {
		return nil, partner.Problems{{File: path, Message: yaml.FormatError(err, false, true)}}
	}
	var problems partner.Problems
	add := func(where, format string, args ...any) {
		problems = append(problems, partner.Problem{
			File:    path,
			Path:    where,
			Message: fmt.Sprintf(format, args...),
		})
	}
	if specs.Tender.File == "" || specs.Tender.Control < 1 {
		add("tender", "needs a file and a control number")
	}
	seen := map[string]bool{specs.Tender.File: true}
	for index, spec := range specs.Replies {
		where := fmt.Sprintf("replies[%d]", index)
		if spec.File == "" || seen[spec.File] {
			add(where+".file", "%q is empty or listed twice", spec.File)
		}
		seen[spec.File] = true
		if !slices.Contains(replyKinds, spec.Reply) {
			add(where+".reply", "%q is not one of %v", spec.Reply, replyKinds)
		}
		if spec.Control < 1 || spec.Control > 999_999_999 {
			add(where+".control", "%d is not an interchange control number", spec.Control)
		}
		if (spec.Reply == ReplyStatus) != (spec.Status != "") {
			add(where+".status", "names the status a status reply picks, and only there")
		}
	}
	if len(problems) > 0 {
		return nil, problems
	}
	return specs, nil
}

func (c *Carrier) TenderSamplePath(specs *SampleSpecs) string {
	return filepath.Join(c.Definition.Dir, ladingDir, samplesDir, specs.Tender.File)
}

func (c *Carrier) GenerateReplies(
	specs *SampleSpecs,
	tenderRaw []byte,
) ([]partner.GeneratedSample, error) {
	tenders, judgement, err := c.Receive(tenderRaw)
	if err != nil {
		return nil, err
	}
	if !judgement.Accepted() {
		return nil, errTenderRejects
	}
	if len(tenders) == 0 {
		return nil, errNoTender
	}
	tender := tenders[0]
	out := make([]partner.GeneratedSample, 0, len(specs.Replies))
	for _, spec := range specs.Replies {
		reply, buildErr := c.sampleReply(tender, spec)
		if buildErr != nil {
			return nil, fmt.Errorf("%s: %w", spec.File, buildErr)
		}
		controls := partner.Controls{
			Interchange: partner.NewCounter(spec.Control),
			Group:       partner.NewCounter(spec.Control),
		}
		written, writeErr := c.Write(reply, controls)
		if writeErr != nil {
			return nil, fmt.Errorf("%s: %w", spec.File, writeErr)
		}
		out = append(out, partner.GeneratedSample{
			Path:    filepath.Join(c.Definition.Dir, ladingDir, samplesDir, spec.File),
			Content: written.Bytes,
		})
	}
	return out, nil
}

func (c *Carrier) sampleReply(tender *Tender, spec ReplySample) (Reply, error) {
	switch spec.Reply {
	case ReplyResponse:
		return c.Response(tender, spec.Scenario)
	case ReplyStray:
		return c.Stray(tender, spec.Scenario)
	case ReplyInvoice:
		return c.Invoice(tender, spec.Scenario)
	case ReplyStatus:
		statuses, err := c.Statuses(tender, spec.Scenario)
		if err != nil {
			return Reply{}, err
		}
		if spec.Last {
			slices.Reverse(statuses)
		}
		for _, status := range statuses {
			if StatusCode(status) == spec.Status {
				return status, nil
			}
		}
		return Reply{}, fmt.Errorf("%w: %s", errNoSuchStatus, spec.Status)
	default:
		return Reply{}, fmt.Errorf("%w: %s", errUnknownReply, spec.Reply)
	}
}

func StatusCode(reply Reply) string {
	for _, segment := range reply.Transaction.Segments {
		if segment.ID == "AT7" && len(segment.Elements) > 0 && len(segment.Elements[0]) > 0 {
			return segment.Elements[0][0]
		}
	}
	return ""
}
