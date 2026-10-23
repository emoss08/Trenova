import type { RateMatrixRecord } from "@/lib/graphql/rate-matrix";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { useState } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { matrixActions, RateMatrixVersions } from "../rate-matrix-versions";

const fetchRateMatrixRecord = vi.fn();
const approve = vi.fn();
const archive = vi.fn();

vi.mock("@/lib/graphql/rate-matrix", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/rate-matrix")>()),
  fetchRateMatrixRecord: (...args: unknown[]) => fetchRateMatrixRecord(...args),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    get rateMatrixService() {
      return { approve, archive, listReviewers: vi.fn().mockResolvedValue([]) };
    },
  },
}));

const USER = { id: "usr_1", name: "Ada Reviewer", username: "ada", profilePicUrl: null };

function record(overrides: Partial<RateMatrixRecord> = {}): RateMatrixRecord {
  return {
    id: "rmx_1",
    code: "LANES",
    name: "Lane tariff",
    status: "Active",
    publishedVersionNumber: 2,
    publishedAt: 1_700_000_000,
    hasDraftChanges: false,
    cellCount: 4,
    submittedById: null,
    submittedAt: null,
    approvedById: null,
    approvedAt: null,
    reviewComment: "",
    version: 5,
    updatedAt: 1_700_000_000,
    versions: [
      {
        id: "rmv_2",
        rateMatrixId: "rmx_1",
        versionNumber: 2,
        code: "LANES",
        name: "Lane tariff",
        valueType: "Money",
        cellCount: 4,
        changeMessage: "Q3 increase",
        submittedById: null,
        publishedById: USER.id,
        createdAt: 1_700_000_000,
        publishedBy: USER,
      },
      {
        id: "rmv_1",
        rateMatrixId: "rmx_1",
        versionNumber: 1,
        code: "LANES",
        name: "Lane tariff",
        valueType: "Money",
        cellCount: 3,
        changeMessage: "",
        submittedById: null,
        publishedById: USER.id,
        createdAt: 1_690_000_000,
        publishedBy: USER,
      },
    ],
    reviews: [],
    usages: [
      {
        templateId: "ft_1",
        templateName: "LTL base",
        templateStatus: "Active",
        templateVersion: 3,
        versionStatus: "Active",
        matrixVersionNumber: 1,
      },
      {
        templateId: "ft_2",
        templateName: "LTL fuel",
        templateStatus: "Active",
        templateVersion: 1,
        versionStatus: "Active",
        matrixVersionNumber: 1,
      },
    ],
    ...overrides,
  } as RateMatrixRecord;
}

function Harness() {
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false } } }),
  );
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <RateMatrixVersions rateMatrixId="rmx_1" />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe("matrixActions", () => {
  it("offers submit only for a draft that differs, or to bring back an archived matrix", () => {
    expect(
      matrixActions({ status: "Draft", hasDraftChanges: true, publishedVersionNumber: 0 }),
    ).toEqual({ submit: true, decide: false, archive: false });
    expect(
      matrixActions({ status: "Active", hasDraftChanges: false, publishedVersionNumber: 2 }),
    ).toEqual({ submit: false, decide: false, archive: true });
    expect(
      matrixActions({ status: "Inactive", hasDraftChanges: false, publishedVersionNumber: 2 }),
    ).toEqual({ submit: true, decide: false, archive: false });
  });

  it("offers only a decision while the draft is in review", () => {
    expect(
      matrixActions({ status: "InReview", hasDraftChanges: true, publishedVersionNumber: 2 }),
    ).toEqual({ submit: false, decide: true, archive: false });
  });
});

describe("RateMatrixVersions", () => {
  beforeEach(() => {
    fetchRateMatrixRecord.mockReset();
    approve.mockReset();
    archive.mockReset();
  });

  it("lists the published versions newest first with what pins each, and who pins them", async () => {
    fetchRateMatrixRecord.mockResolvedValue(record({ hasDraftChanges: true, status: "Draft" }));

    render(<Harness />);

    expect(
      await screen.findByText(
        "Tariffs read version 2. The draft has changes that are not published.",
      ),
    ).toBeInTheDocument();
    const versions = screen.getByRole("region", { name: "Published versions" });
    const rows = within(versions).getAllByRole("listitem");
    expect(rows.map((row) => within(row).getByText(/^v\d$/).textContent)).toEqual(["v2", "v1"]);
    expect(within(rows[0]!).getByText("Lane tariff (tariffs read this one)")).toBeInTheDocument();
    expect(within(rows[1]!).getByText("2 template versions pin it")).toBeInTheDocument();

    const pinned = screen.getByRole("region", { name: "Pinned by" });
    expect(within(pinned).getByRole("link", { name: "LTL base" })).toHaveAttribute(
      "href",
      "/billing/configuration-files/formula-templates/ft_1/edit",
    );
    expect(within(pinned).getAllByText("pins v1")).toHaveLength(2);
    expect(screen.getByRole("button", { name: "Submit for review" })).toBeInTheDocument();
  });

  it("publishes a draft in review from the reviewer's decision", async () => {
    fetchRateMatrixRecord.mockResolvedValue(record({ status: "InReview", hasDraftChanges: true }));
    approve.mockResolvedValue({});

    render(<Harness />);

    fireEvent.click(await screen.findByRole("button", { name: "Publish" }));
    expect(screen.queryByRole("button", { name: "Submit for review" })).toBeNull();
    const dialog = await screen.findByRole("dialog");
    fireEvent.change(within(dialog).getByLabelText("Comment (optional)"), {
      target: { value: "Looks right" },
    });
    fireEvent.click(within(dialog).getByRole("button", { name: "Publish" }));

    await waitFor(() => expect(approve).toHaveBeenCalledWith("rmx_1", "Looks right"));
  });

  it("asks before archiving a matrix template versions pin, and says so to the server", async () => {
    fetchRateMatrixRecord.mockResolvedValue(record());
    archive.mockResolvedValue({});

    render(<Harness />);

    fireEvent.click(await screen.findByRole("button", { name: "Archive" }));
    const dialog = await screen.findByRole("dialog");
    const confirm = within(dialog).getByRole("button", { name: "Archive" });
    expect(confirm).toBeDisabled();

    fireEvent.click(
      within(dialog).getByRole("checkbox", {
        name: "2 template versions pin a version of this matrix. They keep reading the version they pinned. Archive it anyway.",
      }),
    );
    fireEvent.click(confirm);

    await waitFor(() => expect(archive).toHaveBeenCalledWith("rmx_1", true));
  });

  it("says a matrix was never published until a reviewer publishes it", async () => {
    fetchRateMatrixRecord.mockResolvedValue(
      record({
        status: "Draft",
        publishedVersionNumber: 0,
        hasDraftChanges: true,
        versions: [],
        usages: [],
      }),
    );

    render(<Harness />);

    expect(
      await screen.findByText(
        "Never published. Tariffs cannot read it until a reviewer publishes the draft.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("No version has been published yet.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Archive" })).toBeNull();
  });
});
