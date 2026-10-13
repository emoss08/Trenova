import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { useForm } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FuelCardAutocompleteField } from "../autocomplete-fields";

const CURSOR = "eyJpZCI6ImZjcmRfMSJ9";

type Values = { fuelCardId: string };

type FuelCardOption = {
  id: string;
  label: string;
  description: string;
  meta: Record<string, string | null>;
};

// Shaped like fuelCardSelectOption on the server: the label carries the last
// four, the description the provider's label, and meta the raw provider value.
const COMDATA_CARD: FuelCardOption = {
  id: "fcrd_1",
  label: "Unit 118 card •••• 4821",
  description: "Comdata",
  meta: { provider: "Comdata", lastFour: "4821", tractorId: "trk_118", workerId: null },
};

const EFS_CARD: FuelCardOption = {
  id: "fcrd_2",
  label: "Unit 204 card •••• 0917",
  description: "EFS",
  meta: { provider: "EFS", lastFour: "0917", tractorId: null, workerId: null },
};

const UNKNOWN_NETWORK_CARD: FuelCardOption = {
  id: "fcrd_3",
  label: "Spare card •••• 5550",
  description: "Fleetcor",
  meta: { provider: "Fleetcor", lastFour: "5550", tractorId: null, workerId: null },
};

function selectOptionsResponse(nodes: FuelCardOption[]) {
  return new Response(
    JSON.stringify({
      data: {
        selectOptions: {
          edges: nodes.map((node) => ({ cursor: CURSOR, node })),
          pageInfo: { hasNextPage: false, endCursor: CURSOR },
          totalCount: nodes.length,
        },
      },
    }),
    { status: 200, headers: { "Content-Type": "application/json" } },
  );
}

function Harness({ defaultValue }: { defaultValue: string }) {
  const form = useForm<Values>({ defaultValues: { fuelCardId: defaultValue } });

  return (
    <FuelCardAutocompleteField<Values>
      control={form.control}
      name="fuelCardId"
      label="Fuel card"
      placeholder="Select a card"
    />
  );
}

function renderField(defaultValue = "") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <Harness defaultValue={defaultValue} />
    </QueryClientProvider>,
  );
}

describe("FuelCardAutocompleteField", () => {
  afterEach(() => {
    clearCsrfToken();
    cleanup();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("shows each card's network beside it in the list", async () => {
    setCsrfToken("graphql-token");
    const user = userEvent.setup();
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => selectOptionsResponse([COMDATA_CARD, EFS_CARD, UNKNOWN_NETWORK_CARD])),
    );

    renderField();
    await user.click(screen.getByRole("combobox"));

    const comdata = (await screen.findByText("Unit 118 card •••• 4821")).closest("[role=option]");
    const efs = screen.getByText("Unit 204 card •••• 0917").closest("[role=option]");
    const unknown = screen.getByText("Spare card •••• 5550").closest("[role=option]");

    expect(within(comdata as HTMLElement).getByRole("img", { name: "Comdata" })).toBeTruthy();
    expect(within(efs as HTMLElement).getByRole("img", { name: "EFS" })).toBeTruthy();
    expect(within(unknown as HTMLElement).queryByRole("img")).toBeNull();
  });

  it("marks only the chosen card, at the end of its row after the label", async () => {
    setCsrfToken("graphql-token");
    const user = userEvent.setup();
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => selectOptionsResponse([COMDATA_CARD, EFS_CARD])),
    );

    renderField("fcrd_2");
    await user.click(await screen.findByRole("combobox"));

    const options = await screen.findAllByRole("option");
    const optionFor = (label: string) =>
      options.find((option) => option.textContent?.includes(label)) as HTMLElement;
    const chosen = optionFor("Unit 204 card •••• 0917");
    const other = optionFor("Unit 118 card •••• 4821");

    const chosenCheck = chosen.querySelector("[data-slot=command-item-check]");
    const otherCheck = other.querySelector("[data-slot=command-item-check]");

    expect(chosenCheck?.hasAttribute("data-checked")).toBe(true);
    expect(chosen.lastElementChild).toBe(chosenCheck);
    expect(otherCheck).not.toBeNull();
    expect(otherCheck?.hasAttribute("data-checked")).toBe(false);
  });

  it("names the network once under each card rather than repeating the last four", async () => {
    setCsrfToken("graphql-token");
    const user = userEvent.setup();
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => selectOptionsResponse([COMDATA_CARD])),
    );

    renderField();
    await user.click(screen.getByRole("combobox"));

    const row = (await screen.findByText("Unit 118 card •••• 4821")).closest(
      "[role=option]",
    ) as HTMLElement;
    expect(within(row).getByText("Comdata", { selector: "span.truncate" })).toBeTruthy();
    expect(within(row).getAllByText(/4821/)).toHaveLength(1);
  });

  it("keeps the network on the field once a card is chosen", async () => {
    setCsrfToken("graphql-token");
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => selectOptionsResponse([EFS_CARD])),
    );

    renderField("fcrd_2");

    const field = screen.getByRole("combobox");
    await waitFor(() => {
      expect(field).toHaveTextContent("Unit 204 card •••• 0917");
    });
    expect(within(field).getByRole("img", { name: "EFS" })).toBeTruthy();
  });
});
