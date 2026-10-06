import { describe, expect, it } from "vitest";
import { onboardingFormDefaults } from "@/lib/onboarding-form";
import {
  onboardingFormSchema,
  onboardingStateSchema,
  toCompleteOnboardingRequest,
  type OnboardingFormValues,
} from "@/types/onboarding";

const filled: OnboardingFormValues = {
  organization: {
    name: "Rivera Freight LLC",
    timezone: "America/Chicago",
    addressLine1: "100 Main St",
    city: "Dallas",
    stateId: "us_tx",
    postalCode: "75201",
    scacCode: "rvfr",
    dotNumber: "",
  },
  operationType: "asset",
  loadSampleData: true,
};

describe("onboarding form", () => {
  it("builds the documented complete payload, leaving blank optional ids out", () => {
    const parsed = onboardingFormSchema.parse(filled);
    expect(toCompleteOnboardingRequest(parsed)).toEqual({
      organization: {
        name: "Rivera Freight LLC",
        timezone: "America/Chicago",
        addressLine1: "100 Main St",
        city: "Dallas",
        stateId: "us_tx",
        postalCode: "75201",
        scacCode: "RVFR",
      },
      operationType: "asset",
      loadSampleData: true,
    });
  });

  it("validates the optional SCAC and DOT only when they are given", () => {
    const result = onboardingFormSchema.safeParse({
      ...filled,
      organization: { ...filled.organization, scacCode: "TOOLONG", dotNumber: "12a" },
    });
    expect(result.success).toBe(false);
    const paths = result.success ? [] : result.error.issues.map((issue) => issue.path.join("."));
    expect(paths).toEqual(["organization.scacCode", "organization.dotNumber"]);
  });

  it("requires a five-digit ZIP and a state", () => {
    const result = onboardingFormSchema.safeParse({
      ...filled,
      organization: { ...filled.organization, postalCode: "7520", stateId: "" },
    });
    const paths = result.success ? [] : result.error.issues.map((issue) => issue.path.join("."));
    expect(paths.sort()).toEqual(["organization.postalCode", "organization.stateId"]);
  });

  it("prefills from the server, then the session, then the browser", () => {
    const state = onboardingStateSchema.parse({
      required: true,
      status: "pending",
      organization: { name: "", timezone: "Mars/Olympus", city: "Dallas" },
    });

    expect(
      onboardingFormDefaults({
        state,
        organizationName: "Rivera Freight LLC",
        fallbackTimezone: "America/Denver",
      }),
    ).toMatchObject({
      organization: { name: "Rivera Freight LLC", timezone: "America/Denver", city: "Dallas" },
      operationType: "both",
      loadSampleData: true,
    });
  });

  it("does not offer sample data again once it was loaded", () => {
    const state = onboardingStateSchema.parse({
      required: true,
      status: "pending",
      sampleDataLoaded: true,
      operationType: "brokerage",
    });
    expect(
      onboardingFormDefaults({ state, organizationName: undefined, fallbackTimezone: "" }),
    ).toMatchObject({ operationType: "brokerage", loadSampleData: false });
  });

  it("reads a malformed onboarding response as not required", () => {
    expect(onboardingStateSchema.parse({ required: "yes", status: "weird" })).toMatchObject({
      required: false,
      status: "completed",
    });
  });
});
