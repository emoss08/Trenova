import type { AssistantProposal } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { presentProposal, shortRef } from "../proposal-presenters";

function proposal(overrides: Partial<AssistantProposal>): AssistantProposal {
  return {
    id: "aprop_1",
    runId: "arun_1",
    threadId: "athr_1",
    sourceMessageId: "amsg_1",
    toolName: "request_missing_docs",
    arguments: {},
    rationale: "",
    confidence: 0.8,
    autonomyTier: "Propose",
    status: "Pending",
    executedAt: null,
    executionError: "",
    createdAt: 1,
    updatedAt: 1,
    ...overrides,
  } as AssistantProposal;
}

/**
 * Tool arguments follow the server's JSON schemas
 * (services/agenttoolservice/*.go). The card exists to answer "should this
 * run", so a presenter's job is a sentence that answers it; everything the
 * sentence does not need goes to `details`, which is still complete.
 */
describe("presentProposal", () => {
  it("explains a document request by recipient and documents", () => {
    const view = presentProposal(
      proposal({
        toolName: "request_missing_docs",
        arguments: {
          to: ["ap@acme.example", "ops@acme.example"],
          subject: "Missing POD for S12345",
          customerName: "Acme",
          shipmentProNumber: "S12345",
          requestedDocuments: ["Proof of delivery", "Signed BOL"],
          body: "Hello, please send the documents.",
        },
      }),
    );

    expect(view.title).toBe("Email the customer");
    expect(view.summary).toBe("Ask Acme for 2 documents on shipment S12345.");
    expect(view.highlights).toEqual([
      { label: "To", value: "ap@acme.example, ops@acme.example" },
      { label: "Asking for", value: "Proof of delivery, Signed BOL" },
      { label: "Subject", value: "Missing POD for S12345" },
    ]);
    expect(view.reversible).toBe(false);
  });

  it("does not repeat an argument the presenter already spoke for", () => {
    const view = presentProposal(
      proposal({
        toolName: "request_missing_docs",
        arguments: {
          to: ["ap@acme.example"],
          body: "Hello, please send the documents.",
          requestedDocuments: ["Proof of delivery"],
        },
      }),
    );

    const labels = view.highlights.map((entry) => entry.label);
    expect(labels).toEqual(["To", "Asking for"]);
    expect(labels).not.toContain("body");
  });

  it("shows a charge correction as the charges that would replace the current ones", () => {
    const view = presentProposal(
      proposal({
        toolName: "correct_charge_code",
        arguments: {
          billingQueueItemId: "bqi_1",
          additionalCharges: [
            { accessorialChargeId: "acc_1", amount: 125, description: "Detention" },
            { accessorialChargeId: "acc_2", amount: 40 },
          ],
        },
      }),
    );

    expect(view.title).toBe("Correct charge codes");
    expect(view.summary).toBe(
      "Replace the additional charges on this billing item with 2 charges.",
    );
    expect(view.highlights).toEqual([
      { label: "Detention", value: "125" },
      { label: "acc_2", value: "40" },
    ]);
    expect(view.reversible).toBe(true);
  });

  it("describes a move assignment without repeating the move's own id", () => {
    const view = presentProposal(
      proposal({
        toolName: "assign_move",
        arguments: {
          shipmentMoveId: "smv_1",
          primaryWorkerId: "wrk_1",
          secondaryWorkerId: "",
          tractorId: "trc_1",
          trailerId: "trl_1",
        },
      }),
    );

    expect(view.title).toBe("Assign a driver");
    expect(view.summary).toBe("Put driver wrk_1 on this move with tractor trc_1.");
    expect(view.highlights).toEqual([
      { label: "Driver", value: "wrk_1" },
      { label: "Tractor", value: "trc_1" },
      { label: "Trailer", value: "trl_1" },
    ]);
  });

  /**
   * The server sends PULIDs, not names. A 30-character key in a sentence is
   * something an approver has to match by eye, and the earlier card put one
   * there verbatim.
   */
  it("shortens an opaque id in the sentence", () => {
    const view = presentProposal(
      proposal({
        toolName: "assign_move",
        arguments: {
          shipmentMoveId: "smv_01M2PRNXAMQNKK9HK9V5B817QE",
          primaryWorkerId: "wrk_01M2PRNXAMQNKK9HK9V5B817QE",
        },
      }),
    );

    expect(view.summary).toBe("Put driver …B817QE on this move.");
    expect(view.summary).not.toContain("wrk_01M2PRNXAMQNKK9HK9V5B817QE");
  });

  /**
   * "Raise a medium missing bol for a person to handle" was the old sentence:
   * the severity and the category were spliced into the prose in whatever case
   * the server sent, and the same two values were then repeated as rows below.
   */
  it("reads a flag as a sentence and carries severity separately", () => {
    const view = presentProposal(
      proposal({
        toolName: "flag_for_manual_review",
        arguments: {
          subjectId: "shp_01M2PRNXAMQNKK9HK9V5B817QE",
          category: "MissingBOL",
          severity: "Medium",
          blastRadius: 1,
          attemptSummary: "Signed bill of lading is missing.",
        },
      }),
    );

    expect(view.summary).toBe("Record a missing BOL case for a person to resolve.");
    expect(view.severity).toEqual({ label: "Medium", tone: "warning" });
    expect(view.highlights).toEqual([
      { label: "What the agent found", value: "Signed bill of lading is missing." },
    ]);
  });

  /**
   * With no disclosure left on the card, anything a presenter names is the only
   * thing shown. The run id, the subject's PULID and the raw evidence blob are
   * bookkeeping for the audit trail, not inputs to a yes or no, and printing
   * them was the whole reason the old Details panel read as a dump.
   */
  it("keeps a flag's bookkeeping off the card", () => {
    const view = presentProposal(
      proposal({
        toolName: "flag_for_manual_review",
        arguments: {
          runId: "ar_01M2PRQ1C0QN91R9TNRXSSSFZ5",
          subjectId: "shp_01M2PRNXAMQNKK9HK9V5B817QE",
          category: "MissingBOL",
          severity: "Medium",
          blastRadius: 1,
          attemptSummary: "Signed bill of lading is missing.",
          evidence: [{ id: "shp_01M2PRNXAMQNKK9HK9V5B817QE", type: "Shipment" }],
        },
      }),
    );

    const labels = view.highlights.map((entry) => entry.label);
    for (const noise of ["run Id", "subject Id", "evidence", "category", "severity"]) {
      expect(labels, `${noise} should not be on the card`).not.toContain(noise);
    }
  });

  it("omits a blast radius of one rather than stating the obvious", () => {
    const wide = presentProposal(
      proposal({
        toolName: "flag_for_manual_review",
        arguments: { category: "RateMismatch", severity: "High", blastRadius: 7 },
      }),
    );

    expect(wide.highlights).toContainEqual({ label: "Records affected", value: "7" });
    expect(wide.severity).toEqual({ label: "High", tone: "danger" });
  });

  it("puts every argument on the face of the card for a tool it has no words for", () => {
    const view = presentProposal(
      proposal({
        toolName: "rotate_tires",
        arguments: { tractorId: "trc_9", mileage: 120000, notes: null },
      }),
    );

    expect(view.title).toBe("Rotate tires");
    expect(view.summary).toBe("Run rotate tires with the values below.");
    expect(view.highlights).toEqual([
      { label: "mileage", value: "120000" },
      { label: "notes", value: "—" },
      { label: "tractor Id", value: "trc_9" },
    ]);
    expect(view.reversible).toBe(false);
  });

  /**
   * The safety net the disclosure used to provide. A tool that grows an
   * argument server-side must still put it in front of the approver, or the
   * card would quietly understate what approving it does.
   */
  it("never loses an argument the presenter did not expect", () => {
    const view = presentProposal(
      proposal({
        toolName: "assign_move",
        arguments: { shipmentMoveId: "smv_1", primaryWorkerId: "wrk_1", surprise: "yes" },
      }),
    );

    expect(view.highlights).toContainEqual({ label: "surprise", value: "yes" });
  });
});

