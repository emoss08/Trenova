import { describe, expect, it } from "vitest";
import {
  buildMessageJsonFilename,
  buildX12Filename,
  groupDiagnostics,
  parseX12Segments,
} from "../designer/utils/edi-message-utils";
import { dateInputToUnix } from "../designer/utils/edi-date-utils";

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

  it("converts archive date query input to unix day bounds", () => {
    expect(dateInputToUnix("", false)).toBe(0);
    expect(dateInputToUnix("not-a-date", false)).toBe(0);
    expect(dateInputToUnix("2026-05-18", false)).toBe(
      Math.floor(new Date("2026-05-18T00:00:00").getTime() / 1000),
    );
    expect(dateInputToUnix("2026-05-18", true)).toBe(
      Math.floor(new Date("2026-05-18T23:59:59").getTime() / 1000),
    );
  });
});
