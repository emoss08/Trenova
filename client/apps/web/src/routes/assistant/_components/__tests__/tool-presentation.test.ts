import { describe, expect, it } from "vitest";
import { describeToolCall, parseToolResult } from "../tool-presentation";

/**
 * A tool call is shown to a person, not to a model, so the wording comes from
 * what the tool does and what it was asked about — "Look up shipment · S12345"
 * — rather than from the function name and a JSON blob.
 */
describe("describeToolCall", () => {
  it("names the action and the record it was about", () => {
    expect(describeToolCall("get_shipment", { proNumber: "S12345" })).toEqual({
      title: "Look up shipment",
      subject: "S12345",
    });
  });

  it("uses the search text for a search", () => {
    expect(describeToolCall("search_workers", { query: "Ortiz", limit: 10 })).toEqual({
      title: "Search drivers",
      subject: "“Ortiz”",
    });
  });

  it("falls back to a humanized name and the first scalar argument", () => {
    expect(describeToolCall("recompute_rate", { shipmentId: "shp_1", dryRun: true })).toEqual({
      title: "Recompute rate",
      subject: "shp_1",
    });
  });

  it("leaves the subject empty when there is nothing to name", () => {
    expect(describeToolCall("list_terminals", {})).toEqual({
      title: "List terminals",
      subject: "",
    });
  });
});

/**
 * The server fences a tool result as untrusted data and may truncate it. The
 * reader wants the record, not the fence, and wants to know when they are not
 * seeing all of it.
 */
describe("parseToolResult", () => {
  it("unwraps the fence and parses the JSON inside", () => {
    const content =
      'Result from get_shipment:\n<untrusted_data>\n{"proNumber":"S1","status":"InTransit"}\n</untrusted_data>';

    expect(parseToolResult(content)).toEqual({
      kind: "json",
      value: { proNumber: "S1", status: "InTransit" },
      truncated: false,
    });
  });

  it("reports truncation and returns the text when the JSON was cut short", () => {
    const content =
      'Result from search_shipments:\n<untrusted_data>\n[{"proNumber":"S1"},{"pro\n…(truncated)\n</untrusted_data>';

    expect(parseToolResult(content)).toEqual({
      kind: "text",
      text: '[{"proNumber":"S1"},{"pro',
      truncated: true,
    });
  });

  it("extracts the reason from a failed call", () => {
    expect(parseToolResult('Tool "get_shipment" failed: shipment not found')).toEqual({
      kind: "error",
      message: "shipment not found",
    });
  });

  it("treats a proposal receipt as plain text", () => {
    const content =
      'Recorded a proposal to run "flag_for_manual_review". It is awaiting review at the Propose tier and has not run.';

    expect(parseToolResult(content)).toEqual({ kind: "text", text: content, truncated: false });
  });
});
