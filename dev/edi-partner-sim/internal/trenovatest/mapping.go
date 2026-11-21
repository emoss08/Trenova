package trenovatest

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
)

var entityOrder = []string{
	"Customer", "ServiceType", "FormulaTemplate", "Location", "Commodity",
}

func seedOptions() map[string][]map[string]any {
	option := func(id, label, code string) map[string]any {
		return map[string]any{"id": id, "label": label, "description": "",
			"meta": map[string]any{"code": code}}
	}
	return map[string][]map[string]any{
		"SERVICE_TYPE": {option("st_1", "EXP", "EXP"), option("st_2", "STD", "STD")},
		"FORMULA_TEMPLATE": {
			option("ft_1", "Per Mile", ""), option("ft_2", "Flat Rate", ""),
		},
		"LOCATION_CATEGORY": {
			option("lc_1", "Main Terminal", ""), option("lc_2", "Customer Drop Point", ""),
		},
	}
}

func (f *Fake) Created(resource string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.created[resource]
}

func (f *Fake) Mapped(partnerID string) map[string]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]string, len(f.mapped[partnerID]))
	for key, value := range f.mapped[partnerID] {
		out[key] = value
	}
	return out
}

func (f *Fake) Records(resource string) []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]map[string]any, 0, len(f.options[resource]))
	for _, option := range f.options[resource] {
		out = append(out, clone(option))
	}
	return out
}

func tenderPayload(rep *oracle.Report) map[string]any {
	required := map[string][]string{"ServiceType": {"DEFAULT"}, "FormulaTemplate": {"DEFAULT"}}
	add := func(entity, code string) {
		if code != "" && !slices.Contains(required[entity], code) {
			required[entity] = append(required[entity], code)
		}
	}
	stops, commodities := []any{}, []any{}
	var party map[string]any
	for index := range rep.Segments {
		s := &rep.Segments[index]
		switch s.ID {
		case "N1":
			party = nil
			code := s.Element(4)
			if s.Element(1) == "BT" {
				add("Customer", code)
				continue
			}
			party = map[string]any{"locationId": code, "locationCode": code,
				"locationName": s.Element(2)}
			stops = append(stops, party)
			add("Location", code)
		case "N3":
			if party != nil {
				party["locationAddressLine1"] = s.Element(1)
			}
		case "N4":
			if party != nil {
				party["locationCity"], party["locationStateCode"] = s.Element(1), s.Element(2)
				party["locationPostalCode"] = s.Element(3)
			}
		case "L5":
			code := s.Element(3)
			if code == "" {
				code = strings.ToUpper(s.Element(2))
			}
			commodities = append(commodities, map[string]any{"commodityId": code,
				"commodityLabel": code, "commodityDescription": s.Element(2)})
			add("Commodity", code)
		}
	}
	return map[string]any{
		"moves":                    []any{map[string]any{"stops": stops}},
		"commodities":              commodities,
		"requiredMappingEntityIds": required,
	}
}

func (f *Fake) mappingRoutes(handle handleFunc) {
	handle("GET /api/v1/edi/transfers/{id}/mapping-preview/{$}",
		func(w http.ResponseWriter, r *http.Request, _ map[string]any) {
			f.mu.Lock()
			defer f.mu.Unlock()
			fl := f.flows[f.transfers[r.PathValue("id")]]
			if fl == nil {
				problem(w, http.StatusNotFound, "no transfer")
				return
			}
			all := f.resolutions(fl)
			resolved, unresolved := []any{}, []any{}
			for _, row := range all {
				if row["resolved"] == true {
					resolved = append(resolved, row)
				} else {
					unresolved = append(unresolved, row)
				}
			}
			reply(w, map[string]any{"resolved": resolved, "unresolved": unresolved, "all": all})
		})
	handle("POST /api/v1/locations/{$}", f.createLocation)
}

func (f *Fake) resolutions(fl *flow) []map[string]any {
	payload := f.payloads[fl.TransferID]
	required, _ := payload["requiredMappingEntityIds"].(map[string][]string)
	mapped := f.mapped[fl.PartnerID]
	var rows []map[string]any
	for _, entity := range entityOrder {
		for _, source := range required[entity] {
			target := mapped[entity+"|"+source]
			rows = append(rows, map[string]any{"entityType": entity, "sourceId": source,
				"sourceLabel": source, "targetId": target, "resolved": target != ""})
		}
	}
	return rows
}

func (f *Fake) approveMappings(fl *flow, body map[string]any) []map[string]any {
	items, _ := body["mappings"].([]any)
	if f.mapped[fl.PartnerID] == nil {
		f.mapped[fl.PartnerID] = map[string]string{}
	}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		target := fmt.Sprint(item["targetId"])
		if !f.known(target) {
			continue
		}
		f.mapped[fl.PartnerID][fmt.Sprint(item["entityType"])+"|"+
			fmt.Sprint(item["sourceId"])] = target
	}
	var unresolved []map[string]any
	for _, row := range f.resolutions(fl) {
		if row["resolved"] != true {
			unresolved = append(unresolved, row)
		}
	}
	return unresolved
}

func (f *Fake) known(id string) bool {
	for _, options := range f.options {
		for _, option := range options {
			if option["id"] == id {
				return true
			}
		}
	}
	return false
}

