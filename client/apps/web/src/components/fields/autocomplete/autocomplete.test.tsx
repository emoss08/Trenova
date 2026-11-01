import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import type { SelectOption } from "@/lib/graphql/select-options";
import { Autocomplete } from "./autocomplete";

const selectOptionCursor =
  "eyJjcmVhdGVkQXQiOjE3ODA0MTU4ODMsImlkIjoidHJhY18wMUtUNEdXVDlNS1EwRjZCQ0NHQTBWUjJZNSJ9";

function createQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
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

function sourceConnection(sources: string[]) {
  return new Response(
    JSON.stringify({
      data: {
        selectOptions: {
          edges: sources.map((source) => ({
            cursor: selectOptionCursor,
            node: { id: source, label: source, description: null, meta: null },
          })),
          pageInfo: { hasNextPage: false, endCursor: selectOptionCursor },
          totalCount: sources.length,
        },
      },
    }),
    { status: 200, headers: { "Content-Type": "application/json" } },
  );
}

function stubSourceSearch(sources: string[]) {
  const fetchMock = vi.fn(async (_input: RequestInfo, init?: RequestInit) => {
    const body = JSON.parse(init?.body as string) as {
      variables: { input: { query?: string } };
    };
    const query = body.variables.input.query?.toLowerCase() ?? "";
    return sourceConnection(sources.filter((source) => source.toLowerCase().includes(query)));
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

function sourceFromText(text: string): SelectOption {
  return { id: text, label: text, description: null, meta: null };
}

function SourcePicker({
  onOptionChange,
  createOption,
}: {
  onOptionChange: (option: SelectOption | null) => void;
  createOption?: (text: string) => SelectOption;
}) {
  const [value, setValue] = useState<string | null>(null);
  return (
    <Autocomplete<SelectOption, never>
      graphql={{ resource: "BANK_RECEIPT_SOURCE" }}
      preload
      value={value}
      onChange={setValue}
      onOptionChange={onOptionChange}
      renderOption={(option) => option.label}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.label}
      createOption={createOption}
      label="Source"
    />
  );
}

function renderSourcePicker(props: Parameters<typeof SourcePicker>[0]) {
  render(
    <QueryClientProvider client={createQueryClient()}>
      <SourcePicker {...props} />
    </QueryClientProvider>,
  );
}

describe("Autocomplete", () => {
  afterEach(() => {
    clearCsrfToken();
    cleanup();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
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

  it("offers what was typed when no option has that value, and picks it", async () => {
    setCsrfToken("graphql-token");
    const user = userEvent.setup();
    const onOptionChange = vi.fn();
    const fetchMock = stubSourceSearch(["Chase", "Wells Fargo"]);
    renderSourcePicker({ onOptionChange, createOption: sourceFromText });

    await user.click(screen.getByRole("combobox"));
    await user.type(screen.getByPlaceholderText("Search source..."), "  Lockbox 7 ");
    await user.click(await screen.findByText('Use "Lockbox 7"'));

    expect(onOptionChange).toHaveBeenCalledWith({
      id: "Lockbox 7",
      label: "Lockbox 7",
      description: null,
      meta: null,
    });
    await waitFor(() => {
      expect(screen.getByRole("combobox")).toHaveTextContent("Lockbox 7");
    });
    const asked = (fetchMock.mock.calls as unknown as [RequestInfo, RequestInit][]).map(
      ([, init]) => JSON.parse(init.body as string).variables.input,
    );
    expect(asked.some((input) => input.ids)).toBe(false);
  });

  it("offers nothing new when an option already has the typed value in another case", async () => {
    setCsrfToken("graphql-token");
    const user = userEvent.setup();
    stubSourceSearch(["Chase", "Wells Fargo"]);
    renderSourcePicker({ onOptionChange: vi.fn(), createOption: sourceFromText });

    await user.click(screen.getByRole("combobox"));
    await user.type(screen.getByPlaceholderText("Search source..."), "chase");

    expect(await screen.findByText("Chase")).toBeInTheDocument();
    expect(screen.queryByText('Use "chase"')).not.toBeInTheDocument();
  });

  it("keeps to the listed options when the picker takes no new values", async () => {
    setCsrfToken("graphql-token");
    const user = userEvent.setup();
    stubSourceSearch(["Chase"]);
    renderSourcePicker({ onOptionChange: vi.fn() });

    await user.click(screen.getByRole("combobox"));
    await user.type(screen.getByPlaceholderText("Search source..."), "Lockbox 7");

    expect(await screen.findByText("No results found.")).toBeInTheDocument();
    expect(screen.queryByText('Use "Lockbox 7"')).not.toBeInTheDocument();
  });
});
