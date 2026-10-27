import type { ArtifactRows, AssistantArtifact } from "@/types/assistant";
import type { ArtifactRowsRequest } from "@/services/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskTableBody } from "../desk-table-body";

type Commodity = { id: string; name: string; status: string; rate: number };

const list = vi.hoisted(() => ({
  universe: [] as Commodity[],
  searchable: true,
  sortable: ["name"] as string[],
  withheld: [] as string[],
  failures: [] as unknown[],
  reads: [] as ArtifactRowsRequest[],
  estimatedTotal: null as number | null,
}));

vi.mock("@/services/api", () => ({
  apiService: {
    assistantService: {
      artifactRows: async (
        _threadId: string,
        _artifactId: string,
        read: ArtifactRowsRequest,
      ): Promise<ArtifactRows> => {
        list.reads.push(read);
        const failure = list.failures.shift();
        if (failure !== undefined) {
          throw failure;
        }
        let all = list.universe.filter((row) =>
          read.q ? row.name.toLowerCase().includes(read.q.toLowerCase()) : true,
        );
        if (read.sortBy === "name") {
          all = [...all].sort(
            (a, b) => a.name.localeCompare(b.name) * (read.sortDirection === "desc" ? -1 : 1),
          );
        }
        const start = read.after ? Number(read.after.slice(2)) : (read.offset ?? 0);
        const page = all.slice(start, start + read.limit);
        const end = start + page.length;
        return {
          rows: page,
          hasMore: end < all.length,
          nextCursor: end < all.length ? `c-${end}` : "",
          nextOffset: null,
          total: read.total ? (list.estimatedTotal ?? all.length) : null,
          totalEstimated: read.total ? list.estimatedTotal !== null : false,
          withheld: list.withheld,
          searchable: list.searchable,
          sortable: list.sortable,
        };
      },
    },
  },
}));

function commodity(n: number): Commodity {
  return {
    id: `com_${n}`,
    name: `Commodity ${n}`,
    status: n % 2 === 0 ? "Inactive" : "Active",
    rate: n * 10,
  };
}

function range(from: number, to: number): Commodity[] {
  return Array.from({ length: to - from + 1 }, (_, index) => commodity(from + index));
}

function table(payload: Record<string, unknown>): AssistantArtifact {
  return {
    id: "art_1",
    threadId: "athr_1",
    messageId: "",
    runId: "",
    proposalId: "",
    planId: "",
    kind: "table_view",
    status: "Ready",
    title: "Commodities",
    payload: {
      display: 1,
      tool: "masterdata_list_commodities",
      entity: "commodities",
      columns: [
        { key: "name", label: "Name", type: "text" },
        { key: "status", label: "Status", type: "status" },
        { key: "rate", label: "Rate", type: "money" },
      ],
      searchedFor: ["status is Active or Inactive"],
      readBy: { id: "agd_dispatch", name: "Dispatch Desk", icon: "", accent: "teal" },
      readWith: { status: ["Active", "Inactive"] },
      ...payload,
    },
    sourceToolCallId: "call_1",
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    slug: "commodities",
    turn: "",
    createdAt: 1_760_000_000,
    updatedAt: 1_760_000_000,
  };
}

function agentRead(more: Record<string, unknown> = {}): AssistantArtifact {
  return table({
    rows: range(1, 25),
    rowCount: 25,
    hasMore: true,
    nextCursor: "c-25",
    pages: 1,
    ...more,
  });
}

function renderTable(shown: AssistantArtifact) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskTableBody artifact={shown} previous={null} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function pager() {
  return screen.getByRole("button", { name: "Next page" }).closest(".dk-tv-pager") as HTMLElement;
}

beforeEach(() => {
  window.localStorage.clear();
  list.universe = range(1, 60);
  list.searchable = true;
  list.sortable = ["name"];
  list.withheld = [];
  list.failures = [];
  list.reads = [];
  list.estimatedTotal = null;
});

afterEach(() => {
  window.localStorage.clear();
});

