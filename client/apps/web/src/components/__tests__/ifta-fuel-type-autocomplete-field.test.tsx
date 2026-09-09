import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { useForm, useWatch } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { IftaFuelTypeAutocompleteField } from "../autocomplete-fields";

// The fuel a tractor burns is an IFTA_FUEL_TYPE select-option resource served by
// GraphQL. It has no REST route, so the picker carries no `link` and every
// request it makes must be a SelectOptions operation.

const CURSOR = "eyJpZCI6IkRpZXNlbCJ9";

type Values = { fuelType: string };

type FuelOption = {
  id: string;
  label: string;
  description: string;
  meta: Record<string, boolean>;
};

const DIESEL: FuelOption = {
  id: "Diesel",
  label: "Diesel",
  description: "Reported on the quarterly IFTA return",
  meta: { countsForIfta: true, gaseous: false },
};

const REEFER: FuelOption = {
  id: "Reefer",
  label: "Reefer fuel",
  description: "Not reported on the quarterly IFTA return",
  meta: { countsForIfta: false, gaseous: false },
};

function selectOptionsResponse(nodes: FuelOption[]) {
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

function Harness({ defaultValue = "" }: { defaultValue?: string }) {
  const form = useForm<Values>({ defaultValues: { fuelType: defaultValue } });
  const current = useWatch({ control: form.control, name: "fuelType" });

  return (
    <>
      <IftaFuelTypeAutocompleteField<Values>
        control={form.control}
        name="fuelType"
        label="Fuel Type"
        placeholder="Select a fuel type"
      />
      <output aria-label="fuel type value">{current ?? ""}</output>
    </>
  );
}

function renderField(defaultValue?: string) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <Harness defaultValue={defaultValue} />
    </QueryClientProvider>,
  );
}

function requestBodies(fetchMock: ReturnType<typeof vi.fn>) {
  return fetchMock.mock.calls.map(([, init]) => JSON.parse((init as RequestInit).body as string));
}

function requestUrls(fetchMock: ReturnType<typeof vi.fn>) {
  return fetchMock.mock.calls.map(([input]) => String(input));
}

describe("IftaFuelTypeAutocompleteField", () => {
  afterEach(() => {
    clearCsrfToken();
    cleanup();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("writes the chosen fuel's wire value into the form, not the default", async () => {
    setCsrfToken("graphql-token");
    const user = userEvent.setup();
    const fetchMock = vi.fn(async () => selectOptionsResponse([DIESEL, REEFER]));
    vi.stubGlobal("fetch", fetchMock);

    renderField("Diesel");

    await user.click(screen.getByRole("combobox"));
    await user.click(await screen.findByText("Reefer fuel"));

    await waitFor(() => {
      expect(screen.getByLabelText("fuel type value")).toHaveTextContent("Reefer");
    });
  });

  it("searches the IFTA_FUEL_TYPE resource and stores the fuel's wire value", async () => {
    setCsrfToken("graphql-token");
    const user = userEvent.setup();
    const fetchMock = vi.fn(async () => selectOptionsResponse([DIESEL, REEFER]));
    vi.stubGlobal("fetch", fetchMock);

    renderField();

    await user.click(screen.getByRole("combobox"));

    expect(await screen.findByText("Reefer fuel")).toBeInTheDocument();
    expect(screen.getByText("Not reported on the quarterly IFTA return")).toBeInTheDocument();

    await user.click(screen.getByText("Diesel"));

    expect(screen.getByRole("combobox")).toHaveTextContent("Diesel");

    const bodies = requestBodies(fetchMock);
    expect(bodies).not.toHaveLength(0);
    for (const body of bodies) {
      expect(body).toMatchObject({
        operationName: "SelectOptions",
        variables: { input: { resource: "IFTA_FUEL_TYPE" } },
      });
    }
  });

  it("hydrates an already stored fuel through the same resource, never over REST", async () => {
    setCsrfToken("graphql-token");
    const fetchMock = vi.fn(async () => selectOptionsResponse([REEFER]));
    vi.stubGlobal("fetch", fetchMock);

    renderField("Reefer");

    await waitFor(() => {
      expect(screen.getByRole("combobox")).toHaveTextContent("Reefer fuel");
    });

    expect(requestBodies(fetchMock)).toEqual([
      expect.objectContaining({
        operationName: "SelectOptions",
        variables: {
          input: expect.objectContaining({ resource: "IFTA_FUEL_TYPE", ids: ["Reefer"] }),
        },
      }),
    ]);
    for (const url of requestUrls(fetchMock)) {
      expect(url).toContain("graphql");
    }
  });
});
