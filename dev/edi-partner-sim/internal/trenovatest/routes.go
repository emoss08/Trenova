package trenovatest

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

type handleFunc func(string, func(http.ResponseWriter, *http.Request, map[string]any))

func (f *Fake) ladingRoutes(handle handleFunc) {
	handle("GET /api/v1/edi/lading/projects/{a}/{b}/{$}",
		func(w http.ResponseWriter, r *http.Request, body map[string]any) {
			switch {
			case r.PathValue("a") == "by-partner":
				r.SetPathValue("id", r.PathValue("b"))
				f.partnerProject(w, r, body)
			case r.PathValue("b") == "versions":
				r.SetPathValue("id", r.PathValue("a"))
				f.listVersions(w, r, body)
			default:
				problem(w, http.StatusNotFound, "no route")
			}
		})
	handle("POST /api/v1/edi/lading/projects/{$}", f.createProject)
	handle("POST /api/v1/edi/lading/projects/{id}/drafts/{$}", f.newDraft)
	handle("GET /api/v1/edi/lading/versions/{id}/{$}", f.getVersion)
	handle("PUT /api/v1/edi/lading/versions/{id}/{$}", f.saveDraft)
	handle("POST /api/v1/edi/lading/versions/{id}/test/{$}", f.testVersion)
	handle("POST /api/v1/edi/lading/versions/{id}/certify/{$}", f.certify)
	handle("POST /api/v1/edi/lading/versions/{id}/activate/{$}", f.activate)
	handle("POST /api/v1/edi/lading/versions/{id}/replays/{$}",
		func(w http.ResponseWriter, _ *http.Request, body map[string]any) {
			reply(w, map[string]any{"id": "ldr_1", "map": body["map"], "status": "Running"})
		})
	handle("GET /api/v1/edi/lading/replays/{id}/{$}",
		func(w http.ResponseWriter, _ *http.Request, _ map[string]any) {
			reply(w, map[string]any{"id": "ldr_1", "status": "Completed",
				"summary": map[string]any{"total": 1, "same": 1}})
		})
}

func (f *Fake) partnerProject(w http.ResponseWriter, r *http.Request, _ map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, project := range f.projects {
		if project["ediPartnerId"] == r.PathValue("id") {
			reply(w, project)
			return
		}
	}
	problem(w, http.StatusNotFound, "no project")
}

func (f *Fake) createProject(w http.ResponseWriter, _ *http.Request, body map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if body["role"] != "carrier" && body["role"] != "broker" {
		problem(w, http.StatusBadRequest, "role")
		return
	}
	id := f.id("ldp")
	project := map[string]any{"id": id, "ediPartnerId": body["partnerId"],
		"activeVersionId": "", "version": 1.0}
	f.projects[id] = project
	f.newVersion(id, 1, []any{map[string]any{"path": "partner.lading", "source": "starter"}},
		[]any{})
	reply(w, project)
}

func (f *Fake) newVersion(projectID string, number int, files, samples []any) map[string]any {
	id := f.id("ldv")
	version := map[string]any{"id": id, "projectId": projectID, "number": number,
		"status": "Draft", "files": files, "samples": samples, "version": 1.0,
		"manifest": map[string]any{"maps": []any{
			map[string]any{"name": "tenderIn", "direction": "inbound", "source": "x12"},
		}}}
	f.versions[id] = version
	return version
}

func (f *Fake) listVersions(w http.ResponseWriter, r *http.Request, _ map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	items := []any{}
	for _, version := range f.versions {
		if version["projectId"] == r.PathValue("id") {
			items = append(items, version)
		}
	}
	reply(w, map[string]any{"items": items})
}

func (f *Fake) newDraft(w http.ResponseWriter, r *http.Request, body map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	from, ok := f.versions[fmt.Sprint(body["fromVersionId"])]
	if !ok {
		problem(w, http.StatusBadRequest, "no version to start from")
		return
	}
	number := 1
	for _, version := range f.versions {
		if version["projectId"] == r.PathValue("id") {
			number++
		}
	}
	files, _ := from["files"].([]any)
	samples, _ := from["samples"].([]any)
	reply(w, f.newVersion(r.PathValue("id"), number, files, samples))
}

