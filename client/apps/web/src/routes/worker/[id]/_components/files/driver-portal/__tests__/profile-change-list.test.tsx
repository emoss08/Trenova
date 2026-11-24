import type { ProfileChangeRequestRow } from "@/lib/graphql/self-service";
import { stubLayout } from "@/test/layout";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ProfileChangeAlert } from "../../../profile/profile-change-alert";
import { ProfileChangeList } from "../profile-change-list";

const mocks = vi.hoisted(() => ({
  fetchProfileChangeRequests: vi.fn(),
  decideProfileChange: vi.fn(),
}));

vi.mock("@/lib/graphql/self-service", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/self-service")>()),
  fetchProfileChangeRequests: mocks.fetchProfileChangeRequests,
  decideProfileChange: mocks.decideProfileChange,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

const CURRENT = {
  url: "https://files.example/wrk_1/photo.jpg?sig=a",
  thumbnailUrl: "https://files.example/wrk_1/thumb.jpg?sig=a",
};
const PROPOSED = {
  url: "https://files.example/pcr_1/photo.jpg?sig=b",
  thumbnailUrl: "https://files.example/pcr_1/thumb.jpg?sig=b",
};

// Shaped after the ProfileChangeRequests operation: a photo request carries one change entry with
// no words to compare, and the pictures in `worker.photo` and `proposedPhoto`.
function photoRequest(overrides: Partial<ProfileChangeRequestRow> = {}): ProfileChangeRequestRow {
  return {
    id: "pcr_1",
    workerId: "wrk_1",
    status: "Pending",
    note: "New one from the terminal",
    submittedAt: 1_795_000_000,
    decidedAt: null,
    decisionNote: null,
    version: 1,
    changes: [{ field: "photo", label: "Photo", from: "", to: "" }],
    proposedPhoto: PROPOSED,
    worker: { id: "wrk_1", firstName: "Maria", lastName: "Alvarez", photo: CURRENT },
    ...overrides,
  } as ProfileChangeRequestRow;
}

function renderWith(node: React.ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(<QueryClientProvider client={client}>{node}</QueryClientProvider>);
}

let restoreLayout = () => {};
beforeEach(() => {
  restoreLayout = stubLayout(320);
});

afterEach(() => {
  restoreLayout();
  cleanup();
  vi.clearAllMocks();
});

describe("a photo request in the driver's profile changes", () => {
  it("shows the photo sent beside the one on the record, never the field's code", async () => {
    mocks.fetchProfileChangeRequests.mockResolvedValue([photoRequest()]);
    renderWith(<ProfileChangeList workerId="wrk_1" onReview={vi.fn()} />);

    const row = await screen.findByTestId("profile-change-pcr_1");
    // Both faces show the initials until their pictures load.
    expect(within(row).getAllByText("MA")).toHaveLength(2);
    const loading = row.querySelector(`img[src="${PROPOSED.thumbnailUrl}"]`) as HTMLImageElement;
    fireEvent.load(loading);
    const proposed = within(row).getByRole("img", { name: "Proposed photo" });
    expect(proposed.getAttribute("src")).toBe(PROPOSED.thumbnailUrl);
    expect(within(row).getByText("Now")).toBeTruthy();
    expect(within(row).getByText("Proposed")).toBeTruthy();
    expect(row.textContent).not.toContain("(blank)");
  });

  it("says when the worker has no photo yet", async () => {
    mocks.fetchProfileChangeRequests.mockResolvedValue([
      photoRequest({
        worker: { id: "wrk_1", firstName: "Maria", lastName: "Alvarez", photo: null },
      } as Partial<ProfileChangeRequestRow>),
    ]);
    renderWith(<ProfileChangeList workerId="wrk_1" onReview={vi.fn()} />);

    const row = await screen.findByTestId("profile-change-pcr_1");
    expect(within(row).getByText("No photo yet")).toBeTruthy();
  });

  it("approves the photo request as it stands", async () => {
    mocks.fetchProfileChangeRequests.mockResolvedValue([photoRequest()]);
    mocks.decideProfileChange.mockResolvedValue({ id: "pcr_1" });
    renderWith(<ProfileChangeList workerId="wrk_1" onReview={vi.fn()} />);

    const row = await screen.findByTestId("profile-change-pcr_1");
    await userEvent.click(within(row).getByRole("button", { name: "Approve" }));

    await waitFor(() =>
      expect(mocks.decideProfileChange).toHaveBeenCalledWith({ id: "pcr_1", approve: true }),
    );
  });

  it("turns the photo request down through the review pane", async () => {
    mocks.fetchProfileChangeRequests.mockResolvedValue([photoRequest()]);
    const onReview = vi.fn();
    renderWith(<ProfileChangeList workerId="wrk_1" onReview={onReview} />);

    const row = await screen.findByTestId("profile-change-pcr_1");
    await userEvent.click(within(row).getByRole("button", { name: "Turn down" }));

    expect(onReview).toHaveBeenCalledWith("pcr_1");
  });

  it("lists a decided photo request as a new photo, its picture no longer kept", async () => {
    mocks.fetchProfileChangeRequests.mockResolvedValue([
      photoRequest({ status: "Approved", decidedAt: 1_795_100_000, proposedPhoto: null }),
    ]);
    renderWith(<ProfileChangeList workerId="wrk_1" onReview={vi.fn()} />);

    const row = await screen.findByTestId("profile-change-pcr_1");
    expect(within(row).getByText("A new photo")).toBeTruthy();
    expect(within(row).queryByRole("img", { name: "Proposed photo" })).toBeNull();
  });
});

describe("the Profile alert for a photo request", () => {
  it("says the driver sent a new profile photo", async () => {
    mocks.fetchProfileChangeRequests.mockResolvedValue([photoRequest()]);
    renderWith(
      <ProfileChangeAlert workerId="wrk_1" firstName="Maria" waiting={1} onReview={vi.fn()} />,
    );

    expect(await screen.findByText(/^Maria sent a new profile photo /)).toBeTruthy();
  });
});
