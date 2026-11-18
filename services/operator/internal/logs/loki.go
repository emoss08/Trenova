package logs

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cenkalti/backoff/v5"
)

const (
	LabelComponent = "component"
	LabelPod       = "pod"
	LabelContainer = "container"

	lokiAttemptTimeout = 30 * time.Second
	lokiTries          = 4
	lokiMaxBody        = 16 << 20
	lokiMaxError       = 512
)

var (
	ErrPageTooLarge  = errors.New("the log store's answer is too large for one page")
	componentPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$`)
)

type Entry struct {
	Time      time.Time
	Pod       string
	Container string
	Line      string
	stream    string
}

type PageQuery struct {
	Component string
	Start     time.Time
	End       time.Time
	Limit     int
}

type Source interface {
	Components(ctx context.Context, since, until time.Time) ([]string, error)
	NewestFirst(ctx context.Context, q PageQuery) ([]Entry, error)
}

type Loki struct {
	base   *url.URL
	client *http.Client
}

func NewLoki(rawURL string, client *http.Client) (*Loki, error) {
	base, err := url.Parse(strings.TrimSuffix(rawURL, "/") + "/")
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return nil, fmt.Errorf("the Loki URL %q is not an http(s) URL", rawURL)
	}
	if client == nil {
		client = &http.Client{Timeout: lokiAttemptTimeout + 5*time.Second}
	}
	return &Loki{base: base, client: client}, nil
}

func ValidComponent(name string) bool {
	return componentPattern.MatchString(name)
}

func (l *Loki) Components(ctx context.Context, since, until time.Time) ([]string, error) {
	u := l.base.JoinPath("loki", "api", "v1", "label", LabelComponent, "values")
	u.RawQuery = url.Values{
		"start": {nanos(since)},
		"end":   {nanos(until)},
	}.Encode()
	body, err := l.get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("list the sources Loki keeps: %w", err)
	}
	var out struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}
	if err = sonic.Unmarshal(body, &out); err != nil || out.Status != "success" {
		return nil, errors.New("list the sources Loki keeps: the answer is not a label list")
	}
	slices.Sort(out.Data)
	return slices.Compact(out.Data), nil
}

func (l *Loki) NewestFirst(ctx context.Context, q PageQuery) ([]Entry, error) {
	if !ValidComponent(q.Component) {
		return nil, fmt.Errorf("%q is not a component name", q.Component)
	}
	u := l.base.JoinPath("loki", "api", "v1", "query_range")
	u.RawQuery = url.Values{
		"query":     {fmt.Sprintf(`{%s=%q}`, LabelComponent, q.Component)},
		"start":     {nanos(q.Start)},
		"end":       {nanos(q.End)},
		"limit":     {strconv.Itoa(q.Limit)},
		"direction": {"backward"},
	}.Encode()
	body, err := l.get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("read the %s logs Loki keeps: %w", q.Component, err)
	}
	return parseStreams(body)
}

type streamsAnswer struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Stream map[string]string `json:"stream"`
			Values [][]string        `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

func parseStreams(body []byte) ([]Entry, error) {
	var ans streamsAnswer
	if err := sonic.Unmarshal(body, &ans); err != nil {
		return nil, fmt.Errorf("the log store's answer is not JSON: %w", err)
	}
	if ans.Status != "success" || ans.Data.ResultType != "streams" {
		return nil, fmt.Errorf(
			"the log store answered %q with %q rather than log streams", ans.Status, ans.Data.ResultType,
		)
	}
	var out []Entry
	for _, s := range ans.Data.Result {
		key := streamKey(s.Stream)
		for _, v := range s.Values {
			if len(v) < 2 {
				return nil, errors.New("the log store's answer has an entry without a timestamp and a line")
			}
			ns, err := strconv.ParseInt(v[0], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("the log store's answer has a timestamp that is not nanoseconds: %q", v[0])
			}
			out = append(out, Entry{
				Time:      time.Unix(0, ns).UTC(),
				Pod:       s.Stream[LabelPod],
				Container: s.Stream[LabelContainer],
				Line:      v[1],
				stream:    key,
			})
		}
	}
	slices.SortStableFunc(out, newestFirst)
	return out, nil
}

func newestFirst(a, b Entry) int {
	if c := b.Time.Compare(a.Time); c != 0 {
		return c
	}
	return cmp.Compare(a.stream, b.stream)
}

func streamKey(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(labels[k])
		b.WriteByte(0)
	}
	return b.String()
}

type lokiError struct {
	status int
	body   string
}

func (e *lokiError) Error() string {
	if e.body == "" {
		return fmt.Sprintf("the log store answered %d", e.status)
	}
	return fmt.Sprintf("the log store answered %d: %s", e.status, e.body)
}

func (l *Loki) get(ctx context.Context, u *url.URL) ([]byte, error) {
	exp := backoff.NewExponentialBackOff()
	exp.InitialInterval = 500 * time.Millisecond
	exp.MaxInterval = 5 * time.Second
	return backoff.Retry(ctx, func() ([]byte, error) {
		body, err := l.once(ctx, u)
		var le *lokiError
		if errors.As(err, &le) && le.status != http.StatusTooManyRequests && le.status < 500 {
			return nil, backoff.Permanent(err)
		}
		return body, err
	}, backoff.WithBackOff(exp), backoff.WithMaxTries(lokiTries))
}

func (l *Loki) once(ctx context.Context, u *url.URL) ([]byte, error) {
	run, cancel := context.WithTimeout(ctx, lokiAttemptTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(run, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, backoff.Permanent(err)
	}
	req.Header.Set("Accept", "application/json")
	res, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, lokiMaxError))
		return nil, &lokiError{status: res.StatusCode, body: strings.Join(strings.Fields(string(msg)), " ")}
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, lokiMaxBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > lokiMaxBody {
		return nil, backoff.Permanent(ErrPageTooLarge)
	}
	return body, nil
}

func nanos(t time.Time) string {
	return strconv.FormatInt(t.UnixNano(), 10)
}
