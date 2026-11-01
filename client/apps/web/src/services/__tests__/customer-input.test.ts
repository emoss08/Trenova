import type { Customer } from "@trenova/shared/types/customer";
import { defaultCustomerTrackingProfile } from "@trenova/shared/types/customer-tracking";
import { describe, expect, it } from "vitest";
import { toCustomerInput } from "../customer";

const base = {
  status: "Active",
  code: "HALS",
  name: "Halstead Foods",
  addressLine1: "1 Main St",
  addressLine2: null,
  city: "Columbus",
  stateId: "us_1",
  postalCode: "43004",
  isGeocoded: false,
  brokerVettingEnabled: false,
  allowConsolidation: true,
  exclusiveConsolidation: false,
  consolidationPriority: 1,
  statusUpdatePreference: "None",
  statusUpdateRecipients: null,
  version: 3,
} as Customer;

describe("toCustomerInput", () => {
  it("sends the tracking settings with the customer, a blank recipient list as null", () => {
    const input = toCustomerInput({
      ...base,
      trackingProfile: {
        ...defaultCustomerTrackingProfile(),
        id: "ctp_1",
        customerId: "cus_1",
        notifyDelayed: true,
        linkTtlDays: 14,
        recipients: "   ",
        notificationsEnabledAt: 1_800_000_000,
        updatedAt: 1_800_000_000,
      },
    });

    expect(input.trackingProfile).toEqual({
      linkEnabled: true,
      visibleFields: ["Status", "Windows", "Actuals", "Eta", "References"],
      linkTtlDays: 14,
      notifyPickedUp: false,
      notifyDelayed: true,
      notifyDelivered: false,
      notifyPodReady: false,
      includeLink: true,
      recipients: null,
    });
    expect(input.version).toBe(3);
  });

  it("trims the recipients it sends", () => {
    const input = toCustomerInput({
      ...base,
      trackingProfile: {
        ...defaultCustomerTrackingProfile(),
        recipients: "  ops@halstead.example, dock@halstead.example ",
      },
    });

    expect(input.trackingProfile?.recipients).toBe("ops@halstead.example, dock@halstead.example");
  });

  it("leaves the tracking settings alone when the customer carries none", () => {
    expect(toCustomerInput(base).trackingProfile).toBeNull();
  });
});
