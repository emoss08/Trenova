import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { groupExtraction, reviewTally } from "../extraction-groups";
import { extractionFrom } from "../extraction-payload";
import {
  documentMark,
  headFacts,
  headFigure,
  headStanding,
  headSubtitle,
  laneOf,
  readMoney,
} from "../extraction-summary";
import type { ShipmentStage } from "../shipment-stage";
import { currentPayload, legacyPayload } from "./extraction-fixtures";

const t = translate;

type Field = Record<string, unknown>;

function fieldsOf(payload: Record<string, unknown>): Field[] {
  return payload.fields as Field[];
}

function reading(payload: Record<string, unknown>) {
  const extraction = extractionFrom(payload);
  const groups = groupExtraction(extraction);
  return { extraction, groups, tally: reviewTally(groups) };
}

function withFields(change: (fields: Field[]) => Field[]) {
  return reading(currentPayload({ fields: change(fieldsOf(currentPayload())) }));
}

describe("documentMark", () => {
  it("abbreviates the paper the way a dispatcher does", () => {
    expect(documentMark("RateConfirmation", t)).toBe("RC");
    expect(documentMark("BillOfLading", t)).toBe("BOL");
    expect(documentMark("ProofOfDelivery", t)).toBe("POD");
  });

  it("marks a kind it does not know, and never reads one off the object's prototype", () => {
    expect(documentMark("Other", t)).toBe("DOC");
    expect(documentMark("", t)).toBe("DOC");
    expect(documentMark("constructor", t)).toBe("DOC");
    expect(documentMark("toString", t)).toBe("DOC");
  });
});

describe("laneOf", () => {
  it("runs from the first stop's place to the last's and counts every stop", () => {
    expect(laneOf(reading(currentPayload()).groups)).toEqual({
      from: "Memphis, TN",
      to: "Fort Worth, TX",
      stops: 3,
    });
  });

  it("skips a stop that names no place but still counts it", () => {
    const { groups } = withFields((fields) =>
      fields.map((field) =>
        String(field.key).startsWith("stop3.") ? { ...field, place: "" } : field,
      ),
    );

    expect(laneOf(groups)).toEqual({ from: "Memphis, TN", to: "Dallas, TX", stops: 3 });
  });

  it("has no lane when fewer than two stops name a place", () => {
    const { groups } = withFields((fields) =>
      fields.map((field) =>
        String(field.key).startsWith("stop1.") ? field : { ...field, place: undefined },
      ),
    );

    expect(laneOf(groups)).toBeNull();
  });

  it("has no lane on a card saved before stops named their place", () => {
    expect(laneOf(reading(legacyPayload()).groups)).toBeNull();
  });
});

describe("readMoney", () => {
  it("sets a value that reads as a number as money", () => {
    expect(readMoney("2465.00")).toBe("$2,465.00");
    expect(readMoney("$1,250")).toBe("$1,250.00");
    expect(readMoney(" 4500 ")).toBe("$4,500.00");
  });

  it("leaves what does not read as a number exactly as the document wrote it", () => {
    expect(readMoney("TBD")).toBe("TBD");
    expect(readMoney("1,250 USD")).toBe("1,250 USD");
    expect(readMoney("")).toBe("");
    expect(readMoney("  ")).toBe("  ");
  });
});

