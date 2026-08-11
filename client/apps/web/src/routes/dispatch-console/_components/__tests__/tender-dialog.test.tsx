import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Controller, type Control } from "react-hook-form";
import { MemoryRouter } from "react-router";
import { ApiRequestError } from "@trenova/shared/lib/api";
import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { TenderDialog } from "../tender-dialog";
import type { DispatchActions } from "../use-dispatch-actions";

vi.mock("@/lib/queries/dispatch-console", () => ({
  dispatchConsoleQueries: {
    liveTender: (moveId: string) => ({
      queryKey: ["test-live-tender", moveId] as const,
      queryFn: async () => null,
    }),
  },
}));

vi.mock("@/lib/graphql/routing-guide-table", () => ({
  matchRoutingGuideGraphQL: vi.fn(async () => null),
  getRoutingGuideOptionsGraphQL: vi.fn(async () => []),
}));

vi.mock("@/components/autocomplete-fields", () => ({
  CarrierAutocompleteField: ({ control, name }: { control: Control<any>; name: string }) => (
    <Controller
      control={control}
      name={name}
      render={({ field }) => (
        <input
          aria-label="Carrier"
          value={(field.value as string) ?? ""}
          onChange={(event) => field.onChange(event.target.value)}
        />
      )}
    />
  ),
}));

const PROBLEM_BASE = "https://api.trenova.app/problems/";

function overridableError(): ApiRequestError {
  return new ApiRequestError(422, {
    type: `${PROBLEM_BASE}business-rule-violation`,
    title: "Business rule violation",
    status: 422,
    detail:
      "Carrier has insurance warnings: Acme — Carrier AutoLiability insurance policy AUTO-1 expires within 30 days. Confirm the override to proceed",
    params: { overridable: "true" },
  });
}

const move = {
  moveId: "smv_1",
  proNumber: "PRO-1",
  originCity: "Dallas",
  originState: "TX",
  destinationCity: "Tulsa",
  destinationState: "OK",
  originLocationId: null,
  destinationLocationId: null,
  coverageType: "driver",
  carrierAssignmentId: null,
} as unknown as DispatchBoardMove;

function makeActions(sendSpotTender: DispatchActions["sendSpotTender"]): DispatchActions {
  return {
    sendSpotTender,
    startWaterfallTender: vi.fn(),
    cancelTender: vi.fn(),
    recordOfferResponse: vi.fn(),
    isTendering: false,
  } as unknown as DispatchActions;
}

async function submitSpotTender(actions: DispatchActions) {
  const user = userEvent.setup();
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <TenderDialog move={move} actions={actions} onClose={vi.fn()} />
      </QueryClientProvider>
    </MemoryRouter>,
  );

  await user.click(await screen.findByRole("tab", { name: "Spot" }));
  await user.type(await screen.findByLabelText("Carrier"), "car_1");
  await user.click(screen.getByRole("button", { name: "Send offers" }));
  return user;
}

describe("TenderDialog spot override flow", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows the confirm-override prompt for an overridable insurance-warning rejection", async () => {
    const sendSpotTender = vi.fn().mockRejectedValue(overridableError());

    await submitSpotTender(makeActions(sendSpotTender));

    expect(await screen.findByText("Insurance warnings")).toBeInTheDocument();
    expect(screen.getByText(/expires within 30 days/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send anyway" })).toBeInTheDocument();
    expect(sendSpotTender).toHaveBeenCalledTimes(1);
    expect(sendSpotTender.mock.calls[0][0]).toMatchObject({
      overrideInsuranceWarnings: false,
      lines: [{ carrierId: "car_1" }],
    });
  });

  it("resubmits with the override flag when the dispatcher confirms", async () => {
    const sendSpotTender = vi
      .fn()
      .mockRejectedValueOnce(overridableError())
      .mockResolvedValueOnce({ mode: "SpotBroadcast" });

    const user = await submitSpotTender(makeActions(sendSpotTender));

    await user.click(await screen.findByRole("button", { name: "Send anyway" }));

    expect(sendSpotTender).toHaveBeenCalledTimes(2);
    expect(sendSpotTender.mock.calls[1][0]).toMatchObject({
      overrideInsuranceWarnings: true,
      lines: [{ carrierId: "car_1" }],
    });
    expect(screen.queryByText("Insurance warnings")).not.toBeInTheDocument();
  });

  it("keeps the prompt hidden for non-overridable failures", async () => {
    const sendSpotTender = vi.fn().mockRejectedValue(new Error("boom"));

    await submitSpotTender(makeActions(sendSpotTender));

    expect(screen.queryByText("Insurance warnings")).not.toBeInTheDocument();
  });
});
