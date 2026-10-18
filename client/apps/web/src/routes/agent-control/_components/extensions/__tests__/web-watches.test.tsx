import type { WebWatch } from "@/types/agent-extension";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, describe, expect, it, vi } from "vitest";
import { WebWatches } from "../web-watches";

const mocks = vi.hoisted(() => ({
  listWebWatches: vi.fn(),
  webWatchFindings: vi.fn(),
  stopWebWatch: vi.fn(),
}));

vi.mock("@/services/api", () => ({ apiService: { agentExtensionService: mocks } }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function watch(overrides: Partial<WebWatch> = {}): WebWatch {
  return {
    id: "awwt_1",
    extensionType: "Exa",
    query: "FMCSA final rule broker transparency 49 CFR 371.3",
    purpose: "FMCSA finalizing the broker transparency rule",
    sites: ["fmcsa.dot.gov"],
    cadence: "Weekly",
    status: "Active",
    endsAt: 1_792_742_400,
    lastCheckedAt: 1_790_160_000,
    lastRunAt: 1_790_155_000,
    runCount: 2,
    findingCount: 1,
    costUsd: 0.03,
    failure: "",
    agentDefinitionId: "",
    agentRunId: "",
    threadId: "athr_1",
    requestedById: "usr_1",
    stoppedById: "",
    finishedAt: null,
    createdAt: 1_790_150_400,
    ...overrides,
  };
}

function renderWatches(canUpdate: boolean, search = "") {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <NuqsTestingAdapter searchParams={search}>
      <QueryClientProvider client={client}>
        <WebWatches canUpdate={canUpdate} />
      </QueryClientProvider>
    </NuqsTestingAdapter>,
  );
}

describe("WebWatches", () => {
  it("opens the watch a link names, with what it searches for and what it found", async () => {
    mocks.listWebWatches.mockResolvedValue({ items: [watch()], nextBefore: 0 });
    mocks.webWatchFindings.mockResolvedValue({
      items: [
        {
          id: "awwf_1",
          summary: "FMCSA published the final rule.",
          results: [
            {
              title: "Broker transparency final rule",
              url: "https://www.federalregister.gov/d/2026-1",
              publishedDate: "2026-10-01",
              excerpt: "",
            },
          ],
          foundAt: 1_790_155_030,
          deliveredAt: null,
        },
      ],
    });

    renderWatches(false, "?webWatch=awwt_1");

    expect(await screen.findByText("FMCSA finalizing the broker transparency rule")).toBeTruthy();
    expect(screen.getByText("Watching")).toBeTruthy();
    expect(screen.getByText("1 finding")).toBeTruthy();
    expect(screen.getByText("FMCSA final rule broker transparency 49 CFR 371.3")).toBeTruthy();
    expect(screen.getByText("fmcsa.dot.gov")).toBeTruthy();
    expect(await screen.findByText("FMCSA published the final rule.")).toBeTruthy();
    const link = screen.getByRole("link", { name: "Broker transparency final rule" });
    expect(link.getAttribute("href")).toBe("https://www.federalregister.gov/d/2026-1");
    expect(link.getAttribute("rel")).toBe("noopener noreferrer");
    expect(screen.getByText("Not yet passed on")).toBeTruthy();
    expect(mocks.webWatchFindings).toHaveBeenCalledWith("awwt_1", expect.anything());
    expect(screen.queryByRole("button", { name: "Stop watching" })).toBeNull();
  });

  it("stops a watch still looking for someone who can update extensions", async () => {
    mocks.listWebWatches.mockResolvedValue({ items: [watch()], nextBefore: 0 });
    mocks.webWatchFindings.mockResolvedValue({ items: [] });
    mocks.stopWebWatch.mockResolvedValue(watch({ status: "Stopped" }));

    renderWatches(true, "?webWatch=awwt_1");

    expect(
      await screen.findByText("Nothing new yet. Each check that finds something is kept here."),
    ).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Stop watching" }));

    await waitFor(() => expect(mocks.stopWebWatch).toHaveBeenCalledWith("awwt_1"));
  });

  it("offers no stop for a watch that ended", async () => {
    mocks.listWebWatches.mockResolvedValue({
      items: [watch({ status: "Ended", finishedAt: 1_792_742_400 })],
      nextBefore: 0,
    });
    mocks.webWatchFindings.mockResolvedValue({ items: [] });

    renderWatches(true, "?webWatch=awwt_1");

    expect(await screen.findAllByText("Ended")).toHaveLength(2);
    expect(screen.queryByRole("button", { name: "Stop watching" })).toBeNull();
  });

  it("says when there are no watches yet", async () => {
    mocks.listWebWatches.mockResolvedValue({ items: [], nextBefore: 0 });

    renderWatches(true);

    expect(
      await screen.findByText(
        "No web watches yet. When a person asks an agent to keep an eye on something, the watch is listed here with what it found.",
      ),
    ).toBeTruthy();
  });
});
