import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Shipment, ShipmentMove } from "@trenova/shared/types/shipment";
import {
  formatUnixDateTimeShort,
  formatUnixMonthDay,
  formatUnixTime,
} from "@trenova/shared/lib/date";
import { useEffect, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MoveCard } from "../move-card";

const { getUIPolicy, getEta, getLocationOption, getAssignment, assignmentDialog } = vi.hoisted(
  () => ({
    getUIPolicy: vi.fn(),
    getEta: vi.fn(),
    getLocationOption: vi.fn(),
    getAssignment: vi.fn(),
    assignmentDialog: vi.fn(),
  }),
);

vi.mock("@/services/api", () => ({
  apiService: {
    shipmentService: { getUIPolicy, getEta },
    locationService: { getOption: getLocationOption },
    assignmentService: { get: getAssignment },
  },
}));
vi.mock("../../assignment-dialog", () => ({
  AssignmentDialog: (props: { open: boolean }) => {
    assignmentDialog(props.open);
    return null;
  },
}));
vi.mock("../../shipment-split-move-dialog", () => ({ SplitMoveDialog: () => null }));
vi.mock("@/components/carrier-assignment/rate-confirmation-actions", () => ({
  RateConfirmationActions: () => <div data-testid="rate-confirmation" />,
}));
vi.mock("@/components/carrier-assignment/carrier-tracking-link", () => ({
  CarrierTrackingLink: () => <div data-testid="tracking-link" />,
}));
vi.mock("@/components/carrier-assignment/carrier-assignment-cancel-dialog", () => ({
  CarrierAssignmentCancelDialog: () => null,
}));

// Stops as the shipment page fragment returns them: each with the location it names.
function stopAt(
  id: string,
  type: "Pickup" | "Delivery",
  location: { id: string; name: string; city: string; abbreviation: string },
  extra: Record<string, unknown> = {},
) {
  return {
    id,
    type,
    status: "New",
    scheduleType: "Open",
    sequence: type === "Pickup" ? 0 : 1,
    locationId: location.id,
    scheduledWindowStart: 1_793_000_000,
    scheduledWindowEnd: null,
    location: {
      id: location.id,
      name: location.name,
      city: location.city,
      state: { abbreviation: location.abbreviation },
    },
    ...extra,
  };
}

const SECAUCUS = { id: "loc_1", name: "Secaucus DC", city: "Secaucus", abbreviation: "NJ" };
const LEXINGTON = { id: "loc_2", name: "Lexington Yard", city: "Lexington", abbreviation: "KY" };

function inTransitMove(overrides: Partial<ShipmentMove> = {}): ShipmentMove {
  return {
    id: "smv_1",
    shipmentId: "shp_1",
    status: "InTransit",
    loaded: true,
    sequence: 0,
    distance: 2828.6,
    distanceSource: "PCMiler",
    stops: [
      stopAt("stp_1", "Pickup", SECAUCUS, {
        status: "Completed",
        actualArrival: 1_793_000_100,
        actualDeparture: 1_793_003_600,
      }),
      stopAt("stp_2", "Delivery", LEXINGTON),
    ],
    carrierAssignment: {
      id: "cas_1",
      status: "Confirmed",
      carrier: { name: "Hayes & Sons Transport", scac: "UQLG" },
      totalCost: "6736.22",
      baseAmount: "5170",
      fuelSurcharge: "1566.22",
      accessorialTotal: "0",
      currencyCode: "USD",
      proNumber: "MH7391648",
      externalDriverName: "Lawrence Sullivan",
      externalDriverPhone: "+16784914998",
    },
    ...overrides,
  } as unknown as ShipmentMove;
}

function Harness({
  move,
  shipmentId = "shp_1",
  stopError,
}: {
  move: ShipmentMove;
  shipmentId?: string | undefined;
  stopError?: string;
}) {
  const form = useForm<Shipment>({
    defaultValues: { id: shipmentId, moves: [move] } as unknown as Shipment,
  });
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } }),
  );
  useEffect(() => {
    if (stopError) form.setError("moves.0.stops.1.locationId", { message: stopError });
  }, [form, stopError]);
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <FormProvider {...form}>
          <MoveCard moveIndex={0} allowPersistedMoveRemoval onEdit={vi.fn()} onRemove={vi.fn()} />
        </FormProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

function stopRows(): HTMLElement[] {
  return screen.getAllByRole("listitem").filter((item) => item.hasAttribute("data-stop-phase"));
}

afterEach(cleanup);

beforeEach(() => {
  vi.clearAllMocks();
  getUIPolicy.mockResolvedValue({ allowMoveRemovals: true });
  getEta.mockResolvedValue({ stops: [] });
});

