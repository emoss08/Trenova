import { pageDraftSchema, type PageDraftEdit } from "@/types/page-draft";
import { describe, expect, it, vi } from "vitest";
import {
  applyImportDraftEdit,
  createFailureSummary,
  importDraftFromState,
  type ImportDraftHandlers,
} from "../import-draft";
import type { ReconciliationField, ReconciliationState, ReconciliationStop } from "../types";

/*
 * The draft is checked on the server by pagedraft.Draft.Validate: at most 80
 * fields with keys of letters, digits, "_", "-" or "."; labels of 120 runes,
 * values of 1000; at most 20 stops of 300 runes a field; every confidence in
 * [0, 1]. A draft past any bound refuses the whole turn, so the page bounds it.
 */
function field(key: string, overrides: Partial<ReconciliationField> = {}): ReconciliationField {
  return { key, label: key, value: "", confidence: 0.9, status: "accepted", ...overrides };
}

function stop(overrides: Partial<ReconciliationStop> = {}): ReconciliationStop {
  return {
    sequence: 1,
    role: "pickup",
    status: "accepted",
    confidence: 0.8,
    name: field("name", { value: "Acme DC" }),
    addressLine1: field("addressLine1", { value: "1 Main St" }),
    city: field("city", { value: "Dallas" }),
    state: field("state", { value: "TX" }),
    postalCode: field("postalCode", { value: "75201" }),
    date: field("date", { value: "1767225600" }),
    timeWindow: field("timeWindow", { value: "" }),
    locationId: "",
    appointmentRequired: false,
    ...overrides,
  };
}

const required = {
  customerId: "cus_01",
  serviceTypeId: "",
  shipmentTypeId: "",
  formulaTemplateId: "",
};

function state(overrides: Partial<ReconciliationState> = {}): ReconciliationState {
  return { fields: {}, stops: [], overallConfidence: 0.8, ...overrides };
}

describe("importDraftFromState", () => {
  it("hands the assistant the fields, the required records and the stops as text", () => {
    const draft = importDraftFromState(
      state({
        fields: {
          loadNumber: field("loadNumber", { label: "Load Number", value: "L-100" }),
          weight: field("weight", { label: "Weight", value: 42000, status: "needs-review" }),
          rate: field("rate", { label: "Rate", value: null, status: "missing", confidence: 0 }),
        },
        stops: [stop({ locationId: "loc_1" })],
      }),
      required,
    );

    expect(pageDraftSchema.safeParse(draft).success).toBe(true);
    expect(draft.surface).toBe("shipment_import");
    expect(draft.shipmentImport?.fields).toEqual([
      {
        key: "loadNumber",
        label: "Load Number",
        value: "L-100",
        confidence: 0.9,
        status: "accepted",
      },
      { key: "weight", label: "Weight", value: "42000", confidence: 0.9, status: "needs-review" },
      { key: "rate", label: "Rate", value: "", confidence: 0, status: "missing" },
    ]);
    expect(draft.shipmentImport?.required).toEqual(required);
    expect(draft.shipmentImport?.stops).toEqual([
      {
        role: "pickup",
        name: "Acme DC",
        addressLine1: "1 Main St",
        city: "Dallas",
        state: "TX",
        postalCode: "75201",
        date: "1767225600",
        timeWindow: "",
        locationId: "loc_1",
        confidence: 0.8,
      },
    ]);
  });

  it("keeps within every bound the server enforces", () => {
    const fields: Record<string, ReconciliationField> = {};
    for (let index = 0; index < 90; index += 1) {
      fields[`field_${index}`] = field(`field_${index}`);
    }
    fields["bad key!"] = field("bad key!");
    fields.long = field("long", {
      label: "é".repeat(200),
      value: "😀".repeat(1200),
      confidence: 1.7,
    });
    const stops = Array.from({ length: 25 }, () =>
      stop({ name: field("name", { value: "n".repeat(400) }), confidence: -1 }),
    );

    const draft = importDraftFromState(state({ fields, stops }), required);
    const imported = draft.shipmentImport!;

    expect(imported.fields).toHaveLength(80);
    expect(imported.fields.some((item) => item.key === "bad key!")).toBe(false);
    expect(imported.stops).toHaveLength(20);
    expect(Array.from(imported.stops[0].name)).toHaveLength(300);
    expect(imported.stops[0].confidence).toBe(0);

    const long = importDraftFromState(state({ fields: { long: fields.long } }), required)
      .shipmentImport!.fields[0];
    expect(Array.from(long.label)).toHaveLength(120);
    expect(Array.from(long.value)).toHaveLength(1000);
    expect(long.confidence).toBe(1);
  });

  it("writes a structured value as JSON rather than [object Object]", () => {
    const draft = importDraftFromState(
      state({ fields: { billTo: field("billTo", { value: { name: "Acme" } }) } }),
      required,
    );

    expect(draft.shipmentImport?.fields[0].value).toBe('{"name":"Acme"}');
  });
});

