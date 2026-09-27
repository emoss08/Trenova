import { describe, expect, it } from "vitest";
import {
  DEFAULT_INTAKE_FILTER,
  INTAKE_VIEWS,
  batchFilter,
  parseIntakeFilter,
  viewStatuses,
  writeIntakeFilter,
} from "../queue-filter";

describe("intake views", () => {
  it("cover every status exactly once, and everything is every status", () => {
    const covered = INTAKE_VIEWS.filter((view) => view !== "all").flatMap(viewStatuses);
    expect(new Set(covered).size).toBe(covered.length);
    expect(covered.sort()).toEqual(
      [
        "Discarded",
        "Expired",
        "Failed",
        "Filed",
        "PartiallyFiled",
        "Processing",
        "Ready",
        "Receiving",
        "Sealed",
      ].sort(),
    );
    expect(viewStatuses("all")).toEqual([]);
  });
});

describe("the queue's address", () => {
  it("opens on what is waiting when it says nothing", () => {
    expect(parseIntakeFilter(new URLSearchParams())).toEqual(DEFAULT_INTAKE_FILTER);
  });

  it("falls back to the default for values it does not know", () => {
    const filter = parseIntakeFilter(
      new URLSearchParams("view=bogus&source=Fax&sort=Sideways&mine=yes"),
    );
    expect(filter).toEqual(DEFAULT_INTAKE_FILTER);
  });

  it("round-trips a filter and leaves defaults out", () => {
    const filter = {
      view: "closed" as const,
      source: "Print" as const,
      mine: true,
      query: "  PRO 88  ",
      sort: "ExpiringSoonest" as const,
    };
    const written = writeIntakeFilter(new URLSearchParams("batch=cbat_1"), filter);
    expect(written.get("batch")).toBe("cbat_1");
    expect(parseIntakeFilter(written)).toEqual({ ...filter, query: "PRO 88" });

    const reset = writeIntakeFilter(written, DEFAULT_INTAKE_FILTER);
    expect([...reset.keys()]).toEqual(["batch"]);
  });

  it("asks for the view's statuses and no blank search", () => {
    expect(batchFilter({ ...DEFAULT_INTAKE_FILTER, query: "   " })).toEqual({
      statuses: ["Ready", "PartiallyFiled"],
      source: null,
      mine: false,
      query: null,
      sort: "Newest",
    });
  });
});
