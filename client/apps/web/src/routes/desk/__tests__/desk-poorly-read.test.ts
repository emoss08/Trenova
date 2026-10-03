import { poorlyRead } from "@/routes/desk/_components/conversation/desk-poorly-read";
import { describe, expect, it } from "vitest";

const page = (sourceKind: string, ocrConfidence: number) =>
  ({ sourceKind, ocrConfidence }) as never;

describe("poorlyRead", () => {
  it("waits for reading to finish", () => {
    expect(poorlyRead({ status: "Extracting", pages: [page("ocr", 0.1)] })).toBe(false);
  });

  it("does not blame the file for a reading that failed", () => {
    expect(poorlyRead({ status: "Failed", pages: [] })).toBe(false);
  });

  it("flags a scan read with low confidence, extracted or indexed", () => {
    expect(poorlyRead({ status: "Extracted", pages: [page("ocr", 0.3)] })).toBe(true);
    expect(poorlyRead({ status: "Indexed", pages: [page("ocr", 0.2), page("ocr", 0.6)] })).toBe(
      true,
    );
  });

  it("leaves a clear scan and native text alone", () => {
    expect(poorlyRead({ status: "Indexed", pages: [page("ocr", 0.91)] })).toBe(false);
    expect(poorlyRead({ status: "Indexed", pages: [page("native_text", 0)] })).toBe(false);
  });
});
