import { describe, expect, it } from "vitest";
import {
  buildArchiveMessagesQueryString,
  buildMessageJsonFilename,
  buildX12Filename,
  formatRawX12Display,
  groupDiagnostics,
  parseX12Segments,
} from "../designer/utils/edi-message-utils";
import {
  detectX12Delimiters,
  formatX12Document,
  parseX12Document,
} from "../designer/inspector/utils/x12-parser";
import { diagnosticFamilyLabel } from "../designer/inspector/components/diagnostics-tab";

describe("message archive helpers", () => {
  it("formats default terminated raw X12 as one segment per line", () => {
    expect(formatRawX12Display("ISA~GS~ST~")).toBe("ISA~\nGS~\nST~");
  });

  it("formats raw X12 with a custom envelope terminator", () => {
    expect(formatRawX12Display("ISA!GS!ST!", { segmentTerminator: "!" })).toBe("ISA!\nGS!\nST!");
  });

  it("does not add blank lines to already formatted raw X12", () => {
    const formatted = "ISA~\nGS~\nST~";

    expect(formatRawX12Display(formatted)).toBe(formatted);
    expect(formatRawX12Display(formatRawX12Display(formatted))).toBe(formatted);
  });

  it("does not add a terminator when raw X12 is missing the trailing terminator", () => {
    expect(formatRawX12Display("ISA~GS~ST")).toBe("ISA~\nGS~\nST");
  });

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

  it("detects delimiters from envelope before ISA and fallback", () => {
    expect(
      detectX12Delimiters("ISA*00~", {
        elementSeparator: "|",
        segmentTerminator: "!",
        componentSeparator: ":",
        repetitionSeparator: "^",
      }),
    ).toMatchObject({ element: "|", segment: "!", component: ":", source: "envelope" });

    const isa =
      "ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *260519*1200*^*00401*000000001*0*T*>~";
    expect(detectX12Delimiters(isa)).toMatchObject({
      element: "*",
      segment: "~",
      component: ">",
      repetition: "^",
      source: "isa",
    });
    expect(detectX12Delimiters("B2*SHIP~")).toMatchObject({
      element: "*",
      segment: "~",
      component: ">",
      repetition: "^",
      source: "fallback",
    });
  });

  it("preserves empty elements, trailing elements, composites, malformed segments, and controls", () => {
    const parsed = parseX12Document(
      "ISA*00**00**ZZ*SENDER*ZZ*RECEIVER*260519*1200*^*00401*000000001*0*T*>~ST*204*0001~B2**SHIP**PP~N1*SF*Dock>Apt>~BADSEGMENT*X~SE*4*0001~",
    );

    expect(parsed.segments[2]?.elements.map((element) => element.value)).toEqual([
      "",
      "SHIP",
      "",
      "PP",
    ]);
    expect(parsed.segments[3]?.elements[1]?.components).toEqual([
      { index: 1, value: "Dock" },
      { index: 2, value: "Apt" },
      { index: 3, value: "" },
    ]);
    expect(parsed.segments[4]?.malformed).toBe(true);
    expect(parsed.metadata.controlNumbers).toMatchObject({
      isa: "000000001",
      st: "0001",
      se: "0001",
    });
    expect(parsed.metadata.seSegmentCount).toEqual({
      expected: 4,
      actual: 5,
      matches: false,
    });
  });

  it("formats parsed X12 with labels, empty markers, composites, and diagnostics", () => {
    const parsed = parseX12Document("ST*204*0001~B2**SHIP**PP~SE*3*0001~");
    const formatted = formatX12Document(parsed, [
      {
        severity: "Error",
        code: "required",
        segmentId: "B2",
        elementPosition: 2,
        path: "shipment.shipmentId",
        message: "Shipment is required",
        suggestedFix: null,
      },
    ]);

    expect(formatted).toContain("B2 - Beginning Segment for Shipment Information");
    expect(formatted).toContain("B201 Standard Carrier Alpha Code: [empty]");
    expect(formatted).toContain("! Error: Shipment is required");
  });

  it("maps diagnostic code families", () => {
    expect(diagnosticFamilyLabel("starlark_runtime_error")).toBe("starlark");
    expect(diagnosticFamilyLabel("transform_parse_error")).toBe("transform");
    expect(diagnosticFamilyLabel("condition_error")).toBe("condition");
    expect(diagnosticFamilyLabel("required")).toBe("required");
    expect(diagnosticFamilyLabel("max_length")).toBe("max length");
    expect(diagnosticFamilyLabel("render_error")).toBe("render");
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
