import { cleanup, render, screen, waitFor } from "@testing-library/react";
import type { SelectOption } from "@/lib/graphql/select-options";
import { afterEach, describe, expect, it, vi } from "vitest";

const fetchGraphQLSelectOptions = vi.hoisted(() => vi.fn());

vi.mock("@/lib/graphql/select-options", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/select-options")>()),
  fetchGraphQLSelectOptions,
}));

const { MultiSelectAutocomplete } = await import("../multi-select-field");

const DOCUMENTS = { resource: "DOCUMENT" } as const;

describe("MultiSelectAutocomplete", () => {
  afterEach(() => {
    cleanup();
    fetchGraphQLSelectOptions.mockReset();
  });

  it("names every selected value with one lookup that carries the field's filters", async () => {
    fetchGraphQLSelectOptions.mockImplementation(async ({ ids }: { ids?: string[] }) =>
      ids
        ? {
            count: 2,
            next: null,
            results: [
              { id: "doc_1", label: "Bill of lading", description: null, meta: null },
              { id: "doc_2", label: "Proof of delivery", description: null, meta: null },
            ],
          }
        : { count: 0, next: null, results: [] },
    );

    render(
      <MultiSelectAutocomplete<SelectOption>
        graphql={DOCUMENTS}
        values={["doc_1", "doc_2"]}
        onChange={() => undefined}
        extraSearchParams={{ resourceId: "shp_1" }}
        getOptionValue={(option) => option.id}
        getDisplayValue={(option) => option.label}
        renderOption={(option) => option.label}
        maxCount={2}
      />,
    );

    await waitFor(() => expect(screen.getByText("Bill of lading")).toBeInTheDocument());
    expect(screen.getByText("Proof of delivery")).toBeInTheDocument();

    const lookups = fetchGraphQLSelectOptions.mock.calls.filter(([params]) => params.ids);
    expect(lookups).toHaveLength(1);
    expect(lookups[0][0]).toEqual({
      resource: "DOCUMENT",
      ids: ["doc_1", "doc_2"],
      initialLimit: 2,
      filters: { resourceId: "shp_1" },
    });
  });
});
