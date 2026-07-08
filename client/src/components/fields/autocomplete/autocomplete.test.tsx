import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { clearCsrfToken, setCsrfToken } from "@/lib/api";
import type { SelectOption } from "@/lib/graphql/select-options";
import { Autocomplete, buildSelectedValueLookupCandidates } from "./autocomplete";

const selectOptionCursor =
  "eyJjcmVhdGVkQXQiOjE3ODA0MTU4ODMsImlkIjoidHJhY18wMUtUNEdXVDlNS1EwRjZCQ0NHQTBWUjJZNSJ9";

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

function renderGraphQLAutocomplete({
  value = null,
  onChange = vi.fn(),
  onOptionChange = vi.fn(),
}: {
  value?: string | null;
  onChange?: (...event: any[]) => void;
  onOptionChange?: (option: SelectOption | null) => void;
} = {}) {
  const queryClient = createQueryClient();

  render(
    <QueryClientProvider client={queryClient}>
      <Autocomplete<SelectOption, never>
        link="/tractors/select-options/"
        graphql={{ resource: "TRACTOR" }}
        value={value}
        onChange={onChange}
        onOptionChange={onOptionChange}
        renderOption={(option) => option.label}
        getOptionValue={(option) => option.id}
        getDisplayValue={(option) => option.label}
        label="Tractor"
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
    clearCsrfToken();
    cleanup();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
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

  it("displays the selected option from GraphQL ids lookup", async () => {
    setCsrfToken("graphql-token");
    const fetchMock = vi.fn(
      async () =>
        new Response(
          JSON.stringify({
            data: {
              selectOptions: {
                edges: [
                  {
                    cursor: selectOptionCursor,
                    node: {
                      id: "trac_123",
                      label: "TRC-123",
                      description: null,
                      meta: {
                        primaryWorkerId: "wrk_primary",
                        secondaryWorkerId: "wrk_secondary",
                      },
                    },
                  },
                ],
                pageInfo: {
                  hasNextPage: false,
                  endCursor: selectOptionCursor,
                },
                totalCount: 1,
              },
            },
          }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        ),
    );
    vi.stubGlobal("fetch", fetchMock);

    renderGraphQLAutocomplete({ value: "trac_123" });

    await waitFor(() => {
      expect(screen.getByRole("combobox")).toHaveTextContent("TRC-123");
    });

    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
    expect(JSON.parse(init.body as string)).toMatchObject({
      operationName: "SelectOptions",
      variables: {
        input: {
          resource: "TRACTOR",
          ids: ["trac_123"],
        },
      },
    });
  });

  it("searches GraphQL options and passes tractor metadata through selection", async () => {
    setCsrfToken("graphql-token");
    const user = userEvent.setup();
    const onChange = vi.fn();
    const onOptionChange = vi.fn();
    const fetchMock = vi.fn(
      async () =>
        new Response(
          JSON.stringify({
            data: {
              selectOptions: {
                edges: [
                  {
                    cursor: selectOptionCursor,
                    node: {
                      id: "trac_123",
                      label: "TRC-123",
                      description: null,
                      meta: {
                        primaryWorkerId: "wrk_primary",
                        secondaryWorkerId: "wrk_secondary",
                      },
                    },
                  },
                ],
                pageInfo: {
                  hasNextPage: false,
                  endCursor: selectOptionCursor,
                },
                totalCount: 1,
              },
            },
          }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        ),
    );
    vi.stubGlobal("fetch", fetchMock);

    renderGraphQLAutocomplete({ onChange, onOptionChange });

    await user.click(screen.getByRole("combobox"));
    await user.click(await screen.findByText("TRC-123"));

    expect(onChange).toHaveBeenCalledWith("trac_123");
    expect(onOptionChange).toHaveBeenCalledWith(
      expect.objectContaining({
        id: "trac_123",
        meta: {
          primaryWorkerId: "wrk_primary",
          secondaryWorkerId: "wrk_secondary",
        },
      }),
    );
  });
});