func (f *Fake) getVersion(w http.ResponseWriter, r *http.Request, _ map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	version, ok := f.versions[r.PathValue("id")]
	if !ok {
		problem(w, http.StatusNotFound, "no version")
		return
	}
	samples, _ := version["samples"].([]any)
	reply(w, map[string]any{"version": version, "samples": samples})
}

func (f *Fake) saveDraft(w http.ResponseWriter, r *http.Request, body map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	version, ok := f.versions[r.PathValue("id")]
	if !ok || version["status"] != "Draft" {
		problem(w, http.StatusUnprocessableEntity, "only a draft is saved")
		return
	}
	if version["version"] != body["version"] {
		problem(w, http.StatusConflict, "stale draft")
		return
	}
	version["files"], version["samples"] = body["files"], body["samples"]
	version["version"] = version["version"].(float64) + 1
	reply(w, version)
}

func (f *Fake) readiness() map[string]any {
	if f.NotReady {
		return map[string]any{"errors": 0, "failing": 1, "tests": 3, "ready": false}
	}
	return map[string]any{"errors": 0, "failing": 0, "tests": 3, "ready": true}
}

func (f *Fake) testVersion(w http.ResponseWriter, _ *http.Request, _ map[string]any) {
	tests := []any{map[string]any{"path": "flows/tender.lading", "name": "a tender",
		"passed": true}}
	if f.NotReady {
		tests = append(tests, map[string]any{"path": "flows/tender.lading",
			"name": "a broken test", "passed": false,
			"failures": []any{map[string]any{"message": map[string]any{
				"message": "expected held", "args": []any{}}}}})
	}
	reply(w, map[string]any{"tests": tests, "readiness": f.readiness()})
}

func (f *Fake) certify(w http.ResponseWriter, r *http.Request, _ map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.NotReady {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		data, _ := sonic.Marshal(map[string]any{"title": "Invalid", "errors": []any{
			map[string]any{"path": "tests", "message": "1 tests fail"}}})
		_, _ = w.Write(data)
		return
	}
	version := f.versions[r.PathValue("id")]
	version["status"] = "Certified"
	reply(w, version)
}

func (f *Fake) activate(w http.ResponseWriter, r *http.Request, _ map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	version := f.versions[r.PathValue("id")]
	if version["status"] != "Certified" {
		problem(w, http.StatusUnprocessableEntity, "only a certified version is activated")
		return
	}
	project := f.projects[fmt.Sprint(version["projectId"])]
	if previous, ok := f.versions[fmt.Sprint(project["activeVersionId"])]; ok {
		previous["status"] = "Superseded"
	}
	version["status"] = "Active"
	project["activeVersionId"] = version["id"]
	reply(w, version)
}

func (f *Fake) operationRoutes(handle handleFunc) {
	handle("GET /api/v1/edi/flows/instances/{$}", f.listFlows)
	handle("GET /api/v1/edi/flows/instances/{id}/{$}",
		func(w http.ResponseWriter, r *http.Request, _ map[string]any) {
			f.mu.Lock()
			defer f.mu.Unlock()
			reply(w, map[string]any{"instance": f.flows[r.PathValue("id")], "events": []any{}})
		})
	handle("POST /api/v1/edi/flows/instances/{id}/release/{$}",
		func(w http.ResponseWriter, r *http.Request, _ map[string]any) {
			f.mu.Lock()
			defer f.mu.Unlock()
			if fl := f.flows[r.PathValue("id")]; fl != nil {
				fl.Status, fl.Reason = "Open", nil
			}
			reply(w, map[string]any{})
		})
	handle("GET /api/v1/edi/transfers/{id}/{$}", f.getTransfer)
	handle("POST /api/v1/edi/transfers/{id}/approve/{$}", f.decide(true))
	handle("POST /api/v1/edi/transfers/{id}/reject/{$}", f.decide(false))
	handle("GET /api/v1/edi/messages/{$}",
		func(w http.ResponseWriter, _ *http.Request, _ map[string]any) {
			f.mu.Lock()
			defer f.mu.Unlock()
			reply(w, map[string]any{"results": f.messages})
		})
	handle("POST /api/v1/shipment-moves/{id}/assignment/{$}", f.assign)
	handle("POST /api/v1/shipment-moves/{move}/stops/{stop}/record-actual/{$}", f.recordStop)
	handle("POST /api/v1/billing/invoices/from-shipments/{$}", f.createInvoice)
	handle("POST /api/v1/billing/invoices/{id}/post/{$}", f.postInvoice)
}

