import { describe, expect, it } from "vitest";
import {
  buildArchiveMessagesQueryString,
  buildMessageJsonFilename,
  buildX12Filename,
  groupDiagnostics,
  parseX12Segments,
} from "../designer/utils/edi-message-utils";

describe("message archive helpers", () => {
  it("splits raw X12 into ordered segments", () => {
    expect(parseX12Segments("ISA*00~GS*SM~ST*204*0001~")).toEqual([
      { index: 1, segmentId: "ISA", elements: ["00"], raw: "ISA*00" },
      { index: 2, segmentId: "GS", elements: ["SM"], raw: "GS*SM" },
      { index: 3, segmentId: "ST", elements: ["204", "0001"], raw: "ST*204*0001" },
    ]);
  });

  it("ignores empty trailing segment terminators", () => {
    expect(parseX12Segments("ISA*00~GS*SM~~").map((segment) => segment.segmentId)).toEqual([
      "ISA",
      "GS",
    ]);
  });

  it("supports custom separators and defaults", () => {
    expect(
      parseX12Segments("ISA^00!GS^SM!", {
        elementSeparator: "^",
        segmentTerminator: "!",
      }).map((segment) => segment.elements),
    ).toEqual([["00"], ["SM"]]);
    expect(parseX12Segments("ISA*00~")[0]?.elements).toEqual(["00"]);
  });

  it("groups diagnostics by display identity", () => {
    const groups = groupDiagnostics([
      {
        severity: "Error",
        code: "required",
        segmentId: "B2",
        elementPosition: 2,
        path: "shipment.bol",
        message: "Missing BOL",
        suggestedFix: "Map shipment.bol",
      },
      {
        severity: "Error",
        code: "required",
        segmentId: "B2",
        elementPosition: 2,
        path: "shipment.bol",
        message: "Still missing",
        suggestedFix: null,
      },
      {
        severity: "Warning",
        code: "max_length",
        segmentId: "L11",
        elementPosition: 1,
        path: null,
        message: "Too long",
        suggestedFix: null,
      },
    ]);

    expect(groups).toHaveLength(2);
    expect(groups[0]?.diagnostics).toHaveLength(2);
    expect(groups[1]?.severity).toBe("Warning");
  });

  it("derives archive filenames", () => {
    expect(
      buildX12Filename({
        id: "edimsg_1",
        transactionSet: "204",
        transactionControlNumber: "000042",
      }),
    ).toBe("edi-204-000042.x12");
    expect(buildMessageJsonFilename({ id: "edimsg_1" })).toBe("edi-message-edimsg_1.json");
  });

  it("builds archive query strings with unix day bounds", () => {
    const query = buildArchiveMessagesQueryString({
      partnerId: "partner_1",
      transactionSet: "204",
      direction: "Outbound",
      status: "Generated",
      generatedFrom: "2026-05-18",
      generatedTo: "2026-05-19",
      query: "  000042  ",
    });
    const params = new URLSearchParams(query.slice(1));

    expect(params.get("limit")).toBe("50");
    expect(params.get("partnerId")).toBe("partner_1");
    expect(params.get("transactionSet")).toBe("204");
    expect(params.get("direction")).toBe("Outbound");
    expect(params.get("status")).toBe("Generated");
    expect(params.get("query")).toBe("000042");
    expect(params.get("generatedFrom")).toBe(
      String(Math.floor(new Date("2026-05-18T00:00:00").getTime() / 1000)),
    );
    expect(params.get("generatedTo")).toBe(
      String(Math.floor(new Date("2026-05-19T23:59:59").getTime() / 1000)),
    );
  });

  it("omits invalid archive date filters", () => {
    const query = buildArchiveMessagesQueryString({
      generatedFrom: "",
      generatedTo: "not-a-date",
    });
    const params = new URLSearchParams(query.slice(1));

    expect(params.has("generatedFrom")).toBe(false);
    expect(params.has("generatedTo")).toBe(false);
  });
});
