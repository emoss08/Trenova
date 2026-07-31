import { describe, expect, it } from "vitest";
import {
  CAPABILITIES,
  RULE_KEYS,
  describeMatch,
  enforcementLabel,
  getBooleanParameter,
  getNumericParameter,
  getProfile,
  getRule,
  hasCapability,
  isCapabilitySectionVisible,
  isFieldRequired,
  ruleApplies,
  ruleBlocks,
  rulesForField,
} from "@trenova/shared/lib/capability";
import type {
  EnforcementLevel,
  ResolvedCapabilityRule,
  ResolvedModeProfile,
} from "@trenova/shared/types/shipment";

function makeRule(
  overrides: Partial<ResolvedCapabilityRule> & { key: string },
): ResolvedCapabilityRule {
  return {
    capability: CAPABILITIES.core,
    label: "Rule",
    enforcement: "Block",
    enabled: true,
    fields: [],
    parameters: {},
    provenance: {
      profileId: "mpf_1",
      profileCode: "OTR",
      profileName: "OTR Truckload",
      isOrgDefault: true,
      priority: 0,
      specificityScore: 0,
      matchedOn: ["organizationDefault"],
      ruleKey: overrides.key,
      capability: CAPABILITIES.core,
      capabilityLabel: "Core Operations",
      enforcement: "Block",
      defaultEnforcement: "Block",
      overridden: false,
      rationale: "Because.",
    },
    ...overrides,
  };
}

function makeProfile(
  rules: ResolvedCapabilityRule[],
  capabilities: string[] = [CAPABILITIES.core],
): ResolvedModeProfile {
  return {
    profileId: "mpf_1",
    profileCode: "OTR",
    profileName: "OTR Truckload",
    serviceModel: "Truckload",
    equipmentClass: "DryVan",
    executionParty: "CompanyAsset",
    capabilities,
    rules: Object.fromEntries(rules.map((rule) => [rule.key, rule])),
    candidates: [],
    resolvedAt: 0,
  };
}

describe("getProfile", () => {
  it("returns null when the policy has no resolved profile", () => {
    expect(
      getProfile({
        allowMoveRemovals: true,
        checkForDuplicateBols: true,
        checkHazmatSegregation: true,
        maxShipmentWeightLimit: 80000,
      }),
    ).toBeNull();
  });

  it("returns null for an undefined policy", () => {
    expect(getProfile(undefined)).toBeNull();
  });
});

describe("rule predicates", () => {
  it.each<[EnforcementLevel, boolean, boolean]>([
    ["Block", true, true],
    ["RequireReview", true, false],
    ["Warn", true, false],
    ["Ignore", false, false],
  ])("enforcement %s applies=%s blocks=%s", (enforcement, applies, blocks) => {
    const rule = makeRule({ key: RULE_KEYS.maxShipmentWeight, enforcement });

    expect(ruleApplies(rule)).toBe(applies);
    expect(ruleBlocks(rule)).toBe(blocks);
  });

  it("treats a disabled rule as neither applying nor blocking", () => {
    const rule = makeRule({
      key: RULE_KEYS.maxShipmentWeight,
      enforcement: "Block",
      enabled: false,
    });

    expect(ruleApplies(rule)).toBe(false);
    expect(ruleBlocks(rule)).toBe(false);
  });

  it("treats a null rule as not applying", () => {
    expect(ruleApplies(null)).toBe(false);
    expect(ruleBlocks(null)).toBe(false);
  });
});

describe("isFieldRequired", () => {
  it("is true only when a blocking rule covers the field", () => {
    const profile = makeProfile([
      makeRule({
        key: RULE_KEYS.temperatureRange,
        enforcement: "Block",
        fields: ["temperatureMin", "temperatureMax"],
      }),
    ]);

    expect(isFieldRequired(profile, "temperatureMin")).toBe(true);
    expect(isFieldRequired(profile, "weight")).toBe(false);
  });

  it("is false when the covering rule only warns", () => {
    const profile = makeProfile([
      makeRule({
        key: RULE_KEYS.temperatureRange,
        enforcement: "Warn",
        fields: ["temperatureMin"],
      }),
    ]);

    expect(isFieldRequired(profile, "temperatureMin")).toBe(false);
  });

  it("is false when no profile resolved", () => {
    expect(isFieldRequired(null, "temperatureMin")).toBe(false);
  });
});