describe("headFigure", () => {
  it("leads with the rate exactly as read", () => {
    const { extraction, tally } = reading(currentPayload());

    expect(headFigure(extraction, tally, t)).toEqual({
      amount: "$2,465.00",
      caption: "rate as read",
      warn: false,
    });
  });

  it("names the fuel surcharge beside the rate and never adds it in", () => {
    const { extraction, tally } = withFields((fields) => [
      ...fields,
      {
        key: "fuelSurcharge",
        label: "Fuel surcharge",
        value: "315.00",
        confidence: 0.9,
        needsLook: false,
      },
    ]);

    expect(headFigure(extraction, tally, t)).toEqual({
      amount: "$2,465.00",
      caption: "rate as read · fuel $315.00",
      warn: false,
    });
  });

  it("has no figure on a document that carries no rate, such as a bill of lading", () => {
    const { extraction, tally } = withFields((fields) =>
      fields.filter((field) => field.key !== "rate"),
    );

    expect(headFigure(extraction, tally, t)).toBeNull();
  });

  it("warns when the rate itself needs a look", () => {
    const { extraction, tally } = reading(legacyPayload());

    expect(headFigure(extraction, tally, t)).toEqual({
      amount: "$4,500.00",
      caption: "rate as read",
      warn: true,
    });
  });

  it("says the rate was not read when the reading looked for one", () => {
    const fields = fieldsOf(currentPayload()).filter((field) => field.key !== "rate");
    const missing = reading(
      currentPayload({ fields, missingFields: [{ key: "rate", label: "Rate" }] }),
    );

    expect(headFigure(missing.extraction, missing.tally, t)).toEqual({
      amount: "Rate not read",
      caption: "check the document",
      warn: true,
    });
  });
});

describe("headSubtitle", () => {
  it("names the load, who is billed and how many stops", () => {
    const { extraction, groups } = reading(currentPayload());

    expect(headSubtitle(extraction, laneOf(groups), t)).toBe(
      "Load NL-482913 · Northline Logistics · 3 stops",
    );
  });

  it("leaves out what was not read and the stop count when there is no lane", () => {
    const { extraction } = withFields((fields) => fields.filter((field) => field.key !== "billTo"));

    expect(headSubtitle(extraction, null, t)).toBe("Load NL-482913");
    expect(headSubtitle({ fields: [] }, null, t)).toBe("");
  });

  it("says one stop in the singular", () => {
    expect(headSubtitle({ fields: [] }, { from: "A", to: "B", stops: 1 }, t)).toBe("1 stop");
  });
});

describe("headStanding", () => {
  const stand = (stage: ShipmentStage, checks = 4) => headStanding(stage, checks, t);

  it("says where the shipment stands once there is one to speak of", () => {
    expect(stand({ stage: "created", shipmentId: "shp_1", name: "S-1" })).toEqual({
      label: "Shipment created",
      tone: "success",
      mark: "check",
    });
    expect(stand({ stage: "attached", shipmentId: "shp_1" })).toEqual({
      label: "Already a shipment",
      tone: "success",
      mark: "check",
    });
    expect(stand({ stage: "waiting", proposalId: "aprop_1" }).label).toBe(
      "Drafted, waiting on you",
    );
    expect(stand({ stage: "creating", proposalId: "aprop_1" }).label).toBe("Creating the shipment");
    expect(stand({ stage: "drafting" })).toEqual({
      label: "Drafting the shipment",
      tone: "progress",
      mark: "dot",
    });
  });

  it("counts what needs a person before then", () => {
    for (const stage of [
      { stage: "none" },
      { stage: "notDrafted" },
      { stage: "failed", reason: "no" },
      { stage: "declined" },
    ] as ShipmentStage[]) {
      expect(stand(stage)).toEqual({ label: "4 to check", tone: "warn", mark: "dot" });
    }
    expect(stand({ stage: "none" }, 1).label).toBe("1 to check");
    expect(stand({ stage: "none" }, 0)).toEqual({
      label: "Nothing to check",
      tone: "success",
      mark: "check",
    });
  });
});

describe("headFacts", () => {
  it("counts what was not read and what was read clearly", () => {
    expect(headFacts(reading(currentPayload()).tally, t)).toEqual(["3 not read", "7 read clearly"]);
  });

  it("leaves out a count of nothing", () => {
    expect(headFacts({ check: [], missing: [], read: 0 }, t)).toEqual([]);
    expect(headFacts({ check: [], missing: [], read: 1 }, t)).toEqual(["1 read clearly"]);
  });
});
