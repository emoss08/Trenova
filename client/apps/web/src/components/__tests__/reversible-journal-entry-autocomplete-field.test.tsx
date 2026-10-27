import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ReversibleJournalEntryAutocompleteField } from "../autocomplete-fields";

const CURSOR = "eyJpZCI6ImplXzEifQ";

type Values = { originalJournalEntryId: string };

type JournalEntryOption = {
  id: string;
  label: string;
  description: string | null;
  meta: { entryType: string; status: string; accountingDate: number };
};

// Shaped like journalEntrySelectOption on the server: the label is the entry
// number and meta carries only reference-view fields, with no description.
const POSTED_ENTRY: JournalEntryOption = {
  id: "je_01JD8Q3X",
  label: "JE-2026-000418",
  description: null,
  meta: { entryType: "Standard", status: "Posted", accountingDate: 1_792_065_600 },
};

const OLDER_ENTRY: JournalEntryOption = {
  id: "je_01JC2M7R",
  label: "JE-2026-000377",
  description: null,
  meta: { entryType: "Adjusting", status: "Posted", accountingDate: 1_791_460_800 },
};

function selectOptionsResponse(nodes: JournalEntryOption[]) {
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

let chosen: string | undefined;

function Harness({ defaultValue }: { defaultValue: string }) {
  const form = useForm<Values>({ defaultValues: { originalJournalEntryId: defaultValue } });
  const value = useWatch({ control: form.control, name: "originalJournalEntryId" });
  useEffect(() => {
    chosen = value;
  }, [value]);

  return (
    <ReversibleJournalEntryAutocompleteField<Values>
      control={form.control}
      name="originalJournalEntryId"
      label="Original journal entry"
      placeholder="Search by entry number"
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

function requestInputs(fetchMock: ReturnType<typeof vi.fn>) {
  return fetchMock.mock.calls.map(([, init]) => {
    const body = JSON.parse(String((init as RequestInit).body)) as {
      variables: { input: Record<string, unknown> };
    };
    return body.variables.input;
  });
}

describe("ReversibleJournalEntryAutocompleteField", () => {
  afterEach(() => {
    chosen = undefined;
    clearCsrfToken();
    cleanup();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("asks only for journal entries that can still be reversed", async () => {
    setCsrfToken("graphql-token");
    const user = userEvent.setup();
    const fetchMock = vi.fn(async () => selectOptionsResponse([POSTED_ENTRY, OLDER_ENTRY]));
    vi.stubGlobal("fetch", fetchMock);

    renderField();
    await user.click(screen.getByRole("combobox"));
    await screen.findByText("JE-2026-000418");

    const inputs = requestInputs(fetchMock);
    expect(inputs.length).toBeGreaterThan(0);
    for (const input of inputs) {
      expect(input).toMatchObject({ resource: "JOURNAL_ENTRY", filters: { reversible: true } });
    }
  });

  it("lists each entry by its number with its accounting date, never its id", async () => {
    setCsrfToken("graphql-token");
    const user = userEvent.setup();
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => selectOptionsResponse([POSTED_ENTRY, OLDER_ENTRY])),
    );

    renderField();
    await user.click(screen.getByRole("combobox"));

    const row = (await screen.findByText("JE-2026-000418")).closest("[role=option]") as HTMLElement;
    expect(within(row).getByText("Oct 15, 2026")).toBeTruthy();
    expect(screen.getByText("JE-2026-000377")).toBeTruthy();
    expect(screen.queryByText(/je_01/)).toBeNull();
  });

  it("stores the chosen entry's id while the field shows its number", async () => {
    setCsrfToken("graphql-token");
    const user = userEvent.setup();
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => selectOptionsResponse([POSTED_ENTRY, OLDER_ENTRY])),
    );

    renderField();
    await user.click(screen.getByRole("combobox"));
    await user.click(await screen.findByText("JE-2026-000377"));

    await waitFor(() => {
      expect(chosen).toBe("je_01JC2M7R");
    });
    const field = screen.getByRole("combobox");
    expect(field).toHaveTextContent("JE-2026-000377");
    expect(field).not.toHaveTextContent("je_01JC2M7R");
  });
});
