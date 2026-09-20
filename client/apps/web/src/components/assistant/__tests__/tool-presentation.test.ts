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
    expect(describeToolCall("search_worker", { query: "Ortiz", limit: 10 })).toEqual({
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

  /**
   * A filtered list carries no scalar argument at all — its whole question is
   * inside the filters array — so without this it reads as a bare "List
   * shipments" and the reader cannot tell what was asked.
   */
  it("summarizes the filters a list call applied", () => {
    expect(
      describeToolCall("list_shipments", {
        filters: [{ field: "status", operator: "eq", value: "Delivered" }],
        limit: 25,
      }),
    ).toEqual({
      title: "List shipments",
      subject: "status Delivered",
    });
  });

  it("reads a relative window as a window", () => {
    expect(
      describeToolCall("list_trailers", {
        filters: [{ field: "registrationExpiry", operator: "nextndays", days: 30 }],
      }),
    ).toEqual({
      title: "List trailers",
      subject: "registrationExpiry next 30d",
    });
  });

  it("keeps a long filter list to a readable length", () => {
    expect(
      describeToolCall("list_workers", {
        filters: [
          { field: "status", operator: "eq", value: "Active" },
          { field: "type", operator: "eq", value: "Employee" },
          { field: "city", operator: "contains", value: "Dallas" },
        ],
      }).subject,
    ).toBe("status Active, type Employee +1");
  });

  it("names the report a run was started for", () => {
    expect(
      describeToolCall("run_report", {
        reportKey: "ar_aging_by_customer",
        parameters: { asOf: "2026-03-01" },
      }),
    ).toEqual({
      title: "Start report",
      subject: "ar_aging_by_customer",
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

  // The notice is the one the server actually writes (agentruntime/fence.go),
  // not the "…(truncated)" marker it replaced. A fixture copied from the old
  // implementation kept passing while every real truncated result fell through
  // as ordinary text with nothing saying it was short.
  it("reports truncation and returns the text when the JSON was cut short", () => {
    const content =
      'Result from search_shipments:\n<untrusted_data>\n[{"proNumber":"S1"},{"pro\n\n' +
      "[This result was cut off here: it was too long to return in full, so the text " +
      "above ends mid-record and the records after it are missing entirely. Do not " +
      "infer, complete, or count anything from the cut-off portion. Narrow your " +
      "filters and call the tool again, and tell the person you are working from a " +
      "partial result until you do.]\n</untrusted_data>";

    expect(parseToolResult(content)).toEqual({
      kind: "text",
      text: '[{"proNumber":"S1"},{"pro',
      truncated: true,
    });
  });

  // Threads saved before the notice was reworded still have to read correctly.
  it("still recognises the marker the server used to write", () => {
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
