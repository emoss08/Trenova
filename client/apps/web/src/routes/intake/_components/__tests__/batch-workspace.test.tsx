import type { CaptureBatchDetail, CaptureItem, CapturePage } from "@/lib/graphql/capture";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BatchWorkspace } from "../batch-workspace";

const capture = vi.hoisted(() => ({
  fetchCaptureBatch: vi.fn(),
  discardCaptureBatch: vi.fn(),
}));

vi.mock("@/lib/graphql/capture", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/capture")>()),
  fetchCaptureBatch: capture.fetchCaptureBatch,
  discardCaptureBatch: capture.discardCaptureBatch,
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));
vi.mock("@/components/elements/pdf-viewer", () => ({
  PdfViewer: () => <div data-testid="pdf" />,
}));

// Pages as CapturePageFields sends them. `sequence` is the page's place in
// the stack as scanned, 1-based (captureservice numbers from 1). The fixture
// deliberately lists them out of order and splits them across documents, so a
// number taken from an array index or a document position would disagree.
function page(id: string, sequence: number): CapturePage {
  return {
    id,
    sequence,
    status: "Ready",
    rotation: 0,
    widthPx: 850,
    heightPx: 1100,
    dpi: 200,
    isBlank: false,
    isSeparator: false,
    patchCode: "",
    isCoverSheet: false,
    unrecognizedCoverSheet: false,
    failureMessage: "",
    contentPath: `/capture/pages/${id}/content`,
    thumbnailPath: "",
  } as unknown as CapturePage;
}

function item(id: string, position: number, pageIds: string[]): CaptureItem {
  return {
    id,
    position,
    status: "Proposed",
    pageIds,
    pageCount: pageIds.length,
    suggestedType: "",
    suggestedId: null,
    suggestedDocumentTypeId: null,
    suggestionSource: null,
    suggestionConfidence: null,
    suggestionReason: "",
    detectedKind: "",
    filedType: "",
    filedId: null,
    filedDocumentTypeId: null,
    documentId: null,
    filedAt: null,
    failureMessage: "",
    version: 1,
    suggestedRecord: null,
    filedRecord: null,
    filedBy: null,
  } as unknown as CaptureItem;
}

