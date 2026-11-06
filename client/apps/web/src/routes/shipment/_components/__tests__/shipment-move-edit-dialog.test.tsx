import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useEffect, useState } from "react";
import { FormProvider, type FieldErrors, type Resolver, useForm } from "react-hook-form";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MoveEditDialog, type MoveDialogState } from "../shipment-move-edit-dialog";

const { fetchGraphQLSelectOptions, fetchGraphQLSelectedOption } = vi.hoisted(() => ({
  fetchGraphQLSelectOptions: vi.fn(async () => ({ results: [], next: null, count: 0 })),
  fetchGraphQLSelectedOption: vi.fn(async () => null),
}));

vi.mock("@/lib/graphql/select-options", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/select-options")>()),
  fetchGraphQLSelectOptions,
  fetchGraphQLSelectedOption,
}));

function stop(type: "Pickup" | "Delivery", sequence: number) {
  return {
    status: "New",
    type,
    scheduleType: "Open",
    handling: "Live",
    locationId: "",
    sequence,
    scheduledWindowStart: 1_793_000_000,
    scheduledWindowEnd: null,
  };
}

const MOVE = {
  status: "New",
  loaded: true,
  sequence: 0,
  distance: 0,
  stops: [stop("Pickup", 0), stop("Delivery", 1)],
};

// What the shipment's resolver says of the move, in the shape react-hook-form keeps errors.
function resolverSaying(errors: FieldErrors): Resolver<any> {
  return async (values) =>
    Object.keys(errors).length > 0 ? { values: {}, errors } : { values, errors: {} };
}

const NO_ERRORS = {} as FieldErrors;

const deliveryLocationRequired = {
  moves: [
    { stops: [undefined, { locationId: { type: "too_small", message: "Location is required" } }] },
  ],
} as unknown as FieldErrors;

function Harness({
  state,
  errors = NO_ERRORS,
  initialErrors = [],
  onClose = vi.fn(),
}: {
  state: MoveDialogState;
  errors?: FieldErrors;
  /** Errors standing on the form when the editor opens, each a form path and its message. */
  initialErrors?: [string, string][];
  onClose?: () => void;
}) {
  const form = useForm<any>({
    resolver: resolverSaying(errors),
    defaultValues: { moves: [MOVE] },
  });
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } }),
  );
  const [standing] = useState(initialErrors);
  const [ready, setReady] = useState(standing.length === 0);
  useEffect(() => {
    for (const [path, message] of standing) form.setError(path, { message });
    setReady(true);
  }, [form, standing]);

  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <FormProvider {...form}>
          {ready && <MoveEditDialog state={state} onClose={onClose} onCancel={vi.fn()} />}
        </FormProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

function stopCard(index: number): HTMLElement {
  return document.querySelector<HTMLElement>(`[data-stop-index="${index}"]`)!;
}

const EDITING = { open: true, moveIndex: 0, isNew: false } as const;

afterEach(cleanup);

describe("MoveEditDialog", () => {
  it("stays open on Save while the move has a problem, and flags the stop it is on", async () => {
    const onClose = vi.fn();
    render(<Harness state={EDITING} errors={deliveryLocationRequired} onClose={onClose} />);

    await userEvent.click(await screen.findByRole("button", { name: "Save changes" }));

    expect(await within(stopCard(1)).findByText("1 problem to fix")).toBeInTheDocument();
    expect(within(stopCard(0)).queryByText(/problem to fix/)).not.toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();
  });

  it("closes on Save once the move has nothing to fix", async () => {
    const onClose = vi.fn();
    render(<Harness state={EDITING} onClose={onClose} />);

    await userEvent.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
  });

  it("says a problem on a field the editor does not draw on the stop it belongs to", async () => {
    render(
      <Harness
        state={EDITING}
        initialErrors={[["moves.0.stops.0.location.name", "Expected a string"]]}
      />,
    );

    const said = await screen.findByText("Location: Expected a string");
    expect(said.closest("[data-stop-index]")).toBe(stopCard(0));
    expect(within(stopCard(0)).getByText("1 problem to fix")).toBeInTheDocument();
  });

  it("says a problem on the move's list of stops above the stops", async () => {
    render(
      <Harness
        state={EDITING}
        initialErrors={[["moves.0.stops.root", "A move needs a pickup and a delivery"]]}
      />,
    );

    expect(
      await screen.findByText("Stops: A move needs a pickup and a delivery"),
    ).toBeInTheDocument();
  });

  it("opens on the stop it is asked to when the field there has nothing to focus", async () => {
    render(
      <Harness
        state={{ ...EDITING, focus: "moves.0.stops.1.location.name" }}
        initialErrors={[["moves.0.stops.1.location.name", "Expected a string"]]}
      />,
    );

    await waitFor(() => expect(stopCard(1)).toHaveFocus());
  });
});
