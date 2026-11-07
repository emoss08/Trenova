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
import { handleMutationError } from "@/hooks/use-api-mutation";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { MoveCard } from "../move-card";

const { getUIPolicy, getEta, fetchGraphQLSelectedOption, getAssignment, assignmentDialog } =
  vi.hoisted(() => ({
    getUIPolicy: vi.fn(),
    getEta: vi.fn(),
    fetchGraphQLSelectedOption: vi.fn(),
    getAssignment: vi.fn(),
    assignmentDialog: vi.fn(),
  }));

vi.mock("@/services/shipment", () => ({ shipmentService: { getUIPolicy, getEta } }));

vi.mock("@/services/assignment", () => ({ assignmentService: { get: getAssignment } }));
vi.mock("@/lib/graphql/select-options", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/select-options")>()),
  fetchGraphQLSelectedOption,
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
  errors = [],
  refusal,
  onEdit = vi.fn(),
}: {
  move: ShipmentMove;
  shipmentId?: string | undefined;
  stopError?: string;
  /** Errors to set on the form, each a form path and its message. */
  errors?: [string, string][];
  /** A refusal the server sends for the shipment's save, each a field and its message. */
  refusal?: { field: string; message: string }[];
  onEdit?: (focusPath?: string) => void;
}) {
  const form = useForm<Shipment>({
    defaultValues: { id: shipmentId, moves: [move] } as unknown as Shipment,
    // The shipment form validates with its schema; this one finds nothing.
    resolver: async (values) => ({ values, errors: {} }),
  });
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } }),
  );
  // Set once, on mount: the form's errors stand for a save that already failed.
  const [initialErrors] = useState(errors);
  const [initialRefusal] = useState(refusal);
  // After the card's own effects, as a save's refusal arrives after the card has drawn.
  useEffect(() => {
    if (!initialRefusal) return;
    const extensions = {
      code: "INVALID",
      type: "https://api.trenova.app/problems/validation-error",
      traceId: "trace-A",
      errors: initialRefusal.map((error) => ({ ...error, code: "INVALID" })),
    };
    handleMutationError({
      form,
      error: new GraphQLRequestError({
        graphQLErrors: [{ ...extensions, extensions, message: "Validation failed" }],
        kind: "graphql",
        message: "Validation failed",
        status: 200,
      }),
    });
  }, [form, initialRefusal]);
  useEffect(() => {
    if (stopError) form.setError("moves.0.stops.1.locationId", { message: stopError });
    for (const [path, message] of initialErrors) {
      form.setError(path as Parameters<typeof form.setError>[0], { message });
    }
  }, [form, stopError, initialErrors]);
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <FormProvider {...form}>
          <MoveCard moveIndex={0} allowPersistedMoveRemoval onEdit={onEdit} onRemove={vi.fn()} />
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
    expect(fetchGraphQLSelectedOption).not.toHaveBeenCalled();
  });

  it("reads the location of a stop changed since the shipment was read", async () => {
    fetchGraphQLSelectedOption.mockResolvedValue({
      id: "loc_9",
      label: "Columbus Hub",
      description: "COL01",
      meta: {
        code: "COL01",
        addressLine1: "40 Broad St",
        city: "Columbus",
        stateAbbreviation: "OH",
        postalCode: "43215",
      },
    });
    const move = inTransitMove();
    move.stops[1] = { ...move.stops[1], locationId: "loc_9" };
    render(<Harness move={move} />);

    expect(await screen.findByRole("button", { name: "Columbus Hub" })).toBeInTheDocument();
    expect(screen.getByText("Delivery · Columbus, OH")).toBeInTheDocument();
    expect(fetchGraphQLSelectedOption).toHaveBeenCalledTimes(1);
    expect(fetchGraphQLSelectedOption).toHaveBeenCalledWith(
      "LOCATION",
      "loc_9",
      undefined,
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
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

  it("counts the move's problems in its header, in place of the line under the headline", async () => {
    const { container } = render(
      <Harness
        move={inTransitMove()}
        errors={[
          ["moves.0.stops.0.weight", "Weight must be a whole number"],
          ["moves.0.stops.1.locationId", "Location is required"],
        ]}
      />,
    );

    expect(await screen.findByText("2 problems to fix")).toBeInTheDocument();
    expect(screen.getAllByText("1 problem to fix")).toHaveLength(2);
    expect(container.querySelector("[data-move-card]")).toHaveAttribute("data-invalid", "true");
  });

  it("flags a problem on the move itself on the card, where no stop shows it", async () => {
    render(
      <Harness
        move={inTransitMove()}
        errors={[["moves.0.distance", "Distance cannot be negative"]]}
      />,
    );

    expect(await screen.findByText("1 problem to fix")).toBeInTheDocument();
    expect(stopRows().map((row) => row.getAttribute("data-stop-phase"))).toEqual(["done", "next"]);
  });

  it("opens the editor at the move's first problem from its header", async () => {
    const onEdit = vi.fn();
    render(
      <Harness
        move={inTransitMove()}
        onEdit={onEdit}
        errors={[
          ["moves.0.distance", "Distance cannot be negative"],
          ["moves.0.stops.1.locationId", "Location is required"],
        ]}
      />,
    );

    await userEvent.click(await screen.findByText("2 problems to fix"));

    expect(onEdit).toHaveBeenCalledWith("moves.0.distance");
  });

  it("opens the editor at the field to fix from a stop's problems", async () => {
    const onEdit = vi.fn();
    render(
      <Harness
        move={inTransitMove()}
        onEdit={onEdit}
        errors={[["moves.0.stops.1.scheduledWindowEnd", "The window must end after it starts"]]}
      />,
    );

    const delivery = stopRows()[1]!;
    await userEvent.click(await within(delivery).findByText("1 problem to fix"));

    expect(onEdit).toHaveBeenCalledWith("moves.0.stops.1.scheduledWindowEnd");
  });

  // The stop's fields mount only in the move editor, which has not been opened.
  it("shows the server's refusal of a stop field on that stop", async () => {
    render(
      <Harness
        move={inTransitMove()}
        refusal={[
          {
            field: "moves[0].stops[1].scheduledWindowEnd",
            message: "The window must end after it starts",
          },
        ]}
      />,
    );

    const delivery = stopRows()[1]!;
    expect(await within(delivery).findByText("1 problem to fix")).toBeInTheDocument();
    expect(delivery).toHaveAttribute("data-stop-phase", "invalid");
  });

  it("opens the editor at a stop's location from its name", async () => {
    const onEdit = vi.fn();
    render(<Harness move={inTransitMove()} onEdit={onEdit} />);

    await userEvent.click(screen.getByRole("button", { name: "Lexington Yard" }));

    expect(onEdit).toHaveBeenCalledWith("moves.0.stops.1.locationId");
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
