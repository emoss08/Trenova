import { describe, expect, it } from "vitest";
import { extractionFrom, isImageFile } from "../extraction-payload";
import { DOCUMENT, currentPayload, legacyPayload } from "./extraction-fixtures";

describe("extractionFrom", () => {
  it("reads the file, its kind, its pages and the shipment it became", () => {
    const extraction = extractionFrom(currentPayload({ attachedShipmentId: "shp_1" }));

    expect(extraction).toMatchObject({
      documentId: DOCUMENT,
      fileName: "Northline_RateCon.pdf",
      contentType: "application/pdf",
      documentKind: "RateConfirmation",
      kindLabel: "Rate confirmation",
      kindConfidence: 0.94,
      reviewStatus: "NeedsReview",
      pageCount: 2,
      attachedShipmentId: "shp_1",
    });
    expect(extraction.pages).toEqual([
      { number: 1, width: 1275, height: 1650 },
      { number: 2, width: 1650, height: 1275 },
    ]);
  });

  it("keeps each field's reason, readings, box and the stop part it belongs to", () => {
    const fields = extractionFrom(currentPayload()).fields;
    const byKey = new Map(fields.map((field) => [field.key, field]));

    expect(byKey.get("deliveryWindow")).toMatchObject({
      needsLook: true,
      reason: "conflict",
      readings: ["10/21/2026", "10/22/2026"],
      page: 1,
      box: { x: 0.52, y: 0.29, w: 0.11, h: 0.014 },
      stop: null,
    });
    expect(byKey.get("stop1.window")?.stop).toEqual({
      sequence: 1,
      role: "pickup",
      part: "window",
      appointment: true,
    });
    expect(byKey.get("commodity")).toMatchObject({ reason: "low", box: null, page: 1 });
    expect(byKey.get("loadNumber")).toMatchObject({ reason: null, needsLook: false });
  });

  it("reads the missing fields by key, with their labels", () => {
    expect(extractionFrom(currentPayload()).missing).toEqual([
      { key: "proNumber", label: "Pro Number" },
      { key: "pieceCount", label: "Piece Count" },
      { key: "temperature", label: "Temperature" },
    ]);
  });

  it("reads a card saved before reasons, stop parts and page sizes were kept", () => {
    const extraction = extractionFrom(legacyPayload());

    expect(extraction.pages).toEqual([]);
    expect(extraction.kindConfidence).toBeNull();
    expect(extraction.contentType).toBe("");
    expect(extraction.fields.map((field) => field.key)).toEqual(["loadNumber", "stop2", "rate"]);
    expect(extraction.fields[1]).toMatchObject({
      reason: "low",
      stop: { sequence: 2, role: "delivery", part: "whole", appointment: false },
    });
    expect(extraction.fields[2]).toMatchObject({ reason: "review", page: 0, box: null });
    expect(extraction.missing).toEqual([{ key: "", label: "Pro Number" }]);
  });

  it("drops what is not a reading and keeps boxes and confidences inside the page", () => {
    const extraction = extractionFrom({
      documentId: "doc_1",
      pageCount: 0,
      fields: [
        { key: "", value: "x" },
        { key: "rate", value: "" },
        {
          key: "weight",
          value: "38500",
          confidence: 3,
          box: { x: -1, y: 0.5, w: 2, h: 0.01 },
          page: 4,
        },
        { key: "scac", value: "RWFL", box: { x: 0.1, y: 0.1, w: 0, h: 0.01 } },
      ],
      missingFields: [{ key: "", label: "" }, "Pro Number"],
    });

    expect(extraction.fields).toHaveLength(2);
    expect(extraction.fields[0]).toMatchObject({
      confidence: 1,
      box: { x: 0, y: 0.5, w: 1, h: 0.01 },
    });
    expect(extraction.fields[1].box).toBeNull();
    expect(extraction.pageCount).toBe(4);
    expect(extraction.missing).toEqual([]);
  });
});

describe("isImageFile", () => {
  it("reads the file's type, and its name on a card that has no type", () => {
    expect(isImageFile({ contentType: "image/jpeg", fileName: "scan.pdf" })).toBe(true);
    expect(isImageFile({ contentType: "application/pdf", fileName: "photo.jpg" })).toBe(false);
    expect(isImageFile({ contentType: "", fileName: "IMG_4471.JPG" })).toBe(true);
    expect(isImageFile({ contentType: "", fileName: "ratecon.pdf" })).toBe(false);
  });
});
