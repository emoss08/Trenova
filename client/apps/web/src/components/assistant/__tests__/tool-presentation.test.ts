import { describe, expect, it } from "vitest";
import {
  describeToolCall,
  humanizeKey,
  parseToolResult,
  readableEntries,
  readableResult,
} from "../tool-presentation";

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

  it("names a published document by its title", () => {
    expect(
      describeToolCall("publish_artifact", { title: "SEED-SHP-007 brief", body: "# Brief" }),
    ).toEqual({ title: "Publish a document", subject: "SEED-SHP-007 brief" });
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

  it("names the insight tools and the finding they were about", () => {
    expect(describeToolCall("list_insights", { severity: "Critical" })).toEqual({
      title: "Review insights",
      subject: "Critical",
    });
    expect(describeToolCall("get_insight", { insightId: "inst_1" })).toEqual({
      title: "Read insight",
      subject: "inst_1",
    });
    expect(
      describeToolCall("dismiss_insight", { insightId: "inst_1", reason: "Seasonal pattern." }),
    ).toEqual({
      title: "Dismiss insight",
      subject: "inst_1",
    });
  });

  it("names the cash application tools and the record they were about", () => {
    expect(describeToolCall("get_bank_receipt", { bankReceiptId: "brcpt_1" })).toEqual({
      title: "Look up bank receipt",
      subject: "brcpt_1",
    });
    expect(
      describeToolCall("match_bank_receipt", {
        bankReceiptId: "brcpt_1",
        customerPaymentId: "cpay_1",
      }),
    ).toEqual({ title: "Match bank receipt", subject: "brcpt_1" });
    expect(
      describeToolCall("post_customer_payment", { customerId: "cus_1", amount: "1250.00" }),
    ).toEqual({ title: "Record customer payment", subject: "cus_1" });
    expect(
      describeToolCall("resolve_bank_receipt_work_item", { workItemId: "brwi_1", note: "x" }),
    ).toEqual({ title: "Close reconciliation item", subject: "brwi_1" });
    expect(describeToolCall("list_bank_receipt_exceptions", {})).toEqual({
      title: "List unmatched receipts",
      subject: "",
    });
    expect(describeToolCall("list_customer_payments", { query: "ACH 4471" })).toEqual({
      title: "List customer payments",
      subject: "“ACH 4471”",
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

  it("names a saved report by its definition id and a dataset by its key", () => {
    expect(describeToolCall("describe_report", { definitionId: "rdef_1" })).toEqual({
      title: "Read report",
      subject: "rdef_1",
    });
    expect(
      describeToolCall("describe_report_dataset", { dataset: "shipment", query: "revenue" }),
    ).toEqual({
      title: "Read dataset",
      subject: "shipment",
    });
  });

  /**
   * A preview or a new report carries its whole question inside the definition
   * object, which no scalar argument names, so the dataset it is built on is
   * the subject rather than nothing.
   */
  it("names a tracked shipment, a failure and a detention occurrence", () => {
    expect(describeToolCall("get_shipment_tracking", { proNumber: "S12345" })).toEqual({
      title: "Track shipment",
      subject: "S12345",
    });
    expect(
      describeToolCall("resolve_service_failure", {
        serviceFailureId: "sf_1",
        notes: "Shipper closed early.",
      }),
    ).toEqual({ title: "Resolve service failure", subject: "sf_1" });
    expect(describeToolCall("send_detention_notice", { occurrenceId: "dto_1" })).toEqual({
      title: "Send detention notice",
      subject: "dto_1",
    });
  });

  it("names the dataset a definition is built on", () => {
    expect(
      describeToolCall("preview_report", {
        definition: { entity: "shipment", columns: [] },
      }),
    ).toEqual({
      title: "Preview report",
      subject: "shipment",
    });
    expect(
      describeToolCall("create_report", {
        name: "Revenue by customer",
        definition: { entity: "shipment", columns: [] },
      }),
    ).toEqual({
      title: "Create report",
      subject: "Revenue by customer",
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

  // The server appends a note after the fence when the result is already on
  // screen as an artifact. It is addressed to the model and is not part of
  // the record.
  it("reads the record and leaves out the note the model was given after it", () => {
    const content =
      'Result from search_shipments:\n<untrusted_data>\n{"count":1}\n</untrusted_data>\n\n' +
      '[Shown to the person as a table titled "Shipments", which they can open beside the ' +
      "conversation.]";

    expect(parseToolResult(content)).toEqual({
      kind: "json",
      value: { count: 1 },
      truncated: false,
    });
  });

  it("treats a proposal receipt as plain text", () => {
    const content =
      'Recorded a proposal to run "flag_for_manual_review". It is awaiting review at the Propose tier and has not run.';

    expect(parseToolResult(content)).toEqual({ kind: "text", text: content, truncated: false });
  });
});

/**
 * open_page was drawn as "Open page" with its path run into the title, which
 * read as "Open pagereport". It has a title of its own and names the page.
 */
describe("describeToolCall for navigation and the guide", () => {
  it("names the page a navigation asked for", () => {
    expect(describeToolCall("open_page", { page: "/reports/library" })).toEqual({
      title: "Open a page",
      subject: "/reports/library",
    });
  });

  it("quotes the question put to the product guide", () => {
    expect(describeToolCall("find_in_trenova", { question: "Where are reports?" })).toEqual({
      title: "Search the product guide",
      subject: "“Where are reports?”",
    });
  });

  // A record's id names nothing a person can use, so the line says what was
  // done and stops rather than ending in "inst_01M37R…".
  it("never makes a record's id the subject", () => {
    expect(
      describeToolCall("get_insight", { insightId: "inst_01M37R101VKZTB7TSKR30FJ0AT" }),
    ).toEqual({ title: "Read insight", subject: "" });
    expect(
      describeToolCall("get_customer", {
        customerId: "cus_01M37R101VKZTB7TSKR30FJ0AT",
        name: "Peak Distributing",
      }).subject,
    ).toBe("Peak Distributing");
  });

  it("does not take a page of results for a page of the app", () => {
    expect(describeToolCall("list_customers", { page: 2, query: "Peak" }).subject).toBe("“Peak”");
  });
});

describe("humanizeKey", () => {
  it("reads a key as a sentence-case label that keeps its initialisms", () => {
    expect(humanizeKey("customerId")).toBe("Customer ID");
    expect(humanizeKey("proNumber")).toBe("Pro number");
    expect(humanizeKey("dryRun")).toBe("Dry run");
  });
});

/**
 * A call's arguments and result are shown as labelled values. Nested values
 * are reduced to their shape; the literal JSON stays behind Details.
 */
describe("readableEntries", () => {
  it("labels scalars, joins scalar lists, describes filters and sizes the rest", () => {
    const { entries, hidden } = readableEntries({
      customerId: "cus_1",
      tags: ["hazmat", "team"],
      filters: [{ field: "status", operator: "eq", value: "Delivered" }],
      definition: { entity: "shipments", columns: [] },
      rows: [{ a: 1 }, { a: 2 }],
      note: "",
      archived: false,
    });

    expect(hidden).toBe(0);
    expect(entries).toEqual([
      { key: "tags", label: "Tags", value: { kind: "text", text: "hazmat, team" } },
      { key: "filters", label: "Filters", value: { kind: "text", text: "status Delivered" } },
      { key: "definition", label: "Definition", value: { kind: "fields", count: 2 } },
      { key: "rows", label: "Rows", value: { kind: "items", count: 2 } },
      {
        key: "archived",
        label: "Archived",
        value: { kind: "value", type: "boolean", value: false },
      },
    ]);
  });

  /*
   * The owner's report: the details showed a record's id, a detected date of
   * 1790187600 and a metrics list as JSON. An id under any key — or shaped like
   * one under a key that does not say so — is left out, the epoch reads as a
   * date, the metrics as measurements, and which way is worse not at all.
   */
  it("never shows an id, an epoch or a list of objects as text", () => {
    const { entries } = readableEntries({
      id: "inst_01M37R101VKZTB7TSKR30FJ0AT",
      insightId: "inst_01M37R101VKZTB7TSKR30FJ0AT",
      assignedTo: "wrk_01M37R101VKZTB7TSKR30FJ0AT",
      organizationId: "org_01M37R101VKZTB7TSKR30FJ0AT",
      severity: "Critical",
      detectedOn: 1_790_187_600,
      metrics: [
        { direction: "HigherIsWorse", label: "Workers affected", unit: "Count", value: "3" },
      ],
      direction: "HigherIsWorse",
      stale: false,
    });

    expect(entries).toEqual([
      {
        key: "severity",
        label: "Severity",
        value: { kind: "value", type: "status", value: "Critical" },
      },
      {
        key: "detectedOn",
        label: "Detected",
        value: { kind: "value", type: "date", value: 1_790_187_600 },
      },
      {
        key: "metrics",
        label: "Metrics",
        value: {
          kind: "value",
          type: "metrics",
          value: [{ label: "Workers affected", value: "3", unit: "Count" }],
        },
      },
    ]);
  });

  it("stops at the limit and says how many more there are", () => {
    const { entries, hidden } = readableEntries({ a: 1, b: 2, c: 3 }, 2);

    expect(entries.map((entry) => entry.key)).toEqual(["a", "b"]);
    expect(hidden).toBe(1);
  });
});

describe("readableResult", () => {
  it("reads a list by its count and the names of its first records", () => {
    expect(
      readableResult({
        items: [
          { name: "Peak Distributing" },
          { proNumber: "PRO-1" },
          { firstName: "Maria", lastName: "Ortiz" },
        ],
        count: 40,
        hasMore: true,
      }),
    ).toEqual({
      kind: "list",
      count: 40,
      more: true,
      labels: ["Peak Distributing", "PRO-1", "Maria Ortiz"],
    });
  });

  it("reads a bare array as a list", () => {
    expect(readableResult([{ name: "A" }])).toEqual({
      kind: "list",
      count: 1,
      more: false,
      labels: ["A"],
    });
  });

  it("reads a record as labelled values, its name first", () => {
    expect(
      readableResult({ id: "cus_01M37R101VKZTB7TSKR30FJ0AT", status: "Active", name: "Peak" }),
    ).toEqual({
      kind: "record",
      entries: [
        { key: "name", label: "Name", value: { kind: "text", text: "Peak" } },
        {
          key: "status",
          label: "Status",
          value: { kind: "value", type: "status", value: "Active" },
        },
      ],
      hidden: 0,
    });
  });

  it("never names a record in a list by its id", () => {
    expect(
      readableResult({
        items: [{ id: "inst_01M37R101VKZTB7TSKR30FJ0AT", headline: "3 cards expire" }],
        count: 1,
      }),
    ).toEqual({ kind: "list", count: 1, more: false, labels: ["3 cards expire"] });
  });
});