/**
 * A report the model wrote is a definition object nobody should have to read on
 * a card. The presenter says what it would save — the name, the dataset and how
 * many columns — and keeps the definition itself off the face.
 */
describe("presentProposal for report tools", () => {
  const definition = {
    entity: "shipment",
    columns: [
      { id: "c1", ref: { path: ["customer"], field: "name" }, kind: "dimension" },
      { id: "c2", ref: { field: "totalChargeAmount" }, kind: "measure", agg: "sum" },
    ],
    filters: {
      op: "and",
      filters: [{ ref: { field: "status" }, operator: "eq", value: "Completed" }],
    },
    parameters: [{ name: "windowDays", type: "int", required: true }],
  };

  it("describes a new report by name, dataset and shape", () => {
    const view = presentProposal(
      proposal({
        toolName: "create_report",
        arguments: {
          name: "Revenue by customer",
          description: "Completed revenue per customer.",
          category: "Accounting",
          visibility: "shared",
          definition,
        },
      }),
    );

    expect(view.title).toBe("Create a report");
    expect(view.summary).toBe(
      "Save “Revenue by customer” as a new shared report on the shipment dataset.",
    );
    expect(view.highlights).toEqual([
      { label: "Columns", value: "customer.name, sum of totalChargeAmount" },
      { label: "Filters", value: "1 filter" },
      { label: "Parameters", value: "windowDays" },
      { label: "Category", value: "Accounting" },
      { label: "Description", value: "Completed revenue per customer." },
    ]);
    expect(view.reversible).toBe(true);
  });

  it("describes a change to a saved report by what it touches", () => {
    const view = presentProposal(
      proposal({
        toolName: "update_report",
        arguments: {
          definitionId: "rdef_01M3034Q2N7JD99RA1D8DGH1ZF",
          name: "Revenue by customer, completed",
          definition,
        },
      }),
    );

    expect(view.title).toBe("Change a report");
    expect(view.summary).toBe("Change this report's name and definition.");
    expect(view.highlights).toEqual([
      { label: "Name", value: "Revenue by customer, completed" },
      { label: "Columns", value: "customer.name, sum of totalChargeAmount" },
      { label: "Filters", value: "1 filter" },
      { label: "Parameters", value: "windowDays" },
    ]);
    expect(view.reversible).toBe(true);
  });

  it("describes a metadata-only change without inventing a definition", () => {
    const view = presentProposal(
      proposal({
        toolName: "update_report",
        arguments: {
          definitionId: "rdef_01M3034Q2N7JD99RA1D8DGH1ZF",
          visibility: "shared",
          status: "archived",
        },
      }),
    );

    expect(view.summary).toBe("Change this report's visibility and status.");
    expect(view.highlights).toEqual([
      { label: "Visibility", value: "Shared" },
      { label: "Status", value: "Archived" },
    ]);
  });

  it("describes a fork as a copy of the built-in report", () => {
    const view = presentProposal(
      proposal({
        toolName: "fork_report",
        arguments: { reportKey: "ar_aging_by_customer", name: "AR aging, 60-day buckets" },
      }),
    );

    expect(view.title).toBe("Copy a report");
    expect(view.summary).toBe(
      "Make a copy of the built-in ar_aging_by_customer report named “AR aging, 60-day buckets”.",
    );
    expect(view.highlights).toEqual([]);
    expect(view.reversible).toBe(true);
  });
});

