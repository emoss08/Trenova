import { describe, expect, it } from "vitest";
import type { ProposalField } from "@/types/assistant";
import {
  changedValues,
  draftFromArguments,
  parseDraftValue,
  validateDraft,
} from "../proposal-edits";

function field(overrides: Partial<ProposalField>): ProposalField {
  return {
    name: "message",
    label: "Message",
    description: "",
    kind: "Text",
    required: false,
    options: [],
    minimum: null,
    maximum: null,
    maxLength: null,
    ...overrides,
  };
}

const FIELDS: ProposalField[] = [
  field({ name: "workerId", label: "Worker ID", kind: "Text", required: true }),
  field({ name: "message", label: "Message", kind: "Multiline", required: true, maxLength: 500 }),
  field({ name: "priority", label: "Priority", kind: "Choice", options: ["low", "high"] }),
  field({ name: "withinDays", label: "Within days", kind: "Integer", minimum: 1, maximum: 30 }),
  field({ name: "urgent", label: "Urgent", kind: "Boolean" }),
  field({ name: "codes", label: "Codes", kind: "List" }),
  field({ name: "extra", label: "Extra", kind: "JSON" }),
  field({ name: "shipmentIds", label: "Shipment IDs", kind: "RecordSubset", resource: "shipment" }),
];

/**
 * The form edits text; the tool takes values. Each field's text is read back
 * into the value its kind means, and what comes out is compared with what the
 * agent proposed so only real changes are sent.
 */
describe("proposal edits", () => {
  it("drafts every field from the proposed arguments, as text a person can edit", () => {
    const draft = draftFromArguments(FIELDS, {
      workerId: "wrk_1",
      withinDays: 3,
      urgent: true,
      codes: ["A", "B"],
      extra: { a: 1 },
    });

    expect(draft.workerId).toBe("wrk_1");
    expect(draft.message).toBe("");
    expect(draft.withinDays).toBe("3");
    expect(draft.urgent).toBe("true");
    expect(draft.codes).toBe("A, B");
    expect(draft.extra).toBe('{\n  "a": 1\n}');
  });

  it("edits a record subset as the ids it holds, and reads it back as a list", () => {
    const draft = draftFromArguments(FIELDS, { shipmentIds: ["shp_a", "shp_b", "shp_c"] });

    expect(draft.shipmentIds).toBe("shp_a, shp_b, shp_c");
    expect(parseDraftValue(FIELDS[7], "shp_a, shp_c")).toEqual({ value: ["shp_a", "shp_c"] });
  });

  // toolschema.CheckSubsets refuses an empty list, and a null or absent one,
  // whether or not the parameter is required: emptying a subset is never a
  // value the server takes, so the form refuses it rather than sending null.
  it("refuses an emptied record subset even when the parameter is optional", () => {
    expect(FIELDS[7].required).toBe(false);
    expect(parseDraftValue(FIELDS[7], "")).toEqual({
      error: "Keep at least one, or reject the proposal instead.",
    });
    expect(parseDraftValue(FIELDS[7], " , ")).toEqual({
      error: "Keep at least one, or reject the proposal instead.",
    });
    expect(
      changedValues(FIELDS, { shipmentIds: ["shp_a"] }, { shipmentIds: "" }),
    ).not.toHaveProperty("shipmentIds");
  });

  it("reads each kind back into the value the tool takes", () => {
    expect(parseDraftValue(FIELDS[3], "12")).toEqual({ value: 12 });
    expect(parseDraftValue(FIELDS[4], "false")).toEqual({ value: false });
    expect(parseDraftValue(FIELDS[5], "A, B ,C")).toEqual({ value: ["A", "B", "C"] });
    expect(parseDraftValue(FIELDS[6], '{"a":2}')).toEqual({ value: { a: 2 } });
    expect(parseDraftValue(FIELDS[0], "  wrk_2 ")).toEqual({ value: "wrk_2" });
  });

  it("leaves an empty optional field out rather than sending an empty value", () => {
    expect(parseDraftValue(FIELDS[3], "")).toEqual({ value: undefined });
    expect(parseDraftValue(FIELDS[5], "")).toEqual({ value: undefined });
  });

  it("refuses what the tool would refuse, naming the field", () => {
    const errors = validateDraft(FIELDS, {
      workerId: "",
      message: "x".repeat(501),
      priority: "urgent",
      withinDays: "40",
      urgent: "maybe",
      codes: "",
      extra: "{not json",
    });

    expect(errors.workerId).toBe("This value is required");
    expect(errors.message).toBe("At most 500 characters");
    expect(errors.priority).toBe("Choose one of the listed values");
    expect(errors.withinDays).toBe("Must be between 1 and 30");
    expect(errors.urgent).toBe("Must be yes or no");
    expect(errors.extra).toBe("Must be valid JSON");
    expect(errors.codes).toBeUndefined();
  });

  it("sends only the values that differ from the proposal", () => {
    const proposed = { workerId: "wrk_1", message: "Call in", priority: "low", withinDays: 3 };
    const changed = changedValues(FIELDS, proposed, {
      workerId: "wrk_1",
      message: "Call dispatch now",
      priority: "low",
      withinDays: "3",
      urgent: "",
      codes: "",
      extra: "",
    });

    expect(changed).toEqual({ message: "Call dispatch now" });
  });
});