function handlers(): ImportDraftHandlers {
  return {
    acceptField: vi.fn(),
    acceptAllConfident: vi.fn(),
    editField: vi.fn(),
    setRequiredField: vi.fn(),
    setStopLocation: vi.fn(),
    setStopSchedule: vi.fn(),
  };
}

function importEdit(overrides: Partial<PageDraftEdit>): PageDraftEdit {
  return {
    surface: "shipment_import",
    action: "accept_field",
    fieldKey: "",
    value: "",
    label: "",
    stopIndex: null,
    windowStart: 0,
    windowEnd: 0,
    formula: null,
    ...overrides,
  };
}

describe("applyImportDraftEdit", () => {
  it("applies each change to the page the way a person's click would", () => {
    const page = handlers();

    expect(applyImportDraftEdit(importEdit({ fieldKey: "loadNumber" }), page, 2)).toBe(true);
    expect(applyImportDraftEdit(importEdit({ action: "accept_all_confident" }), page, 2)).toBe(
      true,
    );
    expect(
      applyImportDraftEdit(
        importEdit({ action: "set_field_value", fieldKey: "weight", value: "42000" }),
        page,
        2,
      ),
    ).toBe(true);
    expect(
      applyImportDraftEdit(
        importEdit({ action: "set_required_field", fieldKey: "serviceTypeId", value: "st_1" }),
        page,
        2,
      ),
    ).toBe(true);
    expect(
      applyImportDraftEdit(
        importEdit({ action: "set_stop_location", stopIndex: 0, value: "loc_1" }),
        page,
        2,
      ),
    ).toBe(true);
    expect(
      applyImportDraftEdit(
        importEdit({ action: "set_stop_schedule", stopIndex: 1, windowStart: 1767225600 }),
        page,
        2,
      ),
    ).toBe(true);
    expect(
      applyImportDraftEdit(
        importEdit({
          action: "set_stop_schedule",
          stopIndex: 1,
          windowStart: 1767225600,
          windowEnd: 1767232800,
        }),
        page,
        2,
      ),
    ).toBe(true);

    expect(page.acceptField).toHaveBeenCalledWith("loadNumber");
    expect(page.acceptAllConfident).toHaveBeenCalledTimes(1);
    expect(page.editField).toHaveBeenCalledWith("weight", "42000");
    expect(page.setRequiredField).toHaveBeenCalledWith("serviceTypeId", "st_1");
    expect(page.setStopLocation).toHaveBeenCalledWith(0, "loc_1");
    expect(page.setStopSchedule).toHaveBeenNthCalledWith(1, 1, "1767225600", undefined);
    expect(page.setStopSchedule).toHaveBeenNthCalledWith(2, 1, "1767225600", "1767232800");
  });

  it("refuses a change that no longer fits the page", () => {
    const page = handlers();

    expect(applyImportDraftEdit(importEdit({ fieldKey: "" }), page, 1)).toBe(false);
    expect(
      applyImportDraftEdit(
        importEdit({ action: "set_required_field", fieldKey: "tractorTypeId", value: "tt_1" }),
        page,
        1,
      ),
    ).toBe(false);
    expect(
      applyImportDraftEdit(
        importEdit({ action: "set_stop_location", stopIndex: 3, value: "loc_1" }),
        page,
        1,
      ),
    ).toBe(false);
    expect(
      applyImportDraftEdit(importEdit({ action: "set_stop_location", value: "loc_1" }), page, 1),
    ).toBe(false);
    expect(
      applyImportDraftEdit(
        importEdit({ action: "set_stop_schedule", stopIndex: 0, windowStart: 0 }),
        page,
        1,
      ),
    ).toBe(false);
    expect(
      applyImportDraftEdit(importEdit({ surface: "formula", action: "propose_formula" }), page, 1),
    ).toBe(false);

    for (const handler of Object.values(page)) {
      expect(handler).not.toHaveBeenCalled();
    }
  });
});

describe("createFailureSummary", () => {
  it("lists each field the shipment was refused over, one per line", () => {
    const raw = JSON.stringify([
      { path: ["stops", 0, "locationId"], message: "Location is required" },
      { path: ["customerId"], message: "Customer is required" },
    ]);

    expect(createFailureSummary(raw)).toBe(
      "- stops.0.locationId: Location is required\n- customerId: Customer is required",
    );
  });

  it("keeps a plain message as it is", () => {
    expect(createFailureSummary("Rating failed: no rate for this lane")).toBe(
      "Rating failed: no rate for this lane",
    );
  });

  it("keeps a JSON value that is not a list of issues as it is", () => {
    expect(createFailureSummary('{"detail":"nope"}')).toBe('{"detail":"nope"}');
    expect(createFailureSummary("[]")).toBe("[]");
  });

  it("names an issue with no path as the shipment's own", () => {
    expect(createFailureSummary(JSON.stringify([{ message: "Duplicate BOL" }]))).toBe(
      "- shipment: Duplicate BOL",
    );
  });
});