describe("DeskTableBody paging a list", () => {
  it("reads the next page as the table opens and pages on from the rows the agent read", async () => {
    renderTable(agentRead());

    expect(screen.getByText("Commodity 1")).toBeInTheDocument();
    expect(screen.getByText(/^Dispatch Desk read this page · /)).toBeInTheDocument();
    expect(await within(pager()).findByText("1–25 of 60")).toBeInTheDocument();
    expect(list.reads[0]).toEqual({
      limit: 25,
      after: "c-25",
      offset: undefined,
      q: undefined,
      sortBy: undefined,
      sortDirection: "asc",
      total: true,
    });
    expect(screen.getByText("Page 1 of 3")).toBeInTheDocument();
    expect(screen.getByText("60")).toBeInTheDocument();
    expect(screen.getByText("status is Active or Inactive")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Next page" }));
    expect(screen.getByText("Commodity 26")).toBeInTheDocument();
    expect(screen.queryByText("Commodity 25")).not.toBeInTheDocument();
    expect(screen.getByText("Read just now, with your access")).toBeInTheDocument();
    expect(list.reads).toHaveLength(1);

    fireEvent.click(screen.getByRole("button", { name: "Next page" }));
    expect(await screen.findByText("Commodity 51")).toBeInTheDocument();
    expect(list.reads[1]).toMatchObject({ limit: 25, after: "c-50", total: false });
    expect(within(pager()).getByText("51–60 of 60")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Next page" })).toBeDisabled();

    fireEvent.click(screen.getByRole("button", { name: "First page" }));
    expect(screen.getByText("Commodity 1")).toBeInTheDocument();
    expect(list.reads).toHaveLength(2);
  });

  it("shows an estimated total as approximate and pages only as far as it has read", async () => {
    list.estimatedTotal = 1_250_000;
    renderTable(agentRead());

    expect(await within(pager()).findByText("1–25 of about 1.3M")).toBeInTheDocument();
    expect(screen.getByText("about 1.3M")).toBeInTheDocument();
    expect(within(pager()).getByText("Page 1 of 3+")).toBeInTheDocument();
  });

  it("pages on from an offset when the list is read by offset", async () => {
    renderTable(agentRead({ nextCursor: undefined, nextOffset: 25 }));

    await waitFor(() => expect(list.reads).toHaveLength(1));
    expect(list.reads[0]).toMatchObject({ limit: 25, after: undefined, offset: 25, total: true });
  });

  it("searches the whole list from its start when the list takes a search", async () => {
    list.universe = [
      ...range(1, 60),
      { id: "com_steel", name: "Steel coil", status: "Active", rate: 1 },
    ];
    renderTable(agentRead());
    await within(pager()).findByText("1–25 of 61");

    fireEvent.change(screen.getByLabelText("Search the list"), { target: { value: " steel " } });

    expect(await screen.findByText("Steel coil")).toBeInTheDocument();
    expect(list.reads.at(-1)).toEqual({
      limit: 25,
      after: undefined,
      offset: undefined,
      q: "steel",
      sortBy: undefined,
      sortDirection: "asc",
      total: true,
    });
    expect(screen.getByText("row matches")).toBeInTheDocument();
    expect(screen.getByText("Searched just now")).toBeInTheDocument();
    expect(screen.queryByText("Commodity 1")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Next page" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Clear the search" }));
    expect(await screen.findByText("Commodity 1")).toBeInTheDocument();
    expect(screen.getByText(/^Dispatch Desk read this page · /)).toBeInTheDocument();
  });

  it("filters only the rows read when the list takes no search", async () => {
    list.searchable = false;
    renderTable(agentRead());
    await within(pager()).findByText("1–25 of 60");

    fireEvent.change(screen.getByLabelText("Filter rows"), { target: { value: "Commodity 4" } });

    expect(screen.getByText("Commodity 4")).toBeInTheDocument();
    expect(screen.getByText("Commodity 40")).toBeInTheDocument();
    expect(screen.queryByText("Commodity 5")).not.toBeInTheDocument();
    expect(screen.getByText("of 50 rows read")).toBeInTheDocument();
    expect(list.reads.every((read) => read.q === undefined)).toBe(true);
  });

  it("sorts the whole list by a column it sorts by, and leaves the others fixed", async () => {
    renderTable(agentRead());
    await within(pager()).findByText("1–25 of 60");

    fireEvent.click(screen.getByRole("columnheader", { name: "Status" }));
    expect(screen.getByRole("columnheader", { name: "Status" })).not.toHaveAttribute("aria-sort");
    expect(list.reads).toHaveLength(1);

    fireEvent.click(screen.getByRole("columnheader", { name: "Name" }));
    await waitFor(() => expect(list.reads).toHaveLength(2));
    expect(screen.getByRole("columnheader", { name: "Name" })).toHaveAttribute(
      "aria-sort",
      "ascending",
    );
    expect(list.reads[1]).toMatchObject({
      after: undefined,
      sortBy: "name",
      sortDirection: "asc",
      total: true,
    });
    expect(screen.getByRole("columnheader", { name: "Name" })).toHaveAttribute(
      "aria-sort",
      "ascending",
    );

    fireEvent.click(screen.getByRole("columnheader", { name: "Name" }));
    await waitFor(() => expect(list.reads.at(-1)).toMatchObject({ sortDirection: "desc" }));
  });

  it("keeps the pages read when a page fails, and reads it again", async () => {
    renderTable(agentRead());
    await within(pager()).findByText("1–25 of 60");
    list.failures.push(new Error("Failed to fetch"));

    fireEvent.click(screen.getByRole("button", { name: "Page 3" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Page 3 couldn't be read");
    expect(within(pager()).getByText("51–60 of 60")).toBeInTheDocument();
    expect(screen.queryByText("No rows")).not.toBeInTheDocument();
    expect(screen.queryByText("Read just now, with your access")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Page 2" }));
    expect(screen.getByText("Commodity 26")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Page 3" }));
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(await screen.findByText("Commodity 51")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("stops at the rows the agent read when the person may no longer read the list", async () => {
    list.failures.push(new ApiRequestError(403, { title: "Forbidden", status: 403 } as never));
    renderTable(agentRead());

    expect(
      await screen.findByText(
        "You can no longer read this list, so only the rows Dispatch Desk read are shown",
      ),
    ).toBeInTheDocument();
    expect(within(pager()).getByText("1–25 of 25+")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Next page" })).toBeDisabled();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("names the agent that read the rows on its pages and in the page strip", async () => {
    renderTable(agentRead());
    await within(pager()).findByText("1–25 of 60");

    expect(screen.getByLabelText("What Dispatch Desk asked for")).toHaveTextContent(
      "status is Active or Inactive",
    );
    expect(screen.getByRole("button", { name: "Page 1, read by Dispatch Desk" })).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(screen.getByRole("button", { name: "Page 2" })).toBeInTheDocument();
    expect(screen.queryByText(/Nova/)).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Next page" }));
    expect(screen.getByText("Read just now, with your access")).toBeInTheDocument();
  });

  it("says the rows were read in the conversation when the table does not name its agent", async () => {
    list.failures.push(new ApiRequestError(403, { title: "Forbidden", status: 403 } as never));
    renderTable(agentRead({ readBy: undefined }));

    expect(
      await screen.findByText(
        "You can no longer read this list, so only the rows read in this conversation are shown",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText(/^Read in this conversation · /)).toBeInTheDocument();
    expect(screen.getByLabelText("What was asked for")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Page 1, read in this conversation" }),
    ).toBeInTheDocument();
  });

  it("names the columns the person may not see", async () => {
    list.withheld = ["rate"];
    renderTable(agentRead());

    expect(await screen.findByText("Not shown at your access: Rate")).toBeInTheDocument();
  });

  it("reads a table cut to fit from the start once the person pages past its rows", async () => {
    renderTable(agentRead({ nextCursor: undefined, truncated: true, rowCount: 25 }));
    expect(list.reads).toHaveLength(0);

    fireEvent.click(screen.getByRole("button", { name: "Next page" }));

    expect(await screen.findByText("Commodity 26")).toBeInTheDocument();
    expect(list.reads[0]).toMatchObject({ after: undefined, offset: undefined, total: true });
  });

  it("reads the list from its start when it no longer takes the place the agent stopped at", async () => {
    list.failures.push(
      new ApiRequestError(400, {
        title: "Bad Request",
        status: 400,
        errors: [
          {
            field: "after",
            message: "This page can no longer be read. Read the list from its start",
            code: "INVALID",
          },
        ],
      } as never),
    );
    renderTable(agentRead());

    expect(await within(pager()).findByText("1–25 of 60")).toBeInTheDocument();
    expect(list.reads).toHaveLength(2);
    expect(list.reads[0]).toMatchObject({ after: "c-25", total: true });
    expect(list.reads[1]).toMatchObject({ after: undefined, offset: undefined, total: true });
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.getByText("Commodity 1")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Next page" }));
    expect(await screen.findByText("Commodity 26")).toBeInTheDocument();
    expect(list.reads[2]).toMatchObject({ after: "c-25", total: false });
  });

  it("shows fifty rows a page when asked, and remembers it", async () => {
    renderTable(agentRead());
    await within(pager()).findByText("1–25 of 60");

    fireEvent.click(screen.getByRole("button", { name: "50" }));

    expect(await screen.findByText("Commodity 50")).toBeInTheDocument();
    expect(within(pager()).getByText("1–50 of 60")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "50" })).toHaveAttribute("aria-pressed", "true");
    expect(window.localStorage.getItem("desk.table.pageSize")).toBe("50");
  });
});

describe("DeskTableBody paging a table that holds the whole list", () => {
  it("pages where it is and reads nothing again", () => {
    renderTable(table({ rows: range(1, 30), rowCount: 30, hasMore: false, pages: 2 }));

    expect(within(pager()).getByText("1–25 of 30")).toBeInTheDocument();
    expect(screen.getByText(/^Dispatch Desk read every row · /)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Next page" }));

    expect(screen.getByText("Commodity 30")).toBeInTheDocument();
    expect(within(pager()).getByText("26–30 of 30")).toBeInTheDocument();
    expect(list.reads).toHaveLength(0);
  });

  it("draws no pager for a table that fits on one page", () => {
    renderTable(table({ rows: range(1, 12), rowCount: 12 }));

    expect(screen.queryByRole("button", { name: "Next page" })).not.toBeInTheDocument();
    expect(screen.getByText("12 items")).toBeInTheDocument();
    expect(list.reads).toHaveLength(0);
  });
});

describe("DeskTableBody reading a list again", () => {
  it("reads the list again from its start in place of the agent's rows, and goes back", async () => {
    renderTable(agentRead());
    await within(pager()).findByText("1–25 of 60");
    list.universe = [{ ...commodity(1), name: "Steel, read again" }, ...range(2, 60)];

    fireEvent.click(screen.getByRole("button", { name: "Read again now" }));

    expect(await screen.findByText("Steel, read again")).toBeInTheDocument();
    expect(list.reads[1]).toEqual({
      limit: 25,
      after: undefined,
      offset: undefined,
      q: undefined,
      sortBy: undefined,
      sortDirection: "asc",
      total: true,
    });
    expect(screen.queryByText("Commodity 1")).not.toBeInTheDocument();
    expect(screen.getByText("Read just now, with your access")).toBeInTheDocument();
    expect(screen.queryByText(/^Dispatch Desk read this page · /)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /read by Dispatch Desk/ })).not.toBeInTheDocument();

    list.universe = [{ ...commodity(1), name: "Steel, read twice" }, ...range(2, 60)];
    fireEvent.click(screen.getByRole("button", { name: "Read again now" }));
    expect(await screen.findByText("Steel, read twice")).toBeInTheDocument();
    expect(list.reads[2]).toMatchObject({ after: undefined, total: true });

    expect(
      screen.getByText("What Dispatch Desk read is kept in the conversation"),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Show it" }));
    expect(screen.getByText("Commodity 1")).toBeInTheDocument();
    expect(screen.getByText(/^Dispatch Desk read this page · /)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Read again now" })).toBeInTheDocument();
    expect(
      screen.queryByText("What Dispatch Desk read is kept in the conversation"),
    ).not.toBeInTheDocument();
  });

  it("reads a table that holds its whole list again when asked", async () => {
    renderTable(table({ rows: range(1, 30), rowCount: 30, hasMore: false, pages: 2 }));
    expect(within(pager()).getByText("1–25 of 30")).toBeInTheDocument();
    expect(list.reads).toHaveLength(0);

    fireEvent.click(screen.getByRole("button", { name: "Read again now" }));

    expect(await within(pager()).findByText("1–25 of 60")).toBeInTheDocument();
    expect(list.reads[0]).toMatchObject({ after: undefined, offset: undefined, total: true });
    expect(screen.getByText("Read just now, with your access")).toBeInTheDocument();
  });

  it("offers no reading again for a table that does not say how it was read", () => {
    renderTable(agentRead({ readWith: undefined }));

    expect(screen.queryByRole("button", { name: "Read again now" })).not.toBeInTheDocument();
  });

  it("goes back to the agent's rows when the person may no longer read the list", async () => {
    renderTable(table({ rows: range(1, 30), rowCount: 30, hasMore: false, pages: 2 }));
    list.failures.push(new ApiRequestError(403, { title: "Forbidden", status: 403 } as never));

    fireEvent.click(screen.getByRole("button", { name: "Read again now" }));

    expect(
      await screen.findByText(
        "You can no longer read this list, so only the rows Dispatch Desk read are shown",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("Commodity 1")).toBeInTheDocument();
    expect(screen.getByText(/^Dispatch Desk read every row · /)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Read again now" })).not.toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
});
