package contentscanportsmock

import (
	"bytes"
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
)

type Gate struct {
	err      error
	mu       sync.Mutex
	requests []*contentscanports.ScanRequest
}

func NewCleanGate() *Gate {
	return &Gate{}
}

func NewRefusingGate(err error) *Gate {
	return &Gate{err: err}
}

func NewInfectedGate(signature string) *Gate {
	return &Gate{err: &contentscanports.InfectedError{Signature: signature}}
}

func (g *Gate) Check(
	_ context.Context,
	req *contentscanports.ScanRequest,
) (*contentscanports.ScanResult, error) {
	g.mu.Lock()
	g.requests = append(g.requests, req)
	g.mu.Unlock()
	if g.err != nil {
		return nil, g.err
	}
	return &contentscanports.ScanResult{Verdict: contentscanports.VerdictClean}, nil
}

func (g *Gate) Stream(
	_ context.Context,
	req *contentscanports.StreamRequest,
) (contentscanports.Stream, error) {
	return &stream{gate: g, req: req}, nil
}

type stream struct {
	gate *Gate
	req  *contentscanports.StreamRequest
	body bytes.Buffer
}

func (s *stream) Write(p []byte) (int, error) {
	return s.body.Write(p)
}

func (s *stream) Finish() (*contentscanports.ScanResult, error) {
	return s.gate.Check(context.Background(), &contentscanports.ScanRequest{
		TenantInfo: s.req.TenantInfo,
		Source:     s.req.Source,
		FileName:   s.req.FileName,
		Content:    s.body.Bytes(),
	})
}

func (s *stream) Abort() {}

func (g *Gate) Requests() []*contentscanports.ScanRequest {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]*contentscanports.ScanRequest(nil), g.requests...)
}