describe("MoveCard", () => {
  it("heads the move with its number, status and distance, and names its menu", () => {
    render(<Harness move={inTransitMove()} />);

    expect(screen.getByText("Move 1")).toBeInTheDocument();
    expect(screen.getByText("In transit")).toBeInTheDocument();
    expect(screen.getByText("2,828.6 mi")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Move 1 actions" })).toBeInTheDocument();
  });

  it("says in one sentence where the load is arriving and when", () => {
    render(<Harness move={inTransitMove()} />);

    expect(
      screen.getByRole("heading", {
        name: `Arriving at Lexington, KY, ${formatUnixDateTimeShort(1_793_000_000)}`,
      }),
    ).toBeInTheDocument();
  });

  it("writes a stop's window as its date, start and end on lines of their own", () => {
    const move = inTransitMove();
    move.stops[1] = { ...move.stops[1], scheduledWindowEnd: 1_793_007_200 };
    render(<Harness move={move} />);

    const delivery = stopRows()[1]!;
    expect(within(delivery).getByText(formatUnixMonthDay(1_793_000_000))).toBeInTheDocument();
    expect(within(delivery).getByText(formatUnixTime(1_793_000_000))).toBeInTheDocument();
    expect(within(delivery).getByText(`to ${formatUnixTime(1_793_007_200)}`)).toBeInTheDocument();
  });

  it("puts the truck on the leg it is driving, with the leg's distance", () => {
    render(<Harness move={inTransitMove()} />);

    expect(screen.getByText("On the way · 2,828.6 mi")).toBeInTheDocument();
  });

  it("draws each stop from the location it carries, asking the server for none", () => {
    render(<Harness move={inTransitMove()} />);

    expect(screen.getByRole("button", { name: "Secaucus DC" })).toBeInTheDocument();
    expect(screen.getByText("Pickup · Secaucus, NJ")).toBeInTheDocument();
    expect(screen.getByText("Delivery · Lexington, KY")).toBeInTheDocument();
    expect(getLocationOption).not.toHaveBeenCalled();
  });

  it("reads the location of a stop changed since the shipment was read", async () => {
    getLocationOption.mockResolvedValue({
      id: "loc_9",
      name: "Columbus Hub",
      city: "Columbus",
      state: { abbreviation: "OH" },
    });
    const move = inTransitMove();
    move.stops[1] = { ...move.stops[1], locationId: "loc_9" };
    render(<Harness move={move} />);

    expect(await screen.findByRole("button", { name: "Columbus Hub" })).toBeInTheDocument();
    expect(getLocationOption).toHaveBeenCalledTimes(1);
    expect(getLocationOption).toHaveBeenCalledWith("loc_9");
  });

  it("marks a departed stop done and offers the next stop's arrival check call", () => {
    render(<Harness move={inTransitMove()} />);

    const [pickup, delivery] = stopRows();
    expect(pickup).toHaveAttribute("data-stop-phase", "done");
    expect(within(pickup!).getByText(/^Departed /)).toBeInTheDocument();
    expect(delivery).toHaveAttribute("data-stop-phase", "next");
    expect(within(delivery!).getByRole("button", { name: "Arrive" })).toBeInTheDocument();
  });

  it("says how many problems a stop has instead of its progress", async () => {
    render(<Harness move={inTransitMove()} stopError="Location is required" />);

    const delivery = stopRows()[1]!;
    expect(await within(delivery).findByText("1 problem to fix")).toBeInTheDocument();
    expect(delivery).toHaveAttribute("data-stop-phase", "invalid");
  });

  it("covers the move with its carrier in one line that opens to the details", async () => {
    render(<Harness move={inTransitMove()} />);

    expect(screen.getByText("Hayes & Sons Transport")).toBeInTheDocument();
    expect(screen.getByText("UQLG · Lawrence Sullivan · Pro MH7391648")).toBeInTheDocument();
    expect(screen.getByText("$6,736.22")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /Call/ })).not.toBeInTheDocument();
    expect(screen.queryByText("+16784914998")).not.toBeInTheDocument();
    expect(screen.queryByText("MH7391648")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /Hayes & Sons Transport/ }));

    expect(await screen.findByText("MH7391648")).toBeInTheDocument();
    const cost = screen.getByLabelText("Carrier cost");
    expect(within(cost).getByText("Base rate")).toBeInTheDocument();
    expect(within(cost).getByText("$5,170.00")).toBeInTheDocument();
    expect(within(cost).getByText("Fuel surcharge")).toBeInTheDocument();
    expect(within(cost).getByText("$1,566.22")).toBeInTheDocument();
    expect(within(cost).getByText("$0.00")).toBeInTheDocument();
    expect(screen.getByTestId("rate-confirmation")).toBeInTheDocument();
  });

  it("marks where the carrier's tender stands with a dot, said in words for screen readers", () => {
    const { container } = render(<Harness move={inTransitMove()} />);

    expect(container.querySelector("[data-tender]")).toHaveAttribute("data-tender", "Confirmed");
    expect(screen.queryByText("Confirmed")).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", {
        name: /Hayes & Sons Transport · The carrier confirmed the rate for this move/,
      }),
    ).toBeInTheDocument();
  });

  it("names the driver and equipment the move carries without reading the assignment", () => {
    render(
      <Harness
        move={inTransitMove({
          status: "Assigned",
          carrierAssignment: null,
          assignment: {
            id: "asn_1",
            status: "New",
            primaryWorker: { id: "wrk_1", firstName: "Julie", lastName: "Walker" },
            tractor: { id: "trc_1", code: "0630" },
            trailer: { id: "trl_1", code: "530760" },
          },
        } as unknown as Partial<ShipmentMove>)}
      />,
    );

    expect(screen.getByText("Julie Walker")).toBeInTheDocument();
    expect(screen.getByText("Tractor 0630 · Trailer 530760")).toBeInTheDocument();
    expect(getAssignment).not.toHaveBeenCalled();
  });

  it("offers to assign a saved move nobody covers", async () => {
    render(
      <Harness
        move={inTransitMove({ status: "New", carrierAssignment: null, assignment: null })}
      />,
    );

    expect(screen.getByText("No driver or carrier yet")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Assign" }));

    await waitFor(() => expect(assignmentDialog).toHaveBeenLastCalledWith(true));
  });

  it("says a move on an unsaved shipment is assigned once it is saved", () => {
    render(
      <Harness
        shipmentId={undefined}
        move={inTransitMove({
          id: undefined,
          status: "New",
          carrierAssignment: null,
          assignment: null,
        })}
      />,
    );

    expect(screen.getByText("Save the shipment to assign a driver or carrier")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Assign" })).not.toBeInTheDocument();
  });
});
