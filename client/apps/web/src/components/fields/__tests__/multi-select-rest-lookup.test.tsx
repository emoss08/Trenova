import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { clearCsrfToken } from "@trenova/shared/lib/api";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MultiSelectAutocomplete } from "../multi-select-field";

type Doc = { id: string; name: string };

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

describe("MultiSelectAutocomplete over REST", () => {
  afterEach(() => {
    cleanup();
    clearCsrfToken();
    vi.unstubAllGlobals();
  });

  it("looks up every selected value in one request that carries the field's filters", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = new URL(String(input), window.location.origin);
      if (url.searchParams.has("ids")) {
        return jsonResponse({
          results: [
            { id: "doc_1", name: "Bill of lading" },
            { id: "doc_2", name: "Proof of delivery" },
          ],
          count: 2,
          next: null,
        });
      }
      return jsonResponse({ results: [], count: 0, next: null });
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MultiSelectAutocomplete<Doc>
        link="/documents/select-options/"
        values={["doc_1", "doc_2"]}
        onChange={() => undefined}
        extraSearchParams={{ resourceId: "shp_1" }}
        getOptionValue={(doc) => doc.id}
        getDisplayValue={(doc) => doc.name}
        renderOption={(doc) => doc.name}
        maxCount={2}
      />,
    );

    await waitFor(() => expect(screen.getByText("Bill of lading")).toBeInTheDocument());
    expect(screen.getByText("Proof of delivery")).toBeInTheDocument();

    const lookups = fetchMock.mock.calls
      .map(([input]) => new URL(String(input), window.location.origin))
      .filter((url) => url.searchParams.has("ids"));
    expect(lookups).toHaveLength(1);
    expect(lookups[0].pathname).toMatch(/\/documents\/select-options\/$/);
    expect(lookups[0].searchParams.getAll("ids")).toEqual(["doc_1", "doc_2"]);
    expect(lookups[0].searchParams.get("resourceId")).toBe("shp_1");
  });
});
