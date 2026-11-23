import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { worker as workerQueries } from "@/lib/queries/worker";
import { WorkerPage, prefetch } from "../page";

const mocks = vi.hoisted(() => ({
  fetchRecord: vi.fn(),
}));

vi.mock("@tanstack/react-router", () => ({ useParams: () => ({ id: "wrk_1" }) }));
vi.mock("@/hooks/use-breadcrumb-label", () => ({ useBreadcrumbLabel: () => undefined }));
vi.mock("@/components/navigation/sidebar-layout", () => ({
  PageLayout: ({
    children,
    pageHeaderProps,
  }: {
    children: ReactNode;
    pageHeaderProps: { title: string };
  }) => (
    <div>
      <h1>{pageHeaderProps.title}</h1>
      {children}
    </div>
  ),
}));
vi.mock("@/lib/queries/worker", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/queries/worker")>();
  const record = (workerId: string) => ({
    queryKey: ["worker", "record", workerId],
    queryFn: () => mocks.fetchRecord(workerId),
  });
  return {
    ...actual,
    worker: { ...actual.worker, record },
    workerRecordQuery: (workerId: string) => ({ ...record(workerId), refetchOnMount: false }),
  };
});

let client: QueryClient;

beforeEach(() => {
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  mocks.fetchRecord.mockReset();
});
afterEach(cleanup);

function renderPage() {
  render(
    <QueryClientProvider client={client}>
      <WorkerPage />
    </QueryClientProvider>,
  );
}

describe("the worker page", () => {
  it("draws its own layout in shimmer while the worker is read", () => {
    mocks.fetchRecord.mockReturnValue(new Promise(() => undefined));
    renderPage();

    expect(screen.getByLabelText("Loading the worker")).toHaveAttribute("aria-busy", "true");
  });

  it("says no worker has the link when the read finds none", async () => {
    mocks.fetchRecord.mockResolvedValue(null);
    renderPage();

    expect(await screen.findByText("No worker has this link")).toBeTruthy();
    expect(
      screen.getByText("They may have been deleted, or they belong to another organization."),
    ).toBeTruthy();
  });

  it("offers to read again when the read fails, rather than claiming the worker is gone", async () => {
    mocks.fetchRecord.mockRejectedValue(new TypeError("Failed to fetch"));
    renderPage();

    expect(await screen.findByRole("button", { name: /Try again/ })).toBeTruthy();
    expect(screen.queryByText("No worker has this link")).toBeNull();
  });

  it("warms the worker and its overview for a link", () => {
    const entries = prefetch({ params: { id: "wrk_1" }, searchParams: new URLSearchParams() });

    expect(Array.isArray(entries) ? entries.map((entry) => entry.queryKey) : []).toEqual([
      workerQueries.record("wrk_1").queryKey,
      workerQueries.overview("wrk_1").queryKey,
    ]);
  });
});
