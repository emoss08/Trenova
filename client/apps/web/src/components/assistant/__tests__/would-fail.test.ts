import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import type { ProposalField } from "@/types/assistant";
import {
  askAgentMessage,
  editableParam,
  previewWarningText,
  wouldFailReasons,
} from "../proposal-preview/preview-warnings";
import { reason, warning } from "./preview-fixtures";

const t = translate;

const SHIPMENT_FIELD: ProposalField = {
  name: "shipment",
  label: "Shipment",
  description: "",
  kind: "JSON",
  required: true,
  options: [],
  minimum: null,
  maximum: null,
  maxLength: null,
};

describe("wouldFailReasons", () => {
  it("lists each reason the server structured, with the field it names", () => {
    const reasons = wouldFailReasons(
      warning({
        code: "would_fail",
        reasons: [
          reason(),
          reason({ field: "", label: "", param: "", message: "The customer is on credit hold" }),
        ],
      }),
    );

    expect(reasons).toEqual([
      {
        field: "bol",
        label: "BOL",
        message: "BOL is already in use by shipment SEED-DET-009",
        param: "shipment.bol",
      },
      { field: "", label: "", message: "The customer is on credit hold", param: "" },
    ]);
  });

  // A server that sent no structured reasons still said why, in its message.
  // The person reads that, never the generic sentence alone.
  it("falls back to the server's words without the sentence that leads them", () => {
    const reasons = wouldFailReasons(
      warning({
        code: "would_fail",
        reasons: [],
        message:
          "This would be refused as it stands: validation failed:\n- BOL is already in use by shipment(s) with Pro Number(s): SEED-DET-009\n- Location is required",
      }),
    );

    expect(reasons.map((entry) => entry.message)).toEqual([
      "BOL is already in use by shipment(s) with Pro Number(s): SEED-DET-009",
      "Location is required",
    ]);
    expect(reasons[0]?.param).toBe("");
  });

  it("falls back to the first argument, then to a single reason of the whole message", () => {
    expect(
      wouldFailReasons(warning({ code: "would_fail", args: ["Rate not found"], message: "" })),
    ).toEqual([{ field: "", label: "", message: "Rate not found", param: "" }]);
    expect(
      wouldFailReasons(
        warning({ code: "would_fail", message: "This change would fail as proposed: no hold" }),
      ),
    ).toEqual([{ field: "", label: "", message: "no hold", param: "" }]);
    expect(wouldFailReasons(warning({ code: "would_fail", message: "" }))).toEqual([]);
  });

  it("is empty for a warning of another code", () => {
    expect(wouldFailReasons(warning({ code: "withheld", message: "Hidden." }))).toEqual([]);
  });
});

describe("previewWarningText for would_fail", () => {
  it("reads the reasons in one line, and the bare sentence only when the server gave none", () => {
    expect(previewWarningText(warning({ code: "would_fail", reasons: [reason()] }), t)).toBe(
      "This would not go through as it stands: BOL: BOL is already in use by shipment SEED-DET-009",
    );
    expect(previewWarningText(warning({ code: "would_fail", message: "" }), t)).toBe(
      "This would not go through as it stands.",
    );
  });
});

describe("editableParam", () => {
  it("is the parameter's top-level field, when a person may edit it", () => {
    expect(editableParam("shipment.bol", [SHIPMENT_FIELD])).toBe(true);
    expect(editableParam("shipment.moves[0].stops[1].locationId", [SHIPMENT_FIELD])).toBe(true);
    expect(editableParam("shipment", [SHIPMENT_FIELD])).toBe(true);
    expect(editableParam("shipment.bol", [{ ...SHIPMENT_FIELD, readOnly: true }])).toBe(false);
    expect(editableParam("sourceDocumentId", [SHIPMENT_FIELD])).toBe(false);
    expect(editableParam("", [SHIPMENT_FIELD])).toBe(false);
  });
});

describe("askAgentMessage", () => {
  it("names each reason and asks for a corrected proposal", () => {
    const text = askAgentMessage(
      "create_shipment",
      [reason(), { field: "", label: "", message: "The customer is on credit hold", param: "" }],
      t,
    );

    expect(text).toBe(
      "The proposal to create shipment would not go through as it stands: BOL: BOL is already in use by shipment SEED-DET-009; The customer is on credit hold. Fix it and propose it again; ask me for anything you need.",
    );
  });
});
