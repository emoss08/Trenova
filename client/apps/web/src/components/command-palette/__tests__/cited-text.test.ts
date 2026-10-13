import type { AssistantEntityRef } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { CITE_HREF_PREFIX, linkCitedRecords } from "../cited-text";

const LATE: AssistantEntityRef = { type: "shipment", id: "shp_1", label: "S-240193" };
const ACME: AssistantEntityRef = { type: "customer", id: "cus_1", label: "Acme Foods" };
const ACME_SHORT: AssistantEntityRef = { type: "customer", id: "cus_2", label: "Acme" };

describe("linkCitedRecords", () => {
  it("links each record the tools returned where the answer names it", () => {
    const linked = linkCitedRecords("S-240193 is late for Acme Foods; S-240193 left at 8.", [
      LATE,
      ACME,
    ]);

    expect(linked.text).toBe(
      `[S-240193](${CITE_HREF_PREFIX}shipment-shp_1) is late for [Acme Foods](${CITE_HREF_PREFIX}customer-cus_1); [S-240193](${CITE_HREF_PREFIX}shipment-shp_1) left at 8.`,
    );
    expect([...linked.records.entries()]).toEqual([
      [`${CITE_HREF_PREFIX}shipment-shp_1`, LATE],
      [`${CITE_HREF_PREFIX}customer-cus_1`, ACME],
    ]);
  });

  it("prefers the longest name where two overlap", () => {
    const linked = linkCitedRecords("Acme Foods owes more than Acme Labs.", [ACME_SHORT, ACME]);

    expect(linked.text).toBe(
      `[Acme Foods](${CITE_HREF_PREFIX}customer-cus_1) owes more than [Acme](${CITE_HREF_PREFIX}customer-cus_2) Labs.`,
    );
  });

  it("leaves names inside code, links the reply wrote, and longer words alone", () => {
    const linked = linkCitedRecords(
      "See `S-240193` and [the load](/shipments/S-240193); S-2401930 is another.",
      [LATE],
    );

    expect(linked.text).toBe(
      "See `S-240193` and [the load](/shipments/S-240193); S-2401930 is another.",
    );
    expect(linked.records.size).toBe(0);
  });

  it("links nothing the tools did not return, even a name that looks like a record", () => {
    const linked = linkCitedRecords("S-240201 is the real risk.", [LATE]);

    expect(linked.text).toBe("S-240201 is the real risk.");
  });

  it("ignores a one-character label, which would mark every word it starts", () => {
    expect(linkCitedRecords("A is late", [{ type: "worker", id: "wrk_1", label: "A" }]).text).toBe(
      "A is late",
    );
  });
});