function batch(overrides: Partial<CaptureBatchDetail> = {}): CaptureBatchDetail {
  return {
    id: "cpb_01stack",
    userId: "usr_01",
    source: "Scan",
    status: "Ready",
    sourceName: "Front desk scanner",
    jobName: "",
    targetType: "",
    targetId: null,
    receivedPageCount: 3,
    itemCount: 2,
    filedItemCount: 0,
    openItemCount: 2,
    failureMessage: "",
    sealedAt: 1_700_000_000,
    processedAt: 1_700_000_100,
    retainUntil: 1_900_000_000,
    isEditable: true,
    version: 4,
    createdAt: 1_700_000_000,
    updatedAt: 1_700_000_100,
    user: null,
    device: null,
    target: null,
    documentTypeId: null,
    settings: {
      protocol: null,
      dpi: 200,
      pixelType: null,
      duplex: false,
      feeder: true,
      driverVersion: "",
      application: "",
      refused: [],
    },
    pages: [page("pg_c", 3), page("pg_a", 1), page("pg_b", 2)],
    items: [item("cpi_02", 2, ["pg_c"]), item("cpi_01", 1, ["pg_a", "pg_b"])],
    ...overrides,
  } as unknown as CaptureBatchDetail;
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

function renderWorkspace(onClose: () => void = () => {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <BatchWorkspace batchId="cpb_01stack" now={1_700_000_200} onClose={onClose} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

async function openDiscardStack(user: ReturnType<typeof userEvent.setup>) {
  await user.click(await screen.findByRole("button", { name: "Stack actions" }));
  await user.click(await screen.findByRole("menuitem", { name: /Discard the stack/ }));
  return screen.findByRole("alertdialog");
}

describe("BatchWorkspace", () => {
  beforeEach(() => {
    capture.fetchCaptureBatch.mockResolvedValue(batch());
  });
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("keeps the discard-stack dialog open until the discard settles and shows why it failed", async () => {
    const user = userEvent.setup();
    const pending = deferred<never>();
    capture.discardCaptureBatch.mockReturnValue(pending.promise);
    renderWorkspace();

    const dialog = await openDiscardStack(user);
    await user.click(within(dialog).getByRole("button", { name: "Discard stack" }));

    expect(capture.discardCaptureBatch).toHaveBeenCalledWith("cpb_01stack", 4);
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Keep it" })).toBeDisabled();

    pending.reject(
      new GraphQLRequestError({
        kind: "graphql",
        message: "Somebody else changed this stack; reload it and try again",
        graphQLErrors: [
          {
            message: "Somebody else changed this stack; reload it and try again",
            extensions: {},
            type: "business-rule-violation",
          },
        ],
        status: 200,
      }),
    );

    const alert = await within(dialog).findByRole("alert");
    expect(alert).toHaveTextContent("The stack was not discarded");
    expect(alert).toHaveTextContent("Somebody else changed this stack; reload it and try again");
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });

  it("closes the discard-stack dialog once the discard succeeds", async () => {
    const user = userEvent.setup();
    capture.discardCaptureBatch.mockResolvedValue({ ...batch(), status: "Discarded" });
    renderWorkspace();

    const dialog = await openDiscardStack(user);
    await user.click(within(dialog).getByRole("button", { name: "Discard stack" }));

    await waitFor(() => expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument());
  });

  it("titles a page's preview with the number its thumbnail shows", async () => {
    const user = userEvent.setup();
    renderWorkspace();

    for (const number of [1, 2, 3]) {
      await user.click(await screen.findByRole("button", { name: `Page ${number} actions` }));
      await user.click(await screen.findByRole("menuitem", { name: /Preview/ }));
      const dialog = await screen.findByRole("dialog");
      expect(within(dialog).getByRole("heading", { name: `Page ${number}` })).toBeInTheDocument();
      await user.keyboard("{Escape}");
      await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    }
  });

  it("offers each page only what it can do, from the one menu the stack shares", async () => {
    const user = userEvent.setup();
    renderWorkspace();

    await user.click(await screen.findByRole("button", { name: "Page 1 actions" }));
    expect(
      await screen.findByRole("menuitem", { name: /Start a new document after this page/ }),
    ).toBeInTheDocument();
    expect(screen.getAllByRole("menuitem", { name: /Set aside/ })).toHaveLength(1);
    await user.click(screen.getByRole("menuitem", { name: /Move to/ }));
    expect(await screen.findByRole("menuitem", { name: "Document 2" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Document 1" })).not.toBeInTheDocument();
    await user.keyboard("{Escape}{Escape}");
    await waitFor(() => expect(screen.queryByRole("menu")).not.toBeInTheDocument());

    await user.click(screen.getByRole("button", { name: "Page 2 actions" }));
    expect(await screen.findByRole("menuitem", { name: /Preview/ })).toBeInTheDocument();
    expect(
      screen.queryByRole("menuitem", { name: /Start a new document after this page/ }),
    ).not.toBeInTheDocument();
    await user.click(screen.getByRole("menuitem", { name: /Set aside/ }));

    await user.click(await screen.findByRole("button", { name: "Page 2 actions" }));
    expect(
      await screen.findByRole("menuitem", { name: /Make it a document of its own/ }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: /Set aside/ })).not.toBeInTheDocument();
  });

  it("shows the pages of a stack still arriving as received, not set aside", async () => {
    capture.fetchCaptureBatch.mockResolvedValue(
      batch({ status: "Receiving", isEditable: false, items: [], itemCount: 0, openItemCount: 0 }),
    );
    renderWorkspace();

    expect(await screen.findByText("Pages received")).toBeInTheDocument();
    expect(screen.queryByText("Set aside")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Page 3\./ })).toBeInTheDocument();
  });

  it("says a stack that no longer exists is gone and leads back to the queue", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    capture.fetchCaptureBatch.mockRejectedValue(
      new GraphQLRequestError({
        kind: "graphql",
        message: "Capture batch not found",
        graphQLErrors: [
          { message: "Capture batch not found", extensions: {}, type: "resource-not-found" },
        ],
        status: 200,
      }),
    );
    renderWorkspace(onClose);

    expect(await screen.findByText("This stack is gone")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Try again/ })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Back to the queue" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("offers to try again when the stack could not be loaded", async () => {
    const user = userEvent.setup();
    capture.fetchCaptureBatch.mockRejectedValueOnce(new TypeError("Failed to fetch"));
    renderWorkspace();

    expect(await screen.findByText("Could not load this stack")).toBeInTheDocument();
    expect(screen.queryByText("This stack is gone")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Try again/ }));

    expect(await screen.findByRole("button", { name: "Stack actions" })).toBeInTheDocument();
    expect(capture.fetchCaptureBatch).toHaveBeenCalledTimes(2);
  });

  it("says beside the button why there is nothing to file yet", async () => {
    renderWorkspace();

    expect(
      await screen.findByText("Choose a record for each document you want to file"),
    ).toBeVisible();
    expect(screen.getByRole("button", { name: "File documents" })).toBeDisabled();
  });
});
