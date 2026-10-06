import { useDeskSettingsStore } from "@/stores/desk-settings-store";
import type { DeskSearchResult } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeskSearchPalette } from "../desk-search";

const search = vi.hoisted(() => ({
  calls: [] as Array<[string, string]>,
  results: (_query: string, _kind: string): unknown[] => [],
}));

vi.mock("@/lib/queries", () => ({
  queries: {
    assistant: {
      deskSearch: (query: string, kind: string) => ({
        queryKey: ["assistant", "deskSearch", query, kind],
        queryFn: () => {
          search.calls.push([query, kind]);
          return search.results(query, kind);
        },
      }),
    },
  },
}));

vi.mock("@trenova/shared/hooks/use-debounce", () => ({ useDebounce: <T,>(value: T) => value }));

function result(overrides: Partial<DeskSearchResult> & Pick<DeskSearchResult, "kind" | "id">) {
  return {
    threadId: "athr_1",
    agentId: "",
    title: "",
    threadTitle: "Storm loads",
    artifactKind: "",
    status: "",
    at: 0,
    ...overrides,
  };
}

function Where() {
  const location = useLocation();
  return <output data-testid="where">{location.pathname + location.search}</output>;
}

function renderPalette() {
  const onClose = vi.fn();
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/desk"]}>
        <Routes>
          <Route
            path="*"
            element={
              <>
                <DeskSearchPalette agentsById={new Map()} onClose={onClose} />
                <Where />
              </>
            }
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { onClose };
}

describe("DeskSearchPalette", () => {
  beforeEach(() => {
    search.calls = [];
    useDeskSettingsStore.setState({ recentSearches: ["storm warning"] });
  });

  it("offers what was recent and what was searched before", async () => {
    search.results = () => [
      result({ kind: "chat", id: "athr_1", title: "Storm loads" }),
      result({ kind: "art", id: "aart_1", title: "Workers", artifactKind: "table_view" }),
    ];
    renderPalette();

    expect(await screen.findByText("Recent conversations")).toBeInTheDocument();
    expect(screen.getByText("Recent artifacts")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /storm warning/ })).toBeInTheDocument();
    expect(search.calls[0]).toEqual(["", "all"]);
  });

  it("opens the artifact picked with the keyboard and remembers the search", async () => {
    search.results = (query) =>
      query === ""
        ? []
        : [
            result({ kind: "chat", id: "athr_1", title: "Storm loads" }),
            result({ kind: "art", id: "aart_9", title: "Workers", artifactKind: "table_view" }),
          ];
    renderPalette();

    const input = screen.getByRole("combobox", { name: "Search the Desk" });
    fireEvent.change(input, { target: { value: "work" } });
    await screen.findByText("2 results");
    fireEvent.keyDown(input, { key: "ArrowDown" });
    fireEvent.keyDown(input, { key: "Enter" });

    await waitFor(() =>
      expect(screen.getByTestId("where")).toHaveTextContent("?a=aart_9"),
    );
    expect(useDeskSettingsStore.getState().recentSearches[0]).toBe("work");
  });

  it("moves between filters with Tab and says when nothing matches", async () => {
    search.results = () => [];
    renderPalette();

    const input = screen.getByRole("combobox", { name: "Search the Desk" });
    fireEvent.keyDown(input, { key: "Tab" });
    expect(screen.getByRole("tab", { name: "Chats" })).toHaveAttribute("aria-selected", "true");

    fireEvent.change(input, { target: { value: "zzz" } });
    expect(await screen.findByText("Nothing matches “zzz”")).toBeInTheDocument();
    expect(search.calls.at(-1)).toEqual(["zzz", "chat"]);
  });
});
