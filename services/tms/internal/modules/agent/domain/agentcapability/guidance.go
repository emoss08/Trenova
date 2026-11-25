package agentcapability

type Guidance struct {
	Summary   string
	UseWhen   string
	InsteadOf []Instead
	Synonyms  []string
	Examples  []string
	Chain     []string
}

func GuidanceFor(id string) (Guidance, bool) {
	guide, ok := guidance[id]
	return guide, ok
}

var guidance = map[string]Guidance{
	"equipment_list_positions": {
		Synonyms: []string{"where are my trucks", "truck locations", "gps positions", "where is the truck"},
	},
	"analysis_run": {
		UseWhen:   "You need figures worked out over many records: totals, trends, comparisons or a chart.",
		InsteadOf: []Instead{{Capability: "shipment.list", When: "a count or a grouped count of one kind of record is enough"}},
		Synonyms:  []string{"compare", "revenue", "trend", "margin", "by customer", "by month", "chart"},
	},
	"billing_approve_queue_item": {
		Chain:    []string{"billing_queue_item.list", "billing_approve_queue_item", "invoice_send"},
		Synonyms: []string{"bill the load", "bill it"},
	},
	"edi_accept_tender": {
		Chain: []string{"edi_transfer.list", "edi_accept_tender"},
	},
	"invoice_open_dispute": {
		Chain:    []string{"invoice.get", "invoice_open_dispute"},
		Synonyms: []string{"short pay", "dispute an invoice"},
	},
	"rate_send_confirmation": {
		Chain: []string{"carrier_shop", "rate_generate_confirmation", "rate_send_confirmation"},
	},
	"customer_payment.list": {
		Synonyms: []string{"money coming in", "cash in", "payments in"},
	},
	"detention_stay.list": {
		Synonyms: []string{"wait time", "dwell time", "time at the dock"},
	},
	"report_schedule.list": {
		Synonyms: []string{"scheduled reports", "reports that run on a schedule"},
	},
	"shipment.list": {
		UseWhen: "You need loads by status, customer, lane, stop city or state, pickup or delivery window, " +
			"equipment, driver, tractor, trailer, carrier, holds, lateness or billing readiness, or a count of them.",
		InsteadOf: []Instead{
			{Capability: "shipment.get", When: "you have one or a few shipments and need everything about them"},
			{Capability: "dispatch_get_board", When: "you want the dispatch board grouped by what needs a driver"},
		},
		Examples: []string{"late loads", "loads picking up today", "unassigned loads", "loads on hold", "new reefer loads for a customer"},
	},
	"shipment.get": {
		UseWhen: "You have a shipment id or PRO number and need its stops, moves, assignments, charges, holds, " +
			"comments, tracking, documents or permits, all in one call through include.",
		InsteadOf: []Instead{
			{Capability: "shipment.list", When: "you are searching for shipments by what they match"},
		},
		Examples: []string{"where is load 4471", "who is driving pro 123", "everything about this shipment"},
		Synonyms: []string{"missing documents", "missing paperwork", "documents on a shipment"},
	},
	"invoice.list": {
		UseWhen: "You need invoices by customer, status, date, amount, balance due or days past due, unpaid or open " +
			"invoices, or a count or total of them.",
		InsteadOf: []Instead{
			{Capability: "invoice.get", When: "you have the invoice id or number and need its lines and history"},
			{Capability: "receivable_open_item.list", When: "you need the open receivable balance by customer with aging"},
			{Capability: "billing_queue_item.list", When: "you need shipments waiting to be billed, not invoices"},
		},
		Synonyms: []string{"invoice list", "list of invoices", "bills", "unpaid bills", "open bills"},
		Examples: []string{"invoice list", "unpaid invoices", "bills for a customer", "past due invoices"},
	},
	"invoice.get": {
		InsteadOf: []Instead{{Capability: "invoice.list", When: "you are looking for invoices by customer, status or date"}},
	},
	"billing_queue_item.list": {
		UseWhen:   "You need shipments delivered and waiting to be billed, in review, on hold or sent back to operations.",
		InsteadOf: []Instead{{Capability: "invoice.list", When: "the shipments are already invoiced"}},
		Examples:  []string{"what is ready to bill", "unbilled loads", "billing queue"},
	},
	"receivable_open_item.list": {
		UseWhen: "You need what customers owe right now, by customer and age bucket.",
		InsteadOf: []Instead{
			{Capability: "invoice.list", When: "you need invoice records rather than open balances"},
			{Capability: "receivable_get_aging", When: "you need the aging summary rather than each open item"},
		},
		Examples: []string{"open ar", "who owes us money", "outstanding balance for a customer"},
	},
	"worker.list": {
		UseWhen: "You need drivers or other workers by status, type, home terminal, fleet, endorsement or assignment.",
		InsteadOf: []Instead{
			{Capability: "dispatch_rank_candidates", When: "you need the best available driver for a specific move"},
			{Capability: "worker_credential.list", When: "you need licenses, medical cards or other credentials and their expiry"},
		},
		Examples: []string{"driver list", "active drivers", "drivers at the Dallas terminal"},
	},
	"worker.get": {
		UseWhen: "You have a worker and need their profile, with hours of service or schedule through include.",
	},
	"dispatch_rank_candidates": {
		Summary:   "Rank the drivers who could cover a move: who is free, their hours left, distance and fit.",
		UseWhen:   "You need to know who is free or best placed to cover a move: drivers ranked by hours, distance and fit.",
		InsteadOf: []Instead{{Capability: "worker.list", When: "you need workers by their own attributes, not fit for a move"}},
		Synonyms:  []string{"who is free", "available drivers", "who can take this load", "best driver", "find a driver"},
		Examples:  []string{"who's free", "who can cover load 4471", "available drivers near Atlanta"},
		Chain:     []string{"shipment.get", "dispatch_rank_candidates", "dispatch_assign_move"},
	},
	"dispatch_plan": {
		UseWhen: "You need drivers for several shipments at once, such as the ones a list just " +
			"showed: the optimizer plans them together so no driver is used twice.",
		InsteadOf: []Instead{
			{Capability: "dispatch_rank_candidates", When: "you are weighing drivers for one move"},
		},
		Synonyms: []string{"dispatch plan", "plan drivers", "auto assign"},
		Chain:    []string{"shipment.list", "dispatch_plan", "dispatch_assign_move"},
	},
	"dispatch_assign_move": {
		UseWhen:  "You have chosen a driver, tractor and trailer for a move and want them assigned.",
		Chain:    []string{"shipment.list", "dispatch_plan", "dispatch_assign_move"},
		Synonyms: []string{"assign driver", "dispatch the load", "put a driver on"},
	},
	"dispatch_get_board": {
		Synonyms:  []string{"dispatch board", "the board", "board"},
		UseWhen:   "You want the dispatch board: today's moves grouped by what needs a driver, what is late and what is moving.",
		InsteadOf: []Instead{{Capability: "shipment.list", When: "you need shipments by specific conditions or a count"}},
	},
	"edi_send_tender": {
		UseWhen: "A move should be offered to a carrier over EDI (a 204 load tender).",
		Chain:   []string{"shipment.get", "carrier_shop", "edi_send_tender"},
	},
	"carrier_shop": {
		UseWhen:   "You need carriers to cover a load: ranked by lane history, capacity postings and rates.",
		InsteadOf: []Instead{{Capability: "carrier.list", When: "you need carriers by their own attributes"}},
		Synonyms:  []string{"find a carrier", "cover this load", "carriers for this lane"},
	},
	"tractor.list": {
		InsteadOf: []Instead{
			{Capability: "trailer.list", When: "you need trailers, reefers or vans rather than power units"},
			{Capability: "equipment_list_positions", When: "you need where units are right now"},
		},
	},
	"trailer.list": {
		InsteadOf: []Instead{{Capability: "tractor.list", When: "you need tractors or trucks rather than trailers"}},
	},
	"shipment_add_comment": {
		InsteadOf: []Instead{{Capability: "shipment_place_hold", When: "the shipment must stop until something is resolved"}},
	},
	"shipment_place_hold": {
		InsteadOf: []Instead{{Capability: "shipment_add_comment", When: "you only need to leave a note"}},
	},
	"receivable_post_customer_payment": {
		Synonyms: []string{"apply payment", "record payment", "cash application"},
		Chain:    []string{"invoice.list", "receivable_post_customer_payment"},
	},
	"customer_send_email": {
		Synonyms: []string{"email the customer", "send an update", "delay notice"},
	},
	"worker_send_message": {
		Synonyms: []string{"text the driver", "message the driver"},
	},
	"report_run": {
		InsteadOf: []Instead{{Capability: "analysis_run", When: "no saved report answers the question"}},
	},
}

func GuidanceIDs() []string {
	ids := make([]string, 0, len(guidance))
	for id := range guidance {
		ids = append(ids, id)
	}

	return ids
}