/**
 * The monitoring writes reach an approver as what would happen to whom: a
 * message to a driver, an email to a customer, a charge waived. The ids the
 * tool needs stay out of the sentence.
 */
describe("presentProposal for monitoring tools", () => {
  it("describes a driver message with its urgency", () => {
    const view = presentProposal(
      proposal({
        toolName: "notify_driver",
        arguments: {
          workerId: "wrk_01M3034Q2N7JD99RA1D8DGH1ZF",
          title: "Delivery moved to 3 PM",
          message: "Houston DC moved your appointment to 3 PM. No need to rush.",
          priority: "high",
        },
      }),
    );

    expect(view.title).toBe("Message the driver");
    expect(view.summary).toBe("Send driver …DGH1ZF “Delivery moved to 3 PM” in Dash.");
    expect(view.severity).toEqual({ label: "High", tone: "warning" });
    expect(view.highlights).toEqual([
      { label: "Message", value: "Houston DC moved your appointment to 3 PM. No need to rush." },
    ]);
    expect(view.reversible).toBe(false);
  });

  it("describes a customer email by its subject and body", () => {
    const view = presentProposal(
      proposal({
        toolName: "email_customer",
        arguments: {
          shipmentId: "shp_01M3034Q2N7JD99RA1D8DGH1ZF",
          profileId: "emp_01M3034Q2N7JD99RA1D8DGH1ZF",
          subject: "S12345 running about an hour late",
          body: "The truck is held in traffic near Huntsville; we now expect 3:30 PM.",
        },
      }),
    );

    expect(view.title).toBe("Email the customer");
    expect(view.summary).toBe(
      "Send this shipment's customer “S12345 running about an hour late” from the organization's letterhead.",
    );
    expect(view.highlights).toEqual([
      {
        label: "Message",
        value: "The truck is held in traffic near Huntsville; we now expect 3:30 PM.",
      },
    ]);
  });

  it("describes a waiver by its coded reason", () => {
    const view = presentProposal(
      proposal({
        toolName: "waive_detention",
        arguments: {
          occurrenceId: "dto_01M3034Q2N7JD99RA1D8DGH1ZF",
          reason: "CarrierFault",
          note: "Our truck arrived two hours late.",
        },
      }),
    );

    expect(view.title).toBe("Waive detention");
    expect(view.summary).toBe("Waive this detention charge as carrier fault.");
    expect(view.highlights).toEqual([
      { label: "Note", value: "Our truck arrived two hours late." },
    ]);
  });

  it("describes a failure resolution and a failure check", () => {
    const resolve = presentProposal(
      proposal({
        toolName: "resolve_service_failure",
        arguments: {
          serviceFailureId: "sf_01M3034Q2N7JD99RA1D8DGH1ZF",
          reasonCodeId: "sfrc_01M3034Q2N7JD99RA1D8DGH1ZF",
          notes: "Shipper closed early; driver waited until opening.",
        },
      }),
    );
    expect(resolve.title).toBe("Resolve a service failure");
    expect(resolve.highlights).toEqual([
      { label: "Reason code", value: "…DGH1ZF" },
      { label: "Note", value: "Shipper closed early; driver waited until opening." },
    ]);

    const check = presentProposal(
      proposal({
        toolName: "evaluate_service_failures",
        arguments: { shipmentId: "shp_01M3034Q2N7JD99RA1D8DGH1ZF", force: true },
      }),
    );
    expect(check.summary).toBe(
      "Run the late-stop check on this shipment, re-checking stops already evaluated.",
    );
    expect(check.highlights).toEqual([]);
  });
});

describe("shortRef", () => {
  it("shortens a PULID and leaves a human reference alone", () => {
    expect(shortRef("shp_01M2PRNXAMQNKK9HK9V5B817QE")).toBe("…B817QE");
    expect(shortRef("SEED-SHP-001")).toBe("SEED-SHP-001");
    expect(shortRef("")).toBe("");
  });
});
