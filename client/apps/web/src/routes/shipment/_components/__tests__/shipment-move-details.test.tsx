import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FormProvider, type FieldErrors, type Resolver, useForm } from "react-hook-form";
import { beforeEach, describe, expect, it, vi } from "vitest";
import ShipmentMoveDetails from "../move/shipment-move-details";

const { getUIPolicy, moveCardProps } = vi.hoisted(() => ({
  getUIPolicy: vi.fn(),
  moveCardProps: [] as { allowPersistedMoveRemoval: boolean }[],
}));

vi.mock("@/services/shipment", () => ({ shipmentService: { getUIPolicy } }));

// MoveCard pulls in assignment mutations, location lookups and a split dialog,
// none of which this test is about. Stubbing it isolates the decisions the
// parent owns: which removal permission it hands down, and whether a failed save
// focuses a move's problem. The stub draws a move with a problem the way the card
// marks one.
vi.mock("../move/move-card", () => ({
  MoveCard: (props: { allowPersistedMoveRemoval: boolean }) => {
    moveCardProps.push(props);
    return (
      <div data-testid="move-card" data-move-card="" data-invalid="true">
        <button type="button" data-move-problem="">
          Move problems
        </button>
        <button type="button" data-stop-problem="">
          Stop problems
        </button>
      </div>
    );
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

function TestForm({ resolver }: { resolver?: Resolver<any> }) {
  const form = useForm({
    resolver,
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
        <form onSubmit={form.handleSubmit(() => undefined)}>
          <ShipmentMoveDetails />
          <button type="submit">Save</button>
          <output>{form.formState.submitCount}</output>
        </form>
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

// A save that fails the way the schema fails it: the errors the resolver returns, nothing saved.
function failingWith(errors: FieldErrors): Resolver<any> {
  return async () => ({ values: {}, errors });
}

const stopLocationRequired = {
  moves: [
    { stops: [undefined, { locationId: { type: "too_small", message: "Location is required" } }] },
  ],
} as unknown as FieldErrors;

describe("ShipmentMoveDetails after a failed save", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getUIPolicy.mockResolvedValue(policy({ allowMoveRemovals: true }));
  });

  // A stop's fields are drawn only in the move editor, so the form has nothing of its
  // own to focus; without this the save does nothing visible.
  it("focuses the stop's problem when the moves hold the only problems", async () => {
    render(<TestForm resolver={failingWith(stopLocationRequired)} />);
    await screen.findByTestId("move-card");

    await userEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Stop problems" })).toHaveFocus(),
    );
  });

  it("leaves focus to the form when a field above the moves has a problem too", async () => {
    render(
      <TestForm
        resolver={failingWith({
          ...stopLocationRequired,
          bol: { type: "too_small", message: "BOL is required" },
        } as FieldErrors)}
      />,
    );
    await screen.findByTestId("move-card");

    await userEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText("1", { selector: "output" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Stop problems" })).not.toHaveFocus();
  });
});