func unmapped(w http.ResponseWriter, unresolved []map[string]any) {
	fields := make([]any, 0, len(unresolved))
	for _, row := range unresolved {
		fields = append(fields, map[string]any{"path": []any{row["entityType"]},
			"message": fmt.Sprintf("Mapping is required for source ID %s", row["sourceId"])})
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusBadRequest)
	data, _ := sonic.Marshal(map[string]any{"title": "Validation Failed", "errors": fields})
	_, _ = w.Write(data)
}

func (f *Fake) createLocation(w http.ResponseWriter, _ *http.Request, body map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, field := range []string{"name", "addressLine1", "city", "stateId", "postalCode",
		"locationCategoryId"} {
		if fmt.Sprint(body[field]) == "" || body[field] == nil {
			problem(w, http.StatusBadRequest, field+" is required")
			return
		}
	}
	if !f.known(fmt.Sprint(body["locationCategoryId"])) ||
		!strings.HasPrefix(fmt.Sprint(body["stateId"]), "us_") {
		problem(w, http.StatusBadRequest, "unknown category or state")
		return
	}
	id := f.id("loc")
	code := fmt.Sprintf("LOC%04d", f.seq)
	f.remember("LOCATION", map[string]any{"id": id, "label": body["name"], "description": code,
		"meta": map[string]any{"code": code, "city": body["city"],
			"postalCode": body["postalCode"]}})
	reply(w, map[string]any{"id": id, "code": code, "name": body["name"]})
}

func (f *Fake) remember(resource string, option map[string]any) {
	f.options[resource] = append(f.options[resource], option)
	f.created[resource]++
}

func (f *Fake) referenceGraphQL(
	w http.ResponseWriter,
	query string,
	variables map[string]any,
) bool {
	input, _ := variables["input"].(map[string]any)
	data := map[string]any{}
	switch {
	case strings.Contains(query, "selectOptions("):
		data["selectOptions"] = map[string]any{"edges": f.selectOptions(input)}
	case strings.Contains(query, "tractors("):
		data["tractors"] = map[string]any{"edges": []any{
			map[string]any{"node": map[string]any{"id": "trk_2", "primaryWorkerId": "wrk_2",
				"inProgressMoveId": "sm_busy"}},
			map[string]any{"node": map[string]any{"id": "trk_1", "primaryWorkerId": "wrk_1",
				"inProgressMoveId": nil}},
		}}
	case strings.Contains(query, "updateCustomer("):
		profile, _ := input["billingProfile"].(map[string]any)
		option := f.option("CUSTOMER", fmt.Sprint(variables["id"]))
		if option == nil || profile == nil || input["code"] == nil {
			reply(w, map[string]any{"errors": []any{map[string]any{"message": "bad customer"}}})
			return true
		}
		option["ediInvoice"] = profile["ediInvoiceEnabled"] == true &&
			profile["emailInvoiceEnabled"] == false
		f.created["CUSTOMER_UPDATE"]++
		data["updateCustomer"] = map[string]any{"id": option["id"]}
	case strings.Contains(query, "customer("):
		option := f.option("CUSTOMER", fmt.Sprint(variables["id"]))
		if option == nil {
			data["customer"] = nil
			break
		}
		meta, _ := option["meta"].(map[string]any)
		edi := option["ediInvoice"] == true
		data["customer"] = map[string]any{"status": "Active", "code": meta["code"],
			"name": option["label"], "version": 0, "billingProfile": map[string]any{
				"ediInvoiceEnabled": edi, "emailInvoiceEnabled": !edi,
				"autoSendInvoiceOnGeneration": true, "documentTypes": nil}}
	case strings.Contains(query, "createCustomer("):
		id := f.id("cus")
		f.remember("CUSTOMER", map[string]any{"id": id, "label": input["name"],
			"description": input["code"], "meta": map[string]any{"code": input["code"]}})
		data["createCustomer"] = map[string]any{"id": id, "code": input["code"],
			"name": input["name"]}
	case strings.Contains(query, "createCommodity("):
		id := f.id("com")
		f.remember("COMMODITY", map[string]any{"id": id, "label": input["name"],
			"description": input["description"], "meta": map[string]any{}})
		data["createCommodity"] = map[string]any{"id": id, "name": input["name"],
			"description": input["description"]}
	default:
		return false
	}
	reply(w, map[string]any{"data": data})
	return true
}

func (f *Fake) selectOptions(input map[string]any) []any {
	resource := fmt.Sprint(input["resource"])
	query := ""
	if raw, ok := input["query"].(string); ok {
		query = strings.ToLower(raw)
	}
	edges := []any{}
	if resource == "US_STATE" {
		if len(query) == 2 {
			abbreviation := strings.ToUpper(query)
			edges = append(edges, map[string]any{"node": map[string]any{
				"id": "us_" + abbreviation, "label": abbreviation,
				"meta": map[string]any{"abbreviation": abbreviation}}})
		}
		return edges
	}
	for _, option := range f.options[resource] {
		meta, _ := option["meta"].(map[string]any)
		text := strings.ToLower(fmt.Sprint(option["label"], " ", option["description"], " ",
			meta["code"]))
		if query == "" || strings.Contains(text, query) {
			edges = append(edges, map[string]any{"node": option})
		}
	}
	return edges
}

func (f *Fake) option(resource, id string) map[string]any {
	for _, option := range f.options[resource] {
		if option["id"] == id {
			return option
		}
	}
	return nil
}
