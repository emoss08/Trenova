import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { FormProvider, useForm } from "react-hook-form";
import { beforeEach, describe, expect, it, vi } from "vitest";
import ShipmentMoveDetails from "../move/shipment-move-details";

const { getUIPolicy, moveCardProps } = vi.hoisted(() => ({
  getUIPolicy: vi.fn(),
  moveCardProps: [] as { allowPersistedMoveRemoval: boolean }[],
}));

vi.mock("@/services/api", () => ({
  apiService: { shipmentService: { getUIPolicy } },
}));

// MoveCard pulls in assignment mutations, location lookups and a split dialog,
// none of which this test is about. Stubbing it isolates the one decision the
// parent owns: which removal permission it hands down.
vi.mock("../move/move-card", () => ({
  MoveCard: (props: { allowPersistedMoveRemoval: boolean }) => {
    moveCardProps.push(props);
    return <div data-testid="move-card" />;
  },
}));

vi.mock("../shipment-move-edit-dialog", () => ({
  MoveEditDialog: () => null,
}));

function moveRemovalRule(enforcement: string, enabled = true) {
  return {
    key: "dispatch.moveRemoval",
    capability: "Core",
    label: "Move Removal",
    enforcement,
    enabled,
    fields: ["moves"],
    parameters: {},
    provenance: {
      profileId: "mpf_1",
      profileCode: "FLATBED",
      profileName: "Flatbed",
      isOrgDefault: true,
      priority: 0,
      specificityScore: 0,
      matchedOn: ["organizationDefault"],
      ruleKey: "dispatch.moveRemoval",
      capability: "Core",
      capabilityLabel: "Core Operations",
      enforcement,
      defaultEnforcement: "Block",
      overridden: false,
      rationale: "Removing a move after dispatch detaches completed stop history.",
    },
  };
}

function policy({
  allowMoveRemovals,
  rules,
}: {
  allowMoveRemovals: boolean;
  rules?: Record<string, unknown>;
}) {
  return {
    allowMoveRemovals,
    checkForDuplicateBols: true,
    checkHazmatSegregation: true,
    maxShipmentWeightLimit: 80000,
    profile: rules
      ? {
          profileId: "mpf_1",
          profileCode: "FLATBED",
          profileName: "Flatbed",
          serviceModel: "Truckload",
          equipmentClass: "OpenDeck",
          executionParty: "CompanyAsset",
          capabilities: ["Core"],
          rules,
          candidates: [],
          resolvedAt: 0,
        }
      : null,
  };
}

function TestForm() {
  const form = useForm({
    defaultValues: {
      // A persisted shipment: removal permission only matters once there are
      // saved moves whose stop history a delete would detach.
      id: "shp_1",
      moves: [{ id: "smv_1", status: "New", loaded: true, sequence: 0, distance: 0, stops: [] }],
    },
  });

  return (
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
    >
      <FormProvider {...form}>
        <ShipmentMoveDetails />
      </FormProvider>
    </QueryClientProvider>
  );
}

async function renderAndReadPermission() {
  render(<TestForm />);
  await screen.findByTestId("move-card");
  await waitFor(() => expect(getUIPolicy).toHaveBeenCalled());

  return () => moveCardProps.at(-1)?.allowPersistedMoveRemoval;
}

describe("ShipmentMoveDetails removal permission", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    moveCardProps.length = 0;
  });

  // The divergent cell that motivated this change. The server's moveRemovalRuleFor
  // lets the profile rule win, so a blocking profile must deny removal even though
  // the organization flag permits it — otherwise the operator deletes a move and
  // the save fails against a rule the form never showed.
  it("denies removal when the profile blocks it and the org flag permits", async () => {
    getUIPolicy.mockResolvedValue(
      policy({
        allowMoveRemovals: true,
        rules: { "dispatch.moveRemoval": moveRemovalRule("Block") },
      }),
    );

    const permission = await renderAndReadPermission();
    await waitFor(() => expect(permission()).toBe(false));
  });

  // The mirror case: the profile permits, so the org flag must not veto it.
  it("allows removal when the profile ignores the rule and the org flag denies", async () => {
    getUIPolicy.mockResolvedValue(
      policy({
        allowMoveRemovals: false,
        rules: { "dispatch.moveRemoval": moveRemovalRule("Ignore") },
      }),
    );

    const permission = await renderAndReadPermission();
    await waitFor(() => expect(permission()).toBe(true));
  });

  it("falls back to the org flag when the profile carries no removal rule", async () => {
    getUIPolicy.mockResolvedValue(policy({ allowMoveRemovals: false, rules: {} }));

    const permission = await renderAndReadPermission();
    await waitFor(() => expect(permission()).toBe(false));
  });
});
