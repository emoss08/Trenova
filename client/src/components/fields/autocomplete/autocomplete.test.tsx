import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Autocomplete, buildSelectedValueLookupCandidates } from "./autocomplete";

type AccessorialChargeOption = {
  id: string;
  code: string;
  description: string;
};

function createQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
}

function renderAutocomplete() {
  const queryClient = createQueryClient();
  const value = "acc_123";

  return render(
    <QueryClientProvider client={queryClient}>
      <Autocomplete<AccessorialChargeOption, never>
        link="/accessorial-charges/select-options/"
        value={value}
        onChange={vi.fn()}
        renderOption={(option) => option.code}
        getOptionValue={(option) => option.id}
        getDisplayValue={(option) => option.code}
      />
    </QueryClientProvider>,
  );
}

describe("buildSelectedValueLookupCandidates", () => {
  it("builds deduped no-slash and trailing-slash candidates", () => {
    const candidates = buildSelectedValueLookupCandidates(
      "/accessorial-charges/select-options/",
      "acc_123",
    );

    expect(candidates.map((candidate) => new URL(candidate.url).pathname)).toEqual([
      "/api/v1/accessorial-charges/select-options/acc_123",
      "/api/v1/accessorial-charges/select-options/acc_123/",
      "/api/v1/accessorial-charges/acc_123",
      "/api/v1/accessorial-charges/acc_123/",
    ]);
  });

  it("does not duplicate a trailing-slash value candidate", () => {
    const candidates = buildSelectedValueLookupCandidates(
      "/accessorial-charges/select-options/",
      "acc_123/",
    );

    expect(candidates.map((candidate) => new URL(candidate.url).pathname)).toEqual([
      "/api/v1/accessorial-charges/select-options/acc_123/",
      "/api/v1/accessorial-charges/acc_123/",
    ]);
  });
});

describe("Autocomplete", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("displays the selected option from a trailing-slash selected-value route", async () => {
    const selectedOption: AccessorialChargeOption = {
      id: "acc_123",
      code: "FUEL",
      description: "Fuel surcharge",
    };
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const requestURL = typeof input === "string" || input instanceof URL ? input : input.url;
      const url = new URL(requestURL);

      if (url.pathname === "/api/v1/accessorial-charges/select-options/acc_123") {
        return Promise.resolve(new Response(null, { status: 404 }));
      }

      if (url.pathname === "/api/v1/accessorial-charges/select-options/acc_123/") {
        return Promise.resolve(
          new Response(JSON.stringify(selectedOption), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }

      return Promise.resolve(new Response(null, { status: 500 }));
    });

    vi.stubGlobal("fetch", fetchMock);

    renderAutocomplete();

    await waitFor(() => {
      expect(screen.getByRole("combobox")).toHaveTextContent("FUEL");
    });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
