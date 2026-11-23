import { WorkPaneHost, WorkPaneProvider } from "@/components/work-pane";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerActivity from "../worker-activity";

const mocks = vi.hoisted(() => ({ fetchWorkerActivity: vi.fn() }));

vi.mock("@/lib/graphql/worker-activity", () => ({
  fetchWorkerActivity: mocks.fetchWorkerActivity,
  WORKER_ACTIVITY_KEY: "worker-activity",
}));

vi.mock("@/routes/worker/worker-areas", () => ({
  useWorkerAreas: () => ({ isDrawable: () => true }),
}));

vi.mock("@/hooks/use-user-timezone", () => ({ useUserTimezone: () => "UTC" }));

vi.mock("@trenova/shared/stores/auth-store", () => ({
  useAuthStore: (select: (state: { user: { id: string } }) => unknown) =>
    select({ user: { id: "usr_me" } }),
}));

const page = {
  rows: [
    {
      id: "ae_1",
      timestamp: Date.UTC(2025, 10, 20, 14, 41) / 1000,
      resource: "profile_change_request",
      resourceId: "pcr_1",
      operation: "create",
      entityRef: null,
      comment: null,
      channel: "portal",
      principalType: "user",
      principalId: "usr_maria",
      actorName: "Maria Alvarez",
      changes: null,
      subjectResource: "worker",
      subjectId: "wrk_1",
      user: { id: "usr_maria", name: "Maria Alvarez" },
    },
    {
      id: "ae_2",
      timestamp: Date.UTC(2025, 2, 4, 8, 0) / 1000,
      resource: "worker",
      resourceId: "wrk_1",
      operation: "update",
      entityRef: "Maria Alvarez",
      comment: null,
      channel: "web",
      principalType: "user",
      principalId: "usr_2",
      actorName: null,
      changes: { fleetCode: { from: "CHI-LOCAL", to: "MW-OTR", type: "updated" } },
      subjectResource: "worker",
      subjectId: "wrk_1",
      user: { id: "usr_2", name: "Marcus Bell" },
    },
  ],
  total: 2,
  endCursor: null,
  hasNextPage: false,
};

function renderActivity() {
  const onOpen = vi.fn();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <WorkPaneProvider>
        <WorkPaneHost>
          <WorkerActivity workerId="wrk_1" onOpen={onOpen} />
        </WorkPaneHost>
      </WorkPaneProvider>
    </QueryClientProvider>,
  );
  return onOpen;
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("WorkerActivity", () => {
  it("reads the worker's whole file, newest first, leaving the filters to the server", async () => {
    mocks.fetchWorkerActivity.mockResolvedValue(page);
    renderActivity();

    await screen.findByTestId("activity-ae_1");
    expect(mocks.fetchWorkerActivity).toHaveBeenCalledWith(
      "wrk_1",
      { first: 30, after: null },
      { query: "", fieldFilters: [] },
      { signal: expect.any(AbortSignal) },
    );
    expect(screen.getByText("2 changes")).toBeTruthy();
  });

  it("groups entries by day and says where each came from and what it changed", async () => {
    mocks.fetchWorkerActivity.mockResolvedValue(page);
    renderActivity();

    const fromDash = within(await screen.findByTestId("activity-ae_1"));
    expect(fromDash.getByText("Maria Alvarez")).toBeTruthy();
    expect(fromDash.getByText("Dash")).toBeTruthy();

    const fromWeb = within(screen.getByTestId("activity-ae_2"));
    expect(fromWeb.getByText("Web")).toBeTruthy();
    expect(fromWeb.getByText("CHI-LOCAL")).toBeTruthy();
    expect(fromWeb.getByText("MW-OTR")).toBeTruthy();
    expect(screen.getByRole("region", { name: "Tuesday, Mar 4, 2025" })).toBeTruthy();
  });

  it("opens the record an entry is about on this page", async () => {
    mocks.fetchWorkerActivity.mockResolvedValue(page);
    const onOpen = renderActivity();

    await userEvent.setup().click(await screen.findByTestId("activity-ae_1"));
    expect(onOpen).toHaveBeenCalledWith({ tab: "files", view: "driver-portal", request: "pcr_1" });
  });

  it("searches on the server and offers to clear the filters when nothing matches", async () => {
    mocks.fetchWorkerActivity.mockResolvedValueOnce(page).mockResolvedValue({
      rows: [],
      total: 0,
      endCursor: null,
      hasNextPage: false,
    });
    renderActivity();
    const user = userEvent.setup();

    await user.type(await screen.findByRole("textbox", { name: "Search changes" }), "hazmat");
    await waitFor(() =>
      expect(mocks.fetchWorkerActivity).toHaveBeenLastCalledWith(
        "wrk_1",
        { first: 30, after: null },
        { query: "hazmat", fieldFilters: [] },
        { signal: expect.any(AbortSignal) },
      ),
    );
    expect(await screen.findByText("No changes match.")).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Clear the filters" }));
    expect(
      (screen.getByRole("textbox", { name: "Search changes" }) as HTMLInputElement).value,
    ).toBe("");
  });

  it("says nothing has happened yet on an empty file", async () => {
    mocks.fetchWorkerActivity.mockResolvedValue({
      rows: [],
      total: 0,
      endCursor: null,
      hasNextPage: false,
    });
    renderActivity();

    expect(await screen.findByText("No activity yet")).toBeTruthy();
  });

  it("says the trail could not be loaded when the read fails", async () => {
    mocks.fetchWorkerActivity.mockRejectedValue(new Error("boom"));
    renderActivity();

    expect(await screen.findByText("The activity could not be loaded")).toBeTruthy();
  });
});
