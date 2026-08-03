import { describe, expect, it } from "vitest";
import {
  isDescriptorVisible,
  visibleDescriptors,
  type FieldDescriptor,
} from "@trenova/shared/components/capability-form-section";
import { CAPABILITIES } from "@trenova/shared/lib/capability";
import type { ResolvedModeProfile } from "@trenova/shared/types/shipment";

function descriptor(overrides: Partial<FieldDescriptor> & { name: string }): FieldDescriptor {
  return { render: () => null, ...overrides };
}

function profile(capabilities: string[]): ResolvedModeProfile {
  return {
    profileId: "mpf_1",
    profileCode: "FLATBED",
    profileName: "Flatbed",
    serviceModel: "Truckload",
    equipmentClass: "OpenDeck",
    executionParty: "Asset",
    capabilities,
    rules: {},
  } as ResolvedModeProfile;
}

describe("isDescriptorVisible", () => {
  it("always shows a field with no capability requirement", () => {
    expect(isDescriptorVisible(descriptor({ name: "customerId" }), profile([]))).toBe(true);
  });

  it("shows a gated field only when the profile declares the capability", () => {
    const gated = descriptor({
      name: "lengthFeet",
      capability: CAPABILITIES.dimensionalCargo,
    });

    expect(isDescriptorVisible(gated, profile([CAPABILITIES.dimensionalCargo]))).toBe(true);
    expect(isDescriptorVisible(gated, profile([CAPABILITIES.temperatureControl]))).toBe(false);
  });

  // An unresolved profile must not hide fields. A form that renders empty and
  // then sprouts inputs once a query settles is worse than briefly showing a
  // field the profile turns out not to want.
  it("shows a gated field while the profile is still unresolved", () => {
    const gated = descriptor({
      name: "lengthFeet",
      capability: CAPABILITIES.dimensionalCargo,
    });

    expect(isDescriptorVisible(gated, null)).toBe(true);
    expect(isDescriptorVisible(gated, undefined)).toBe(true);
  });
});

describe("visibleDescriptors", () => {
  it("keeps ungated fields and drops gated ones the profile does not carry", () => {
    const visible = visibleDescriptors(
      [
        descriptor({ name: "customerId" }),
        descriptor({ name: "temperatureMin", capability: CAPABILITIES.temperatureControl }),
        descriptor({ name: "lengthFeet", capability: CAPABILITIES.dimensionalCargo }),
      ],
      profile([CAPABILITIES.dimensionalCargo]),
    );

    expect(visible.map((entry) => entry.name)).toEqual(["customerId", "lengthFeet"]);
  });

  it("preserves declaration order", () => {
    const visible = visibleDescriptors(
      [descriptor({ name: "a" }), descriptor({ name: "b" }), descriptor({ name: "c" })],
      profile([]),
    );

    expect(visible.map((entry) => entry.name)).toEqual(["a", "b", "c"]);
  });
});
