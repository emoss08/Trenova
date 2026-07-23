import type { EDIDiagnostic, EDIX12EnvelopeSettings } from "@trenova/shared/types/edi";
import {
  getElementLabel,
  getElementRequirement,
  getSegmentLabel,
  isControlSegment,
} from "./x12-dictionary";

export type X12Delimiters = {
  element: string;
  segment: string;
  component: string;
  repetition: string;
  source: "envelope" | "isa" | "fallback";
};

export type X12Component = {
  index: number;
  value: string;
};

export type X12Element = {
  position: number;
  value: string;
  label: string;
  required?: boolean;
  empty: boolean;
  components: X12Component[];
};

export type X12Segment = {
  index: number;
  segmentId: string;
  label: string;
  raw: string;
  rawWithTerminator: string;
  elements: X12Element[];
  malformed: boolean;
  control: boolean;
};

export type X12ControlNumbers = {
  isa?: string;
  iea?: string;
  gs?: string;
  ge?: string;
  st?: string;
  se?: string;
};

export type X12CountComparison = {
  expected: number;
  actual: number;
  matches: boolean;
};

export type X12ParseMetadata = {
  controlNumbers: X12ControlNumbers;
  seSegmentCount?: X12CountComparison;
};

export type ParsedX12Document = {
  delimiters: X12Delimiters;
  segments: X12Segment[];
  metadata: X12ParseMetadata;
};

export type SegmentDiagnostic = EDIDiagnostic & {
  segmentIndex?: number;
};

const fallbackDelimiters: X12Delimiters = {
  element: "*",
  segment: "~",
  component: ">",
  repetition: "^",
  source: "fallback",
};

export function detectX12Delimiters(
  rawX12: string,
  envelope?: Partial<EDIX12EnvelopeSettings> | null,
): X12Delimiters {
  if (hasEnvelopeDelimiters(envelope)) {
    return {
      element: envelope.elementSeparator || fallbackDelimiters.element,
      segment: envelope.segmentTerminator || fallbackDelimiters.segment,
      component: envelope.componentSeparator || fallbackDelimiters.component,
      repetition: envelope.repetitionSeparator || fallbackDelimiters.repetition,
      source: "envelope",
    };
  }

  const isaStart = rawX12.indexOf("ISA");
  const isaSegmentTerminator = rawX12.charAt(isaStart + 105);
  const isaRaw = isaStart >= 0 ? rawX12.slice(isaStart) : "";
  if (
    isaStart >= 0 &&
    rawX12.length > isaStart + 105 &&
    isaSegmentTerminator &&
    !/[A-Za-z0-9]/.test(isaSegmentTerminator) &&
    isaRaw.indexOf(isaSegmentTerminator) === 105
  ) {
    return {
      element: rawX12.charAt(isaStart + 3) || fallbackDelimiters.element,
      repetition: rawX12.charAt(isaStart + 82) || fallbackDelimiters.repetition,
      component: rawX12.charAt(isaStart + 104) || fallbackDelimiters.component,
      segment: rawX12.charAt(isaStart + 105) || fallbackDelimiters.segment,
      source: "isa",
    };
  }

  return fallbackDelimiters;
}

export function parseX12Document(
  rawX12: string,
  envelope?: Partial<EDIX12EnvelopeSettings> | null,
): ParsedX12Document {
  const delimiters = detectX12Delimiters(rawX12, envelope);
  const segments = splitSegments(rawX12, delimiters).map((rawSegment, index) =>
    parseSegment(rawSegment, index + 1, delimiters),
  );

  return {
    delimiters,
    segments,
    metadata: {
      controlNumbers: controlNumbers(segments),
      seSegmentCount: seSegmentCount(segments),
    },
  };
}

export function diagnosticsForX12Segment(
  diagnostics: EDIDiagnostic[],
  segment: X12Segment,
): SegmentDiagnostic[] {
  return diagnostics
    .filter((diagnostic) => diagnostic.segmentId === segment.segmentId)
    .map((diagnostic) => ({ ...diagnostic, segmentIndex: segment.index }));
}

export function diagnosticsForX12Element(
  diagnostics: EDIDiagnostic[],
  segment: X12Segment,
  position: number,
) {
  return diagnosticsForX12Segment(diagnostics, segment).filter(
    (diagnostic) => diagnostic.elementPosition === position,
  );
}

export function findSegmentForDiagnostic(segments: X12Segment[], diagnostic: EDIDiagnostic) {
  if (!diagnostic.segmentId) return undefined;
  return segments.find((segment) => segment.segmentId === diagnostic.segmentId);
}

