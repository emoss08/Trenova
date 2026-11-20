package trenovatest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/transcript"
	"github.com/emoss08/trenova/shared/as2"
)

const (
	csrfToken  = "fake-csrf"
	apiPrefix  = "/api/v1"
	inboundURL = apiPrefix + "/edi/as2/inbound/"
)

type Envelope struct {
	UsQualifier   string
	UsID          string
	UsGroup       string
	ThemQualifier string
	ThemID        string
	ThemGroup     string
	SCAC          string
}

type flow struct {
	ID         string `json:"id"`
	PartnerID  string `json:"ediPartnerId"`
	Flow       string `json:"flow"`
	Key        string `json:"key"`
	Status     string `json:"status"`
	Reason     any    `json:"reason"`
	ShipmentID string `json:"shipmentId"`
	TransferID string `json:"transferId"`
	UpdatedAt  int64  `json:"updatedAt"`
	total      int64
	payment    string
}

type stop struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Sequence  int    `json:"sequence"`
	Status    string `json:"status"`
	Location  string `json:"locationId"`
	Arrival   *int64 `json:"actualArrival"`
	Departure *int64 `json:"actualDeparture"`
}

type shipment struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	ProNumber  string `json:"proNumber"`
	BOL        string `json:"bol"`
	CustomerID string `json:"customerId"`
	ServiceID  string `json:"serviceTypeId"`
	Total      string `json:"totalChargeAmount"`
	moveID     string
	assigned   bool
	stops      []*stop
	key        string
	flowID     string
}

type outbound struct {
	ID          string `json:"id"`
	Direction   string `json:"direction"`
	Set         string `json:"transactionSet"`
	Interchange string `json:"interchangeControlNumber"`
	Group       string `json:"groupControlNumber"`
	Control     string `json:"transactionControlNumber"`
	AckStatus   string `json:"ackStatus"`
	key         string
}

type Fake struct {
	t         testing.TB
	Server    *httptest.Server
	Envelope  Envelope
	NotReady  bool
	BuildFail string
	mu        sync.Mutex
	seq       int
	control   int
	calls     []string
	partners  map[string]map[string]any
	profiles  map[string]map[string]any
	projects  map[string]map[string]any
	versions  map[string]map[string]any
	flows     map[string]*flow
	transfers map[string]string
	shipments map[string]*shipment
	invoices  map[string]string
	messages  []*outbound
	tokens    map[string]string
	payloads  map[string]map[string]any
	mapped    map[string]map[string]string
	options   map[string][]map[string]any
	created   map[string]int
}

func New(t testing.TB, envelope Envelope) *Fake {
	t.Helper()
	f := &Fake{
		t:         t,
		Envelope:  envelope,
		control:   500,
		partners:  map[string]map[string]any{},
		profiles:  map[string]map[string]any{},
		projects:  map[string]map[string]any{},
		versions:  map[string]map[string]any{},
		flows:     map[string]*flow{},
		transfers: map[string]string{},
		shipments: map[string]*shipment{},
		invoices:  map[string]string{},
		tokens:    map[string]string{},
		payloads:  map[string]map[string]any{},
		mapped:    map[string]map[string]string{},
		options:   seedOptions(),
		created:   map[string]int{},
	}
	f.Server = httptest.NewServer(f.routes())
	t.Cleanup(f.Server.Close)
	return f
}

func (f *Fake) API() string {
	return f.Server.URL + apiPrefix
}

func (f *Fake) Calls(prefix string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, call := range f.calls {
		if strings.HasPrefix(call, prefix) {
			out = append(out, call)
		}
	}
	return out
}

func (f *Fake) Count(call string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, candidate := range f.calls {
		if candidate == call {
			count++
		}
	}
	return count
}

func (f *Fake) Flows() []flow {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]flow, 0, len(f.flows))
	for _, fl := range f.flows {
		out = append(out, *fl)
	}
	return out
}

func (f *Fake) SetFlow(key, status, why string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, fl := range f.flows {
		if fl.Key == key {
			fl.Status, fl.Reason = status, reason(why)
		}
	}
}

func (f *Fake) id(prefix string) string {
	f.seq++
	return fmt.Sprintf("%s_%04d", prefix, f.seq)
}

