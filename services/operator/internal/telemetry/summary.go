package telemetry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/bytedance/sonic"
)

const maxSummary = 8 << 20

type Summary struct {
	Node SummaryNode  `json:"node"`
	Pods []SummaryPod `json:"pods"`
}

type SummaryNode struct {
	FS *FSStats `json:"fs,omitempty"`
}

type FSStats struct {
	CapacityBytes *uint64 `json:"capacityBytes,omitempty"`
	UsedBytes     *uint64 `json:"usedBytes,omitempty"`
}

type SummaryPod struct {
	PodRef  PodRef          `json:"podRef"`
	Volumes []SummaryVolume `json:"volume,omitempty"`
}

type PodRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type SummaryVolume struct {
	Name          string  `json:"name"`
	PVCRef        *PodRef `json:"pvcRef,omitempty"`
	CapacityBytes *uint64 `json:"capacityBytes,omitempty"`
	UsedBytes     *uint64 `json:"usedBytes,omitempty"`
}

type Summaries interface {
	Summary(ctx context.Context, kubelet string) (Summary, error)
}

type KubeletSummaries struct {
	client    *http.Client
	tokenFile string
}

type KubeletParams struct {
	CA        []byte
	TokenFile string
}

func NewKubeletSummaries(p KubeletParams) (*KubeletSummaries, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(p.CA) {
		return nil, errors.New("telemetry: the cluster's CA bundle holds no certificate")
	}
	if p.TokenFile == "" {
		return nil, errors.New(
			"telemetry: the kubelet is read with the ServiceAccount token, and its file is not named",
		)
	}
	return &KubeletSummaries{
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool},
				ForceAttemptHTTP2: true,
			},
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		tokenFile: p.TokenFile,
	}, nil
}

func (k *KubeletSummaries) Summary(ctx context.Context, kubelet string) (Summary, error) {
	raw, err := os.ReadFile(k.tokenFile)
	if err != nil {
		return Summary{}, fmt.Errorf("the ServiceAccount token: %w", err)
	}
	token := strings.TrimSpace(string(raw))
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://"+kubelet+"/stats/summary",
		http.NoBody,
	)
	if err != nil {
		return Summary{}, fmt.Errorf("the kubelet's stats summary at %s: %w", kubelet, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	res, err := k.client.Do(req)
	if err != nil {
		return Summary{}, fmt.Errorf("the kubelet's stats summary at %s: %w", kubelet, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return Summary{}, fmt.Errorf("the kubelet at %s answered %d", kubelet, res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxSummary+1))
	if err != nil {
		return Summary{}, fmt.Errorf("the kubelet's stats summary at %s: %w", kubelet, err)
	}
	if len(body) > maxSummary {
		return Summary{}, fmt.Errorf(
			"the kubelet's stats summary at %s is larger than %d bytes",
			kubelet,
			maxSummary,
		)
	}
	var out Summary
	if err = sonic.Unmarshal(body, &out); err != nil {
		return Summary{}, fmt.Errorf("the kubelet's stats summary at %s: %w", kubelet, err)
	}
	return out, nil
}