describe("rulesForField", () => {
  it("returns every applying rule that names the field", () => {
    const profile = makeProfile([
      makeRule({
        key: RULE_KEYS.temperatureRange,
        enforcement: "Warn",
        fields: ["temperatureMin"],
      }),
      makeRule({
        key: RULE_KEYS.maxShipmentWeight,
        enforcement: "Block",
        fields: ["weight"],
      }),
    ]);

    expect(rulesForField(profile, "temperatureMin").map((r) => r.key)).toEqual([
      RULE_KEYS.temperatureRange,
    ]);
  });

  it("omits ignored rules so the explainer stays silent about them", () => {
    const profile = makeProfile([
      makeRule({
        key: RULE_KEYS.temperatureRange,
        enforcement: "Ignore",
        fields: ["temperatureMin"],
      }),
    ]);

    expect(rulesForField(profile, "temperatureMin")).toEqual([]);
  });
});

describe("isCapabilitySectionVisible", () => {
  it("shows everything when no profile has resolved", () => {
    expect(isCapabilitySectionVisible(null, CAPABILITIES.temperatureControl)).toBe(true);
  });

  it("hides a section whose capability the profile does not declare", () => {
    const profile = makeProfile([], [CAPABILITIES.core]);

    expect(isCapabilitySectionVisible(profile, CAPABILITIES.temperatureControl)).toBe(false);
  });

  it("shows a section whose capability the profile declares", () => {
    const profile = makeProfile([], [CAPABILITIES.core, CAPABILITIES.temperatureControl]);

    expect(isCapabilitySectionVisible(profile, CAPABILITIES.temperatureControl)).toBe(true);
  });
});

describe("hasCapability and getRule", () => {
  it("reads capabilities and rules off the resolved profile", () => {
    const rule = makeRule({ key: RULE_KEYS.maxShipmentWeight });
    const profile = makeProfile([rule], [CAPABILITIES.core]);

    expect(hasCapability(profile, CAPABILITIES.core)).toBe(true);
    expect(hasCapability(profile, CAPABILITIES.hazmat)).toBe(false);
    expect(getRule(profile, RULE_KEYS.maxShipmentWeight)?.key).toBe(RULE_KEYS.maxShipmentWeight);
    expect(getRule(profile, RULE_KEYS.hazmatSegregation)).toBeNull();
  });
});

describe("parameter accessors", () => {
  it("reads typed parameters and rejects mismatched types", () => {
    const rule = makeRule({
      key: RULE_KEYS.maxShipmentWeight,
      parameters: { maxWeight: 95000, requireBothBounds: true, bogus: "text" },
    });

    expect(getNumericParameter(rule, "maxWeight")).toBe(95000);
    expect(getNumericParameter(rule, "bogus")).toBeNull();
    expect(getBooleanParameter(rule, "requireBothBounds")).toBe(true);
    expect(getBooleanParameter(rule, "maxWeight")).toBeNull();
    expect(getNumericParameter(null, "maxWeight")).toBeNull();
  });
});

describe("describeMatch and enforcementLabel", () => {
  it("renders scope matches in human terms", () => {
    expect(describeMatch(["customer", "serviceType"])).toBe("customer, service type");
    expect(describeMatch([])).toBe("no specific scope");
    expect(describeMatch(null)).toBe("no specific scope");
  });

  it("labels every enforcement level", () => {
    expect(enforcementLabel("Block")).toBe("Blocks saving");
    expect(enforcementLabel("Warn")).toBe("Warns and records");
    expect(enforcementLabel("RequireReview")).toBe("Requires review");
    expect(enforcementLabel("Ignore")).toBe("Not enforced");
  });
});
