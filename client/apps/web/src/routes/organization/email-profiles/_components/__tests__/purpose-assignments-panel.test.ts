import { describe, expect, it } from "vitest";
import {
  purposeAssignmentsSchema,
  toPayload,
  type PurposeAssignmentsFormValues,
} from "../purpose-assignments-panel";

const ALL_ASSIGNED: PurposeAssignmentsFormValues = {
  General: "ep_general",
  Billing: "ep_billing",
  Reporting: "ep_reporting",
  Operations: "ep_ops",
  Authentication: "ep_auth",
  Notifications: "ep_notify",
};

describe("purpose assignments", () => {
  it("accepts a purpose the autocomplete cleared to null", () => {
    const result = purposeAssignmentsSchema.safeParse({ ...ALL_ASSIGNED, Billing: null });

    expect(result.success).toBe(true);
  });

  it("accepts a purpose that was never assigned", () => {
    const result = purposeAssignmentsSchema.safeParse({ ...ALL_ASSIGNED, Reporting: "" });

    expect(result.success).toBe(true);
  });

  it("saves every assigned purpose, in purpose order", () => {
    expect(toPayload(ALL_ASSIGNED)).toEqual([
      { purpose: "General", profileId: "ep_general" },
      { purpose: "Billing", profileId: "ep_billing" },
      { purpose: "Reporting", profileId: "ep_reporting" },
      { purpose: "Operations", profileId: "ep_ops" },
      { purpose: "Authentication", profileId: "ep_auth" },
      { purpose: "Notifications", profileId: "ep_notify" },
    ]);
  });

  it("leaves cleared and unassigned purposes out of the save, removing their assignment", () => {
    expect(
      toPayload({
        General: "ep_general",
        Billing: null,
        Reporting: "",
        Operations: null,
        Authentication: "ep_auth",
        Notifications: "",
      }),
    ).toEqual([
      { purpose: "General", profileId: "ep_general" },
      { purpose: "Authentication", profileId: "ep_auth" },
    ]);
  });

  it("saves nothing when every purpose is cleared", () => {
    expect(
      toPayload({
        General: null,
        Billing: null,
        Reporting: null,
        Operations: null,
        Authentication: null,
        Notifications: null,
      }),
    ).toEqual([]);
  });
});
