import { describe, expect, it } from "vitest";
import { getPartnerFormDefaults, toPartnerRequest } from "../edi-schemas";

describe("EDI partner form helpers", () => {
  it("defaults new partners to an external setup", () => {
    expect(getPartnerFormDefaults()).toMatchObject({
      kind: "External",
      status: "Active",
      country: "US",
      enabledForInbound: true,
      enabledForOutbound: true,
      settingsJson: "{}",
    });
  });

  it("omits pair-controlled internal fields for external payloads", () => {
    const request = toPartnerRequest({
      ...getPartnerFormDefaults(),
      internalOrganizationId: "org_target",
      ediConnectionId: "edicn_123",
      code: "EXT",
      name: "External Partner",
      enabledForInbound: false,
      enabledForOutbound: false,
    });

    expect(request.kind).toBe("External");
    expect(request.internalOrganizationId).toBeUndefined();
    expect(request.ediConnectionId).toBeUndefined();
    expect(request.enabledForInbound).toBe(false);
    expect(request.enabledForOutbound).toBe(false);
  });

  it("preserves supported fields and parsed settings for edit payloads", () => {
    const request = toPartnerRequest({
      ...getPartnerFormDefaults(),
      code: "CARR",
      name: "Carrier Partner",
      customerId: "cust_123",
      defaultTransportId: "edicp_123",
      defaultMappingProfileId: "edimp_123",
      timezone: "America/New_York",
      settingsJson: '{ "envelope": { "receiverId": "CARR" } }',
      version: 3,
    });

    expect(request).toMatchObject({
      code: "CARR",
      name: "Carrier Partner",
      customerId: "cust_123",
      defaultTransportId: "edicp_123",
      defaultMappingProfileId: "edimp_123",
      timezone: "America/New_York",
      settings: { envelope: { receiverId: "CARR" } },
      version: 3,
    });
  });
});
