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
  isMoveRemovalAllowed,
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

// Mirrors moveRemovalRuleFor in services/tms/internal/core/services/shipmentservice/
// capability_policy.go: the profile's dispatch.moveRemoval rule decides when it is
// present, and the org's allowMoveRemovals flag is consulted only as a fallback.
//
// Fixtures are written from that Go source, not from what the client did before.
// The two cells where profile and flag disagree are exactly the divergence: the
// client used to read the flag alone, so it permitted a removal the server would
// reject and forbade one the server would accept.
describe("isMoveRemovalAllowed", () => {
  function moveRemovalRule(enforcement: EnforcementLevel) {
    return makeRule({
      key: RULE_KEYS.moveRemoval,
      enforcement,
      fields: ["moves"],
    });
  }

  it.each<[EnforcementLevel, boolean, boolean]>([
    ["Block", true, false],
    ["Block", false, false],
    ["Ignore", true, true],
    ["Ignore", false, true],
    ["Warn", true, true],
    ["Warn", false, true],
  ])(
    "lets the %s rule decide over an allowMoveRemovals of %s",
    (enforcement, allowMoveRemovals, expected) => {
      expect(
        isMoveRemovalAllowed(makeProfile([moveRemovalRule(enforcement)]), allowMoveRemovals),
      ).toBe(expected);
    },
  );

  // A disabled rule is not a decision. ruleBlocks already requires enabled, so a
  // disabled Block rule must not read as blocking.
  it("treats a disabled rule as permitting removal", () => {
    const disabled = makeRule({
      key: RULE_KEYS.moveRemoval,
      enforcement: "Block",
      enabled: false,
      fields: ["moves"],
    });

    expect(isMoveRemovalAllowed(makeProfile([disabled]), false)).toBe(true);
  });

  it.each([true, false])(
    "falls back to an allowMoveRemovals of %s when the profile carries no such rule",
    (allowMoveRemovals) => {
      const unrelated = makeRule({ key: RULE_KEYS.duplicateBol, fields: ["bol"] });

      expect(isMoveRemovalAllowed(makeProfile([unrelated]), allowMoveRemovals)).toBe(
        allowMoveRemovals,
      );
    },
  );

  it.each([true, false])(
    "falls back to an allowMoveRemovals of %s when the profile is unresolved",
    (allowMoveRemovals) => {
      expect(isMoveRemovalAllowed(null, allowMoveRemovals)).toBe(allowMoveRemovals);
      expect(isMoveRemovalAllowed(undefined, allowMoveRemovals)).toBe(allowMoveRemovals);
    },
  );

  // The flag is nullish while the UI policy query is in flight. Defaulting to
  // "allowed" there would flash an enabled remove button that then disables.
  it("denies removal when the flag has not loaded and no rule decides", () => {
    expect(isMoveRemovalAllowed(null, undefined)).toBe(false);
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
  it("is true only when a blocking rule requires the field", () => {
    const profile = makeProfile([
      makeRule({
        key: RULE_KEYS.temperatureRange,
        enforcement: "Block",
        fields: ["temperatureMin", "temperatureMax"],
        requiredFields: ["temperatureMin", "temperatureMax"],
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
        requiredFields: ["temperatureMin"],
      }),
    ]);

    expect(isFieldRequired(profile, "temperatureMin")).toBe(false);
  });

  it("is false when no profile resolved", () => {
    expect(isFieldRequired(null, "temperatureMin")).toBe(false);
  });

  // Targeting a field is not requiring it. The catalog marks bol, moves and
  // weight as targeted-only: a duplicate check constrains a BOL rather than
  // demanding one, forbidding a removal does not make moves mandatory, and
  // validateWeight returns early when weight is absent instead of requiring it.
  //
  // Reading `fields` here — which is what this did before requiredFields
  // existed — marks all three required on any profile that carries the rule.
  it.each([
    [RULE_KEYS.duplicateBol, "bol"],
    [RULE_KEYS.moveRemoval, "moves"],
    [RULE_KEYS.maxShipmentWeight, "weight"],
  ])("does not require %s's field %s, which it only targets", (key, field) => {
    const profile = makeProfile([
      makeRule({ key, enforcement: "Block", fields: [field], requiredFields: [] }),
    ]);

    expect(rulesForField(profile, field)).toHaveLength(1);
    expect(isFieldRequired(profile, field)).toBe(false);
  });

  // The server drops temperatureMax from requiredFields when requireBothBounds
  // is off, leaving it in fields. The form must follow, or it marks a field
  // mandatory that the server accepts empty.
  it("follows the server's parameter narrowing rather than the targeted list", () => {
    const profile = makeProfile([
      makeRule({
        key: RULE_KEYS.temperatureRange,
        enforcement: "Block",
        fields: ["temperatureMin", "temperatureMax"],
        requiredFields: ["temperatureMin"],
        parameters: { requireBothBounds: false },
      }),
    ]);

    expect(isFieldRequired(profile, "temperatureMin")).toBe(true);
    expect(isFieldRequired(profile, "temperatureMax")).toBe(false);
    // Still explained, just not demanded.
    expect(rulesForField(profile, "temperatureMax")).toHaveLength(1);
  });

  it("treats a rule with no required fields as requiring nothing", () => {
    const profile = makeProfile([
      makeRule({ key: RULE_KEYS.hazmatSegregation, fields: ["commodities"] }),
    ]);

    expect(isFieldRequired(profile, "commodities")).toBe(false);
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
