import { describe, expect, it } from "vitest";
import { getProfile, isFieldRequired, isMoveRemovalAllowed } from "@trenova/shared/lib/capability";
import { shipmentUIPolicySchema } from "@trenova/shared/types/shipment";

// The profile crosses as a JSON scalar, so nothing in the GraphQL schema
// constrains its shape — the `json` tags on modeprofile.ResolvedPolicy are the
// contract, and this is the client half of it. The fixture is written from
// those Go tags, not from what the client happens to accept.
//
// Its Go counterpart is TestResolvedPolicyJSON_MatchesTheClientContract in
// services/tms/internal/api/graphql/resolver/uipolicy_profile_test.go.
const SERVER_PAYLOAD = {
  allowMoveRemovals: true,
  checkForDuplicateBols: true,
  checkHazmatSegregation: true,
  maxShipmentWeightLimit: 80000,
  profile: {
    profileId: "mpf_01ARZ3NDEKTSV4RRFFQ69G5FAV",
    profileCode: "FLATBED",
    profileName: "Flatbed",
    serviceModel: "Truckload",
    equipmentClass: "Flatbed",
    executionParty: "CompanyAsset",
    capabilities: ["Core", "DimensionalCargo", "Permitting"],
    rules: {
      "cargo.dimensionsRequired": {
        key: "cargo.dimensionsRequired",
        capability: "DimensionalCargo",
        label: "Cargo Dimensions",
        enforcement: "Block",
        enabled: true,
        fields: ["commodities"],
        requiredFields: ["commodities"],
        provenance: {
          profileId: "mpf_01ARZ3NDEKTSV4RRFFQ69G5FAV",
          profileCode: "FLATBED",
          profileName: "Flatbed",
          isOrgDefault: true,
          priority: 0,
          specificityScore: 0,
          matchedOn: ["organizationDefault"],
          ruleKey: "cargo.dimensionsRequired",
          capability: "DimensionalCargo",
          capabilityLabel: "Dimensional Cargo",
          enforcement: "Block",
          defaultEnforcement: "Block",
          overridden: false,
          rationale: "Open deck freight cannot be planned without dimensions.",
        },
      },
      // Serialised without `requiredFields` at all: the Go tag is omitempty and
      // this rule requires nothing.
      "dispatch.moveRemoval": {
        key: "dispatch.moveRemoval",
        capability: "Core",
        label: "Move Removal",
        enforcement: "Block",
        enabled: true,
        fields: ["moves"],
        provenance: {
          profileId: "mpf_01ARZ3NDEKTSV4RRFFQ69G5FAV",
          profileCode: "FLATBED",
          profileName: "Flatbed",
          isOrgDefault: true,
          priority: 0,
          specificityScore: 0,
          matchedOn: ["organizationDefault"],
          ruleKey: "dispatch.moveRemoval",
          capability: "Core",
          capabilityLabel: "Core Operations",
          enforcement: "Block",
          defaultEnforcement: "Block",
          overridden: false,
          rationale: "Removing a move after dispatch detaches stop history.",
        },
      },
    },
    candidates: [
      {
        profileId: "mpf_01ARZ3NDEKTSV4RRFFQ69G5FAV",
        profileCode: "FLATBED",
        profileName: "Flatbed",
        priority: 0,
        specificityScore: 0,
        matchedOn: ["organizationDefault"],
        selected: true,
      },
    ],
    resolvedAt: 1700000000,
  },
};

describe("shipmentUIPolicySchema against the server payload", () => {
  it("parses a resolved profile as the server renders it", () => {
    const result = shipmentUIPolicySchema.safeParse(SERVER_PAYLOAD);

    expect(result.success).toBe(true);
  });

  it("drives the capability helpers once parsed", () => {
    const policy = shipmentUIPolicySchema.parse(SERVER_PAYLOAD);
    const profile = getProfile(policy);

    expect(profile).not.toBeNull();
    expect(isFieldRequired(profile, "commodities")).toBe(true);

    // Targeted but not required, and omitted from the payload entirely.
    expect(isFieldRequired(profile, "moves")).toBe(false);

    // The blocking profile rule overrides the permissive org flag.
    expect(isMoveRemovalAllowed(profile, policy.allowMoveRemovals)).toBe(false);
  });

  // Before the profile was added to the GraphQL selection this was the shape
  // every client received, and it must stay valid: an older cached response or
  // a tenant with no matching profile still has to parse.
  it("still parses a policy with no profile at all", () => {
    const { profile: _profile, ...withoutProfile } = SERVER_PAYLOAD;
    const result = shipmentUIPolicySchema.safeParse(withoutProfile);

    expect(result.success).toBe(true);
    expect(getProfile(result.data)).toBeNull();
  });
});