func (f *Fake) listFlows(w http.ResponseWriter, r *http.Request, _ map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	query := r.URL.Query()
	results := []any{}
	for _, fl := range sortedFlows(f.flows) {
		if partner := query.Get("partnerId"); partner != "" && fl.PartnerID != partner {
			continue
		}
		if key := query.Get("query"); key != "" && !strings.Contains(fl.Key, key) {
			continue
		}
		if statuses := query["status"]; len(statuses) > 0 && !contains(statuses, fl.Status) {
			continue
		}
		results = append(results, fl)
	}
	reply(w, map[string]any{"results": results})
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func (f *Fake) getTransfer(w http.ResponseWriter, r *http.Request, _ map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fl := f.flows[f.transfers[r.PathValue("id")]]
	if fl == nil {
		problem(w, http.StatusNotFound, "no transfer")
		return
	}
	status, why := "Approved", ""
	if f.BuildFail != "" && fl.ShipmentID != "" {
		status, why = "Failed", f.BuildFail
	}
	reply(w, map[string]any{"id": fl.TransferID, "targetShipmentId": fl.ShipmentID,
		"status": status, "failureReason": why, "tenderPayload": f.payloads[fl.TransferID]})
}

func (f *Fake) decide(accept bool) func(http.ResponseWriter, *http.Request, map[string]any) {
	return func(w http.ResponseWriter, r *http.Request, body map[string]any) {
		f.mu.Lock()
		fl := f.flows[f.transfers[r.PathValue("id")]]
		if fl == nil {
			f.mu.Unlock()
			problem(w, http.StatusNotFound, "no transfer")
			return
		}
		code := "D"
		if accept {
			if unresolved := f.approveMappings(fl, body); len(unresolved) > 0 {
				f.mu.Unlock()
				unmapped(w, unresolved)
				return
			}
			code = "A"
			s := &shipment{
				ID: f.id("shp"), Status: "New", ProNumber: "PRO" + fl.Key, BOL: "BOL" + fl.Key,
				CustomerID: "cus_1", ServiceID: "svc_1", moveID: f.id("sm"), key: fl.Key,
				flowID: fl.ID, Total: fmt.Sprintf("%d.%02d", fl.total/100, fl.total%100),
			}
			s.stops = []*stop{
				{ID: f.id("stp"), Type: "Pickup", Sequence: 0, Status: "New", Location: "loc_1"},
				{ID: f.id("stp"), Type: "Delivery", Sequence: 1, Status: "New", Location: "loc_2"},
			}
			f.shipments[s.ID] = s
			fl.ShipmentID = s.ID
		} else {
			fl.Status = "Closed"
		}
		fl.UpdatedAt = time.Now().UnixNano()
		partnerID, key, scac := fl.PartnerID, fl.Key, f.Envelope.SCAC
		f.mu.Unlock()
		reply(w, map[string]any{"id": r.PathValue("id"), "status": "Processing"})
		go f.send(partnerID, "GF", "990", key,
			"990~B1*"+scac+"*"+key+"*20260302*"+code)
	}
}

func (f *Fake) shipmentByMove(moveID string) *shipment {
	for _, s := range f.shipments {
		if s.moveID == moveID {
			return s
		}
	}
	return nil
}

func (f *Fake) assign(w http.ResponseWriter, r *http.Request, body map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.shipmentByMove(r.PathValue("id"))
	if s == nil || body["primaryWorkerId"] != "wrk_1" || body["tractorId"] != "trk_1" {
		problem(w, http.StatusBadRequest, "no such move or resources")
		return
	}
	s.assigned, s.Status = true, "Assigned"
	reply(w, map[string]any{"id": f.id("asg")})
}

func (f *Fake) recordStop(w http.ResponseWriter, r *http.Request, body map[string]any) {
	f.mu.Lock()
	s := f.shipmentByMove(r.PathValue("move"))
	if s == nil {
		f.mu.Unlock()
		problem(w, http.StatusNotFound, "no move")
		return
	}
	var target *stop
	for _, candidate := range s.stops {
		if candidate.ID == r.PathValue("stop") {
			target = candidate
		}
	}
	if target == nil {
		f.mu.Unlock()
		problem(w, http.StatusNotFound, "no stop")
		return
	}
	now := time.Now().Unix()
	last := target == s.stops[len(s.stops)-1]
	code := ""
	switch {
	case body["action"] == "Arrive" && target.Type == "Pickup":
		target.Arrival, code = &now, "X3"
	case body["action"] == "Depart" && target.Type == "Pickup":
		target.Departure, code = &now, "AF"
	case body["action"] == "Arrive":
		target.Arrival, code = &now, "X1"
	default:
		target.Departure, code = &now, "CD"
		if last {
			code = "D1"
			s.Status = "Completed"
		}
	}
	if s.Status == "Assigned" {
		s.Status = "InTransit"
	}
	partnerID, key, scac := f.flows[s.flowID].PartnerID, s.key, f.Envelope.SCAC
	f.mu.Unlock()
	reply(w, map[string]any{"id": r.PathValue("move")})
	go f.send(partnerID, "QM", "214", key,
		"214~B10*"+s.ProNumber+"*"+key+"*"+scac+"~LX*1~AT7*"+code+"*NS")
}

func (f *Fake) createInvoice(w http.ResponseWriter, _ *http.Request, body map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids, _ := body["shipmentIds"].([]any)
	if len(ids) != 1 || f.shipments[fmt.Sprint(ids[0])] == nil {
		problem(w, http.StatusBadRequest, "one shipment")
		return
	}
	id := f.id("inv")
	f.invoices[id] = fmt.Sprint(ids[0])
	reply(w, map[string]any{"id": id, "status": "Draft"})
}

func (f *Fake) postInvoice(w http.ResponseWriter, r *http.Request, _ map[string]any) {
	f.mu.Lock()
	s := f.shipments[f.invoices[r.PathValue("id")]]
	if s == nil {
		f.mu.Unlock()
		problem(w, http.StatusNotFound, "no invoice")
		return
	}
	s.Status = "Invoiced"
	fl := f.flows[s.flowID]
	fl.Status = "Closed"
	partnerID, key, payment := fl.PartnerID, s.key, fl.payment
	total := fl.total
	f.mu.Unlock()
	reply(w, map[string]any{"id": r.PathValue("id"), "status": "Posted"})
	go f.send(partnerID, "IM", "210", key,
		fmt.Sprintf("210~B3**%s*%s*%s**20260302*%d", r.PathValue("id"), key, payment, total))
}

func (f *Fake) graphQL(w http.ResponseWriter, _ *http.Request, body map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	query := fmt.Sprint(body["query"])
	variables, _ := body["variables"].(map[string]any)
	if f.referenceGraphQL(w, query, variables) {
		return
	}
	if strings.Contains(query, "shipments(") {
		edges := []any{}
		for _, s := range f.shipments {
			edges = append(edges, map[string]any{"node": f.shipmentView(s)})
		}
		reply(w, map[string]any{"data": map[string]any{"shipments": map[string]any{
			"edges": edges}}})
		return
	}
	s := f.shipments[fmt.Sprint(variables["id"])]
	if s == nil {
		reply(w, map[string]any{"data": map[string]any{"shipment": nil}})
		return
	}
	reply(w, map[string]any{"data": map[string]any{"shipment": f.shipmentView(s)}})
}

func (f *Fake) shipmentView(s *shipment) map[string]any {
	stops := make([]any, 0, len(s.stops))
	for _, st := range s.stops {
		stops = append(stops, st)
	}
	var assignment any
	if s.assigned {
		assignment = map[string]any{"id": "asg_1"}
	}
	return map[string]any{
		"id": s.ID, "status": s.Status, "proNumber": s.ProNumber, "bol": s.BOL,
		"customerId": s.CustomerID, "serviceTypeId": s.ServiceID, "shipmentTypeId": "sht_1",
		"totalChargeAmount": s.Total,
		"moves": []any{map[string]any{"id": s.moveID, "sequence": 0, "status": s.Status,
			"assignment": assignment, "stops": stops}},
	}
}