export function formatX12Document(document: ParsedX12Document, diagnostics: EDIDiagnostic[]) {
  return document.segments
    .map((segment) => {
      const prefix = indentationForSegment(segment.segmentId);
      const segmentDiagnostics = diagnosticsForX12Segment(diagnostics, segment);
      const rows = segment.elements.map((element) => {
        const marker = element.empty ? "[empty]" : element.value;
        const componentText =
          element.components.length > 1
            ? ` (${element.components.map((component) => component.value || "[empty]").join(" > ")})`
            : "";
        return `${prefix}  ${segment.segmentId}${String(element.position).padStart(2, "0")} ${element.label}: ${marker}${componentText}`;
      });
      const diagnosticRows = segmentDiagnostics.map(
        (diagnostic) => `${prefix}  ! ${diagnostic.severity}: ${diagnostic.message}`,
      );
      return [`${prefix}${segment.segmentId} - ${segment.label}`, ...rows, ...diagnosticRows].join(
        "\n",
      );
    })
    .join("\n");
}

export function x12DisplayText(document: ParsedX12Document) {
  return document.segments.map((segment) => segment.rawWithTerminator).join("\n");
}

function hasEnvelopeDelimiters(
  envelope?: Partial<EDIX12EnvelopeSettings> | null,
): envelope is Partial<EDIX12EnvelopeSettings> {
  return !!(
    envelope?.elementSeparator ||
    envelope?.segmentTerminator ||
    envelope?.componentSeparator ||
    envelope?.repetitionSeparator
  );
}

function splitSegments(rawX12: string, delimiters: X12Delimiters) {
  if (!rawX12) return [];
  const chunks = rawX12.split(delimiters.segment);
  const segments: string[] = [];
  for (let index = 0; index < chunks.length; index += 1) {
    const raw = chunks[index]?.replace(/^[\r\n]+|[\r\n]+$/g, "") ?? "";
    if (!raw) continue;
    const hasTerminator = index < chunks.length - 1;
    segments.push(hasTerminator ? `${raw}${delimiters.segment}` : raw);
  }
  return segments;
}

function parseSegment(rawWithTerminator: string, index: number, delimiters: X12Delimiters) {
  const raw = rawWithTerminator.endsWith(delimiters.segment)
    ? rawWithTerminator.slice(0, -delimiters.segment.length)
    : rawWithTerminator;
  const [segmentId = "", ...rawElements] = raw.split(delimiters.element);
  const elements = rawElements.map((value, elementIndex) => {
    const position = elementIndex + 1;
    const components = value.split(delimiters.component).map((component, componentIndex) => ({
      index: componentIndex + 1,
      value: component,
    }));
    return {
      position,
      value,
      label: getElementLabel(segmentId, position),
      required: getElementRequirement(segmentId, position),
      empty: value === "",
      components,
    };
  });

  return {
    index,
    segmentId,
    label: getSegmentLabel(segmentId),
    raw,
    rawWithTerminator,
    elements,
    malformed: !/^[A-Z0-9]{2,3}$/.test(segmentId),
    control: isControlSegment(segmentId),
  };
}

function controlNumbers(segments: X12Segment[]): X12ControlNumbers {
  const numbers: X12ControlNumbers = {};
  for (const segment of segments) {
    switch (segment.segmentId) {
      case "ISA":
        numbers.isa = segment.elements[12]?.value;
        break;
      case "IEA":
        numbers.iea = segment.elements[1]?.value;
        break;
      case "GS":
        numbers.gs = segment.elements[5]?.value;
        break;
      case "GE":
        numbers.ge = segment.elements[1]?.value;
        break;
      case "ST":
        numbers.st = segment.elements[1]?.value;
        break;
      case "SE":
        numbers.se = segment.elements[1]?.value;
        break;
    }
  }
  return numbers;
}

function seSegmentCount(segments: X12Segment[]) {
  const stIndex = segments.findIndex((segment) => segment.segmentId === "ST");
  const seIndex = segments.findIndex((segment) => segment.segmentId === "SE");
  if (stIndex < 0 || seIndex < stIndex) return undefined;

  const expected = Number(segments[seIndex]?.elements[0]?.value);
  if (!Number.isFinite(expected) || expected <= 0) return undefined;

  const actual = seIndex - stIndex + 1;
  return {
    expected,
    actual,
    matches: expected === actual,
  };
}

function indentationForSegment(segmentId: string) {
  switch (segmentId) {
    case "ISA":
    case "IEA":
      return "";
    case "GS":
    case "GE":
      return "  ";
    default:
      return "    ";
  }
}
