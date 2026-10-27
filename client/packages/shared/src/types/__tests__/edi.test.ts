import { describe, expect, it } from "vitest";
import { ediMessageInspectionSchema } from "@trenova/shared/types/edi";

describe("ediMessageInspectionSchema provenance", () => {
  // ediservice.EDIInspectionProvenance keeps its Lading version and flow instance ids nullzero:
  // a message received before Lading, or one no flow handled, has none, and a nil pulid.ID
  // marshals as null.
  const provenance = ediMessageInspectionSchema.shape.provenance;

  it("reads a message no Lading version or flow touched", () => {
    expect(
      provenance.parse({
        messageId: "edimsg_1",
        ladingVersionId: null,
        flowInstanceId: null,
        generatedAt: 1_760_000_000,
        generatedById: null,
      }),
    ).toEqual({
      messageId: "edimsg_1",
      ladingVersionId: null,
      flowInstanceId: null,
      generatedAt: 1_760_000_000,
      generatedById: null,
    });
  });

  it("keeps the version that read or wrote the message and the flow that handled it", () => {
    const parsed = provenance.parse({
      messageId: "edimsg_1",
      ladingVersionId: "ldv_1",
      flowInstanceId: "edfi_1",
      generatedAt: 1_760_000_000,
      generatedById: "usr_1",
    });
    expect(parsed.ladingVersionId).toBe("ldv_1");
    expect(parsed.flowInstanceId).toBe("edfi_1");
  });
});