func reason(text string) any {
	if text == "" {
		return nil
	}
	return map[string]any{"message": text, "args": []any{}}
}

func (f *Fake) routes() http.Handler {
	mux := http.NewServeMux()
	handle := func(pattern string, fn func(http.ResponseWriter, *http.Request, map[string]any)) {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			body := map[string]any{}
			if r.Body != nil {
				raw, _ := io.ReadAll(r.Body)
				if len(bytes.TrimSpace(raw)) > 0 && raw[0] == '{' {
					_ = sonic.Unmarshal(raw, &body)
				} else if len(raw) > 0 {
					body["_raw"] = raw
				}
			}
			if r.Method != http.MethodGet && !strings.HasSuffix(r.URL.Path, "/auth/login") &&
				!strings.HasPrefix(r.URL.Path, inboundURL) &&
				r.Header.Get("X-CSRF-Token") != csrfToken {
				problem(w, http.StatusForbidden, "CSRF token missing")
				return
			}
			f.mu.Lock()
			f.calls = append(f.calls, r.Method+" "+strings.TrimPrefix(r.URL.Path, apiPrefix))
			f.mu.Unlock()
			fn(w, r, body)
		})
	}
	handle("POST /api/v1/auth/login", f.login)
	handle("GET /api/v1/edi/partners/{$}", f.listPartners)
	handle("POST /api/v1/edi/partners/{$}", f.savePartner)
	handle("GET /api/v1/edi/partners/{id}/{$}", f.getPartner)
	handle("PUT /api/v1/edi/partners/{id}/{$}", f.savePartner)
	handle("POST /api/v1/edi/communication-profiles/{$}", f.saveProfile)
	handle("PUT /api/v1/edi/communication-profiles/{id}/{$}", f.saveProfile)
	handle("POST /api/v1/edi/communication-profiles/{id}/test-connection/{$}",
		func(w http.ResponseWriter, _ *http.Request, _ map[string]any) {
			reply(w, map[string]any{"success": true, "checks": []any{}})
		})
	handle("POST /api/v1/edi/communication-profiles/{id}/poll/{$}",
		func(w http.ResponseWriter, _ *http.Request, _ map[string]any) {
			reply(w, map[string]any{"stagedFileIds": []string{}})
		})
	f.ladingRoutes(handle)
	f.operationRoutes(handle)
	f.mappingRoutes(handle)
	handle("POST /api/v1/edi/as2/inbound/{token}/{$}", f.inbound)
	handle("POST /graphql", f.graphQL)
	return mux
}

func reply(w http.ResponseWriter, payload any) {
	data, _ := sonic.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func problem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	data, _ := sonic.Marshal(map[string]any{
		"title": http.StatusText(status), "status": status, "detail": detail,
	})
	_, _ = w.Write(data)
}

func (f *Fake) login(w http.ResponseWriter, _ *http.Request, body map[string]any) {
	if body["password"] != "admin123!" {
		problem(w, http.StatusUnauthorized, "wrong password")
		return
	}
	reply(w, map[string]any{"csrfToken": csrfToken, "activeRoleIds": []string{"rol_1"}})
}

func (f *Fake) listPartners(w http.ResponseWriter, r *http.Request, _ map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	query := strings.ToLower(r.URL.Query().Get("query"))
	results := []any{}
	for _, p := range f.partners {
		if strings.Contains(strings.ToLower(fmt.Sprint(p["code"])), query) {
			results = append(results, p)
		}
	}
	reply(w, map[string]any{"results": results})
}

func (f *Fake) getPartner(w http.ResponseWriter, r *http.Request, _ map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.partners[r.PathValue("id")]
	if !ok {
		problem(w, http.StatusNotFound, "no partner")
		return
	}
	out := clone(p)
	if profile, found := f.profiles[fmt.Sprint(p["defaultTransportId"])]; found {
		out["defaultTransport"] = f.profileView(profile)
	}
	reply(w, out)
}

func (f *Fake) savePartner(w http.ResponseWriter, r *http.Request, body map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := r.PathValue("id")
	if id == "" {
		id = f.id("edip")
		body["version"] = 1.0
	} else {
		current, ok := f.partners[id]
		if !ok || current["version"] != body["version"] {
			problem(w, http.StatusConflict, "stale partner version")
			return
		}
		body["version"] = current["version"].(float64) + 1
	}
	body["id"] = id
	delete(body, "defaultTransport")
	f.partners[id] = body
	reply(w, body)
}

