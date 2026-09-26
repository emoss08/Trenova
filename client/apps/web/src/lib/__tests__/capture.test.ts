import {
  CAPTURE_RECORD_KINDS,
  CAPTURE_RETENTION_WARNING_SECONDS,
  captureDocumentCategory,
  captureRequestsToShow,
  captureRetention,
  isCaptureRecordKind,
} from "@/lib/capture";
import { RECORD_LINKS } from "@/config/record-links";
import { describe, expect, it } from "vitest";

const NOW = 1_800_000_000;
const DAY = 86_400;

describe("captureRetention", () => {
  it("says nothing while deletion is more than a week off", () => {
    expect(captureRetention(NOW + CAPTURE_RETENTION_WARNING_SECONDS + 1, NOW)).toEqual({
      state: "later",
    });
  });

  it("counts whole days up, so hours left still read as a day", () => {
    expect(captureRetention(NOW + 3 * 3600, NOW)).toEqual({ state: "soon", daysLeft: 1 });
    expect(captureRetention(NOW + 2 * DAY, NOW)).toEqual({ state: "soon", daysLeft: 2 });
    expect(captureRetention(NOW + 2 * DAY + 1, NOW)).toEqual({ state: "soon", daysLeft: 3 });
    expect(captureRetention(NOW + CAPTURE_RETENTION_WARNING_SECONDS, NOW)).toEqual({
      state: "soon",
      daysLeft: 7,
    });
  });

  it("is due at and past the retention date", () => {
    expect(captureRetention(NOW, NOW)).toEqual({ state: "due" });
    expect(captureRetention(NOW - DAY, NOW)).toEqual({ state: "due" });
  });
});

describe("capture record kinds", () => {
  it("open on a page, so a filed document can link to its record", () => {
    for (const kind of CAPTURE_RECORD_KINDS) {
      expect(Object.hasOwn(RECORD_LINKS, kind)).toBe(true);
    }
  });

  it("recognise only the kinds the server files onto", () => {
    expect(isCaptureRecordKind("shipment")).toBe(true);
    expect(isCaptureRecordKind("invoice")).toBe(false);
    expect(isCaptureRecordKind("")).toBe(false);
  });

  it("narrow document types only where the kind has a category of its own", () => {
    expect(captureDocumentCategory("shipment")).toBe("Shipment");
    expect(captureDocumentCategory("worker")).toBe("Worker");
    expect(captureDocumentCategory("tractor")).toBeNull();
  });
});

describe("captureRequestsToShow", () => {
  const request = (id: string, isOpen: boolean, completedAt: number | null, createdAt: number) => ({
    id,
    isOpen,
    completedAt,
    createdAt,
  });

  it("shows open requests and those finished in the last quarter hour, newest first", () => {
    const shown = captureRequestsToShow(
      [
        request("old", false, NOW - 16 * 60, NOW - 20 * 60),
        request("open", true, null, NOW - 60),
        request("recent", false, NOW - 15 * 60, NOW - 18 * 60),
        request("newest", true, null, NOW),
      ],
      NOW,
    );
    expect(shown.map((row) => row.id)).toEqual(["newest", "open", "recent"]);
  });

  it("drops a closed request with no finish time", () => {
    expect(captureRequestsToShow([request("x", false, null, NOW)], NOW)).toEqual([]);
  });
});
