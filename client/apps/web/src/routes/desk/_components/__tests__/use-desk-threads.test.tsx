"use no memo";
import { threadListQuery, threadsOf, updateCachedThread } from "@/lib/thread-list";
import type { AssistantThread } from "@/types/assistant";
import { QueryClient, QueryClientProvider, useInfiniteQuery } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useDeskThreads } from "../use-desk-threads";

const listThreads = vi.hoisted(() =>
  vi.fn<(options: { cursor?: string; until?: string }) => Promise<unknown>>(),
);

vi.mock("@/services/api", () => ({
  apiService: { assistantService: { listThreads } },
}));

function thread(id: string, title = id): AssistantThread {
  return {
    id,
    businessUnitId: "bu",
    organizationId: "org",
    userId: "u",
    agentDefinitionId: "agtd_1",
    preferredProviderId: "",
    origin: "Desk",
    pinned: false,
    subjectType: "",
    subjectId: "",
    canContinue: true,
    title,
    status: "Active",
    lastMessageAt: 100,
    version: 0,
    createdAt: 50,
    updatedAt: 100,
  };
}

const renders = { frame: 0 };
const rail: { loadMore: () => Promise<unknown> } = { loadMore: () => Promise.resolve() };

/** Stands in for the Desk's frame: it reads only what the hook selects. */
function Frame({ activeThreadId }: { activeThreadId: string | null }) {
  renders.frame += 1;
  const { threads, listedThread } = useDeskThreads(activeThreadId);
  return (
    <p data-testid="frame">
      {listedThread?.title ?? "none"} · {threads.length} recent
    </p>
  );
}

/** Stands in for the rail: it reads the whole list and asks for the next page. */
function Rail() {
  const query = useInfiniteQuery(threadListQuery());
  rail.loadMore = query.fetchNextPage;
  return <p data-testid="rail">{threadsOf(query.data).length} listed</p>;
}

let queryClient: QueryClient;

beforeEach(() => {
  renders.frame = 0;
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Number.POSITIVE_INFINITY } },
  });
  listThreads.mockReset();
  listThreads.mockImplementation(({ cursor = "" }) =>
    Promise.resolve(
      cursor === ""
        ? { items: [thread("a", "Open one"), thread("b")], nextCursor: "b" }
        : { items: [thread("c"), thread("d", "Deep one")], nextCursor: "" },
    ),
  );
});

function renderDesk(activeThreadId: string | null) {
  return render(
    <QueryClientProvider client={queryClient}>
      <Frame activeThreadId={activeThreadId} />
      <Rail />
    </QueryClientProvider>,
  );
}

/**
 * The rail reads older conversations a page at a time. The Desk's frame, and
 * every page under it, read only the open conversation and the first page,
 * so a page arriving re-renders the rail and nothing else.
 */
describe("useDeskThreads", () => {
  it("leaves the frame alone when the rail reads another page", async () => {
    renderDesk("a");
    expect(await screen.findByText("2 listed")).toBeInTheDocument();
    expect(screen.getByTestId("frame")).toHaveTextContent("Open one · 2 recent");
    const before = renders.frame;

    await act(() => rail.loadMore());

    expect(screen.getByText("4 listed")).toBeInTheDocument();
    expect(renders.frame).toBe(before);
  });

  it("finds an open conversation on a page the rail reads later", async () => {
    renderDesk("d");
    expect(await screen.findByText("2 listed")).toBeInTheDocument();
    expect(screen.getByTestId("frame")).toHaveTextContent("none · 2 recent");

    await act(() => rail.loadMore());

    expect(screen.getByTestId("frame")).toHaveTextContent("Deep one · 2 recent");
  });

  it("follows a change to the open conversation", async () => {
    renderDesk("a");
    expect(await screen.findByText("2 listed")).toBeInTheDocument();

    act(() => updateCachedThread(queryClient, "a", (item) => ({ ...item, title: "Renamed" })));

    expect(screen.getByTestId("frame")).toHaveTextContent("Renamed · 2 recent");
  });
});