func (f *Fake) saveProfile(w http.ResponseWriter, r *http.Request, body map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := r.PathValue("id")
	if id == "" {
		id = f.id("edicp")
		body["version"] = 1.0
		f.tokens[id] = "tok" + id
	} else {
		current, ok := f.profiles[id]
		if !ok || current["version"] != body["version"] {
			problem(w, http.StatusConflict, "stale profile version")
			return
		}
		body["version"] = current["version"].(float64) + 1
	}
	body["id"] = id
	f.profiles[id] = body
	reply(w, f.profileView(body))
}

func (f *Fake) profileView(profile map[string]any) map[string]any {
	out := clone(profile)
	delete(out, "secrets")
	if out["method"] == "AS2" {
		out["as2InboundPath"] = inboundURL + f.tokens[fmt.Sprint(out["id"])] + "/"
	}
	return out
}

func clone(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func (f *Fake) profileByToken(token string) map[string]any {
	for id, candidate := range f.tokens {
		if candidate == token {
			return f.profiles[id]
		}
	}
	return nil
}

func (f *Fake) inbound(w http.ResponseWriter, r *http.Request, body map[string]any) {
	raw, _ := body["_raw"].([]byte)
	parsed, err := as2.ParseMessage(r.Header.Get("Content-Type"), raw, &as2.ParseMessageOptions{
		EntityHeaders: r.Header,
	})
	if err != nil {
		problem(w, http.StatusBadRequest, err.Error())
		return
	}
	f.mu.Lock()
	profile := f.profileByToken(r.PathValue("token"))
	f.mu.Unlock()
	if profile == nil {
		problem(w, http.StatusNotFound, "no profile")
		return
	}
	read, err := new(oracle.GuideSet).Judge(parsed.Payload)
	if err != nil {
		problem(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	partnerID := fmt.Sprint(profile["ediPartnerId"])
	for _, ic := range read.Interchanges {
		for _, g := range ic.Groups {
			if g.FunctionalID == "FA" {
				f.reconcile(g)
				continue
			}
			for _, rep := range g.Reports {
				if rep.Set == "204" {
					f.tender(partnerID, rep)
				}
			}
			f.acknowledge(partnerID, g)
		}
	}
}

func segment(rep *oracle.Report, id string) *oracle.Segment {
	for index := range rep.Segments {
		if rep.Segments[index].ID == id {
			return &rep.Segments[index]
		}
	}
	return nil
}

func (f *Fake) tender(partnerID string, rep *oracle.Report) {
	key := transcript.KeyOf(rep)
	purpose := "00"
	if b2a := segment(rep, "B2A"); b2a != nil {
		purpose = b2a.Element(1)
	}
	total := int64(0)
	if l3 := segment(rep, "L3"); l3 != nil {
		total, _ = strconv.ParseInt(l3.Element(5), 10, 64)
	}
	payment := "PP"
	if b2 := segment(rep, "B2"); b2 != nil && b2.Element(6) != "" {
		payment = b2.Element(6)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var existing *flow
	for _, fl := range f.flows {
		if fl.Key == key && fl.PartnerID == partnerID {
			existing = fl
		}
	}
	switch {
	case existing == nil && purpose == "00":
		id := f.id("edfi")
		transfer := f.id("edilt")
		f.flows[id] = &flow{
			ID: id, PartnerID: partnerID, Flow: "tender", Key: key, Status: "Open",
			TransferID: transfer, UpdatedAt: time.Now().UnixNano(), total: total, payment: payment,
		}
		f.transfers[transfer] = id
		f.payloads[transfer] = tenderPayload(rep)
	case existing == nil:
		return
	case purpose == "04" && total != existing.total:
		existing.Status, existing.Reason = "Held", reason("Acme changed the rate on "+key)
	case purpose == "01":
		s := f.shipments[existing.ShipmentID]
		if s != nil && s.assigned {
			existing.Status = "Held"
			existing.Reason = reason("Acme cancelled " + key + " after dispatch")
			return
		}
		if s != nil {
			s.Status = "Canceled"
		}
		existing.Status = "Closed"
	}
}

func (f *Fake) acknowledge(partnerID string, g *oracle.Group) {
	var body []string
	body = append(body, "997~AK1*"+g.FunctionalID+"*"+g.ControlNumber)
	for _, rep := range g.Reports {
		body[0] += "~AK2*" + rep.Set + "*" + rep.ControlNumber + "~AK5*A"
	}
	body[0] += fmt.Sprintf("~AK9*A*%d*%d*%d", len(g.Reports), len(g.Reports), len(g.Reports))
	go f.send(partnerID, "FA", "", "", body...)
}

func (f *Fake) reconcile(g *oracle.Group) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, rep := range g.Reports {
		ak1, ak5 := segment(rep, "AK1"), segment(rep, "AK5")
		if ak1 == nil {
			continue
		}
		status := "Accepted"
		if ak5 != nil && ak5.Element(1) == "R" {
			status = "Rejected"
		}
		for _, m := range f.messages {
			if m.Group == ak1.Element(2) {
				m.AckStatus = status
				for _, fl := range f.flows {
					if status == "Rejected" && fl.Key == m.key {
						fl.Status = "Escalated"
						fl.Reason = reason("The partner rejected " + m.Set + " " + m.Control)
					}
				}
			}
		}
	}
}

func (f *Fake) send(partnerID, group, set, key string, bodies ...string) {
	f.mu.Lock()
	var profile map[string]any
	for _, candidate := range f.profiles {
		if candidate["ediPartnerId"] == partnerID {
			profile = candidate
		}
	}
	f.control++
	control := f.control
	if set != "" {
		f.messages = append(f.messages, &outbound{
			ID: f.id("edimsg"), Direction: "Outbound", Set: set,
			Interchange: fmt.Sprintf("%09d", control), Group: strconv.Itoa(control),
			Control: "0001", AckStatus: "Pending", key: key,
		})
	}
	f.mu.Unlock()
	if profile == nil {
		f.t.Errorf("trenovatest: no profile for partner %s", partnerID)
		return
	}
	config, _ := profile["config"].(map[string]any)
	payload := f.x12(control, group, bodies...)
	built, err := as2.BuildMessage(&as2.BuildMessageOptions{
		From:     fmt.Sprint(config["localAS2Id"]),
		To:       fmt.Sprint(config["partnerAS2Id"]),
		FileName: "trenova.edi",
		Payload:  []byte(payload),
	})
	if err != nil {
		f.t.Errorf("trenovatest: build AS2: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprint(config["endpointUrl"]), bytes.NewReader(built.Body))
	if err != nil {
		f.t.Errorf("trenovatest: %v", err)
		return
	}
	for key, values := range built.Headers {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}
	request.Header.Set("Content-Type", built.ContentType)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}
	_ = response.Body.Close()
}

func (f *Fake) x12(control int, group string, bodies ...string) string {
	e := f.Envelope
	pad := func(value string) string { return fmt.Sprintf("%-15s", value) }
	var out strings.Builder
	fmt.Fprintf(&out,
		"ISA*00*          *00*          *%s*%s*%s*%s*260302*0900*^*00401*%09d*0*T*>~\n",
		e.UsQualifier, pad(e.UsID), e.ThemQualifier, pad(e.ThemID), control)
	fmt.Fprintf(&out, "GS*%s*%s*%s*20260302*0900*%d*X*004010~\n", group, e.UsGroup, e.ThemGroup,
		control)
	for index, body := range bodies {
		segments := strings.Split(body, "~")
		fmt.Fprintf(&out, "ST*%s*%04d~\n", segments[0], index+1)
		for _, s := range segments[1:] {
			fmt.Fprintf(&out, "%s~\n", s)
		}
		fmt.Fprintf(&out, "SE*%d*%04d~\n", len(segments)+1, index+1)
	}
	fmt.Fprintf(&out, "GE*%d*%d~\nIEA*1*%09d~\n", len(bodies), control, control)
	return out.String()
}

func sortedFlows(flows map[string]*flow) []*flow {
	out := make([]*flow, 0, len(flows))
	for _, fl := range flows {
		out = append(out, fl)
	}
	slices.SortFunc(out, func(a, b *flow) int { return int(b.UpdatedAt - a.UpdatedAt) })
	return out
}
