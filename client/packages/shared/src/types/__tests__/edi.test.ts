import { describe, expect, it } from "vitest";
import {
  ediAcknowledgmentConfigSchema,
  ediPartnerDocumentProfileSchema,
} from "@trenova/shared/types/edi";

// The wire form of edi.AcknowledgmentConfig (services/tms/internal/modules/edi/domain/edi/
// document.go). The column defaults to '{}', so a profile saved without acknowledgment
// expectations decodes to the zero value and marshals every field: an empty type and an
// empty severity, never an absent one.
const unsetAcknowledgment = {
  expected: false,
  type: "",
  slaInMinutes: 0,
  missingAckSeverity: "",
};

const wireProfile = (acknowledgment: Record<string, unknown>) => ({
  id: "epdp_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  ediPartnerId: "edip_1",
  documentTypeId: "edidt_1",
  templateId: "editpl_1",
  templateVersionId: null,
  name: "Tyson 214",
  status: "Active",
  direction: "Outbound",
  standard: "X12",
  transactionSet: "214",
  x12VersionOverride: "",
  functionalGroupId: "QM",
  envelope: {
    interchangeSenderQualifier: "ZZ",
    interchangeSenderId: "TRENOVA",
    interchangeReceiverQualifier: "ZZ",
    interchangeReceiverId: "TYSON",
    applicationSenderCode: "TRENOVA",
    applicationReceiverCode: "TYSON",
    interchangeUsageIndicator: "P",
    elementSeparator: "*",
    segmentTerminator: "~",
    componentSeparator: ">",
    repetitionSeparator: "^",
  },
  acknowledgment,
  validationMode: "Strict",
  partnerSettings: {},
  partnerSettingsSchemaId: null,
  partnerSettingsSchemaVersion: 0,
  version: 1,
  createdAt: 1_760_000_000,
  updatedAt: 1_760_000_000,
});

describe("ediAcknowledgmentConfigSchema", () => {
  it("reads a severity the profile never set as the default warning", () => {
    expect(ediAcknowledgmentConfigSchema.parse(unsetAcknowledgment)).toEqual({
      expected: false,
      type: "",
      slaInMinutes: 0,
      missingAckSeverity: "Warning",
    });
    expect(ediAcknowledgmentConfigSchema.parse({}).missingAckSeverity).toBe("Warning");
  });

  it("keeps a severity the profile set, and still refuses one that is not a severity", () => {
    expect(
      ediAcknowledgmentConfigSchema.parse({ ...unsetAcknowledgment, missingAckSeverity: "Error" })
        .missingAckSeverity,
    ).toBe("Error");
    expect(
      ediAcknowledgmentConfigSchema.safeParse({
        ...unsetAcknowledgment,
        missingAckSeverity: "Critical",
      }).success,
    ).toBe(false);
  });
});

describe("ediPartnerDocumentProfileSchema", () => {
  it("reads a profile saved without acknowledgment expectations", () => {
    const profile = ediPartnerDocumentProfileSchema.parse(wireProfile(unsetAcknowledgment));

    expect(profile.acknowledgment.missingAckSeverity).toBe("Warning");
    expect(profile.acknowledgment.expected).toBe(false);
  });
});
