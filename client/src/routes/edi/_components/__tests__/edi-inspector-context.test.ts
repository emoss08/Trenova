import type {
  EDIDiagnostic,
  EDIDocumentPreview,
  EDIPartnerDocumentProfile,
  EDIX12EnvelopeSettings,
  EDIMessageInspection,
} from "@/types/edi";
import { describe, expect, it } from "vitest";
import {
  buildMessageInspectorContext,
  buildPreviewInspectRequest,
  buildPreviewInspectorContext,
} from "../designer/inspector/inspector-context";

describe("EDI inspector context", () => {
  it("builds the backend inspect request from a provisional preview", () => {
    const diagnostics: EDIDiagnostic[] = [
      {
        severity: "Warning",
        code: "render_missing_value",
        segmentId: "N1",
        elementPosition: 2,
        path: "partner.name",
        message: "Partner name is missing.",
        suggestedFix: "Populate partner.name.",
      },
    ];
    const envelope = x12Envelope();
    const preview = documentPreview({
      profile: partnerDocumentProfile({
        transactionSet: "204",
        envelope,
      }),
      diagnostics,
    });

    expect(buildPreviewInspectRequest(preview)).toEqual({
      rawX12: preview.rawX12,
      transactionSet: "204",
      x12Version: "005010",
      envelope,
      diagnostics,
    });
  });

  it("maps provisional previews without archived-only payload or provenance", () => {
    const context = buildPreviewInspectorContext(documentPreview());

    expect(context.title).toBe("Preview 0003");
    expect(context.status).toEqual({ label: "Provisional", variant: "info" });
    expect(context.payload).toBeUndefined();
    expect(context.provenanceRows).toBeUndefined();
    expect(context.controlRows[0]).toEqual(["Interchange Control Number (Provisional)", "0001"]);
    expect(context.rawFilename).toBe("edi-preview-x12-0003.x12");
  });

  it("maps archived messages with payload and provenance sections", () => {
    const context = buildMessageInspectorContext(messageInspection());

    expect(context.title).toBe("Message 0003");
    expect(context.status).toEqual({ label: "Generated", variant: "active" });
    expect(context.payload?.filename).toBe("edi-message-msg_1.json");
    expect(context.provenanceRows).toContainEqual(["Message ID", "msg_1"]);
    expect(context.controlRows[2]).toEqual(["Transaction Control Number", "0003"]);
  });
});

function documentPreview(overrides: Partial<EDIDocumentPreview> = {}): EDIDocumentPreview {
  return {
    rawX12:
      "ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *260519*1200*^*00501*0001*0*T*>~GS*SM*SENDER*RECEIVER*20260519*1200*0002*X*005010~ST*204*0003~SE*2*0003~GE*1*0002~IEA*1*0001~",
    segmentCount: 6,
    x12Version: "005010",
    interchangeControlNumber: "0001",
    groupControlNumber: "0002",
    transactionControlNumber: "0003",
    diagnostics: [],
    profile: null,
    templateVersion: null,
    ...overrides,
  };
}

function partnerDocumentProfile(
  overrides: Partial<EDIPartnerDocumentProfile> = {},
): EDIPartnerDocumentProfile {
  return {
    id: "profile_1",
    businessUnitId: "bu_1",
    organizationId: "org_1",
    ediPartnerId: "partner_1",
    documentTypeId: "document_type_1",
    templateId: "template_1",
    templateVersionId: null,
    name: "Load Tender",
    status: "Active",
    direction: "Outbound",
    standard: "X12",
    transactionSet: "204",
    x12VersionOverride: null,
    functionalGroupId: "SM",
    envelope: x12Envelope(),
    acknowledgment: {
      expected: false,
      type: "None",
      slaInMinutes: 0,
      missingAckSeverity: "Warning",
    },
    validationMode: "WarnOnly",
    partnerSettings: {},
    version: 1,
    createdAt: 1779192000,
    updatedAt: 1779192000,
    partner: null,
    documentType: null,
    template: null,
    templateVersion: null,
    ...overrides,
  };
}

function x12Envelope(): EDIX12EnvelopeSettings {
  return {
    interchangeSenderId: "SENDER",
    interchangeReceiverId: "RECEIVER",
    applicationSenderCode: "SENDER",
    applicationReceiverCode: "RECEIVER",
    interchangeUsageIndicator: "T",
    elementSeparator: "*",
    segmentTerminator: "~",
    componentSeparator: ">",
    repetitionSeparator: "^",
  };
}

function messageInspection(): EDIMessageInspection {
  return {
    message: {
      id: "msg_1",
      businessUnitId: "bu_1",
      organizationId: "org_1",
      ediPartnerId: "partner_1",
      documentTypeId: "document_type_1",
      partnerDocumentProfileId: "profile_1",
      templateId: "template_1",
      templateVersionId: "template_version_1",
      shipmentId: null,
      transferId: null,
      direction: "Outbound",
      standard: "X12",
      transactionSet: "204",
      x12Version: "005010",
      status: "Generated",
      validationMode: "WarnOnly",
      interchangeControlNumber: "0001",
      groupControlNumber: "0002",
      transactionControlNumber: "0003",
      segmentCount: 6,
      rawX12: "ST*204*0003~SE*2*0003~",
      payloadSnapshot: { transactionSet: "204" },
      generatedById: "user_1",
      generatedAt: 1779192000,
      diagnosticCount: 0,
      partner: null,
      documentType: null,
      partnerDocumentProfile: null,
      template: null,
      templateVersion: null,
      validationErrors: [],
    },
    inspection: {
      rawX12: "ST*204*0003~SE*2*0003~",
      transactionSet: "204",
      x12Version: "005010",
      separators: {
        element: "*",
        segment: "~",
        component: ">",
        repetition: "^",
        source: "isa",
        hasConflict: false,
      },
      summary: {
        segmentCount: 2,
        groupCount: 0,
        transactionCount: 1,
        errorCount: 0,
        warningCount: 0,
        infoCount: 0,
      },
      envelope: {
        isaControlNumber: null,
        ieaControlNumber: null,
        expectedGroups: 0,
        actualGroups: 0,
      },
      groups: [],
      transactions: [],
      segments: [],
      formatted: "ST*204*0003~\nSE*2*0003~",
      diagnostics: [],
    },
    provenance: {
      messageId: "msg_1",
      profileId: "profile_1",
      templateId: "template_1",
      templateVersionId: "template_version_1",
      generatedAt: 1779192000,
      generatedById: "user_1",
    },
  };
}
