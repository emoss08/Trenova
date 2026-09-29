import type { CaptureItem } from "@/lib/graphql/capture";
import { DndContext } from "@dnd-kit/core";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";
import { DocumentCard } from "../document-card";

// A proposed document as CaptureItemFields sends it: open, not yet filed,
// with no suggestion and nothing detected.
const ITEM: CaptureItem = {
  id: "cpi_01proposed",
  position: 1,
  status: "Proposed",
  pageIds: [],
  pageCount: 2,
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
  version: 3,
  suggestedRecord: null,
  filedRecord: null,
  filedBy: null,
} as unknown as CaptureItem;

function deferred() {
  let resolve!: () => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<void>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

function renderCard(onDiscard: () => Promise<unknown>, item: CaptureItem = ITEM) {
  const props = {
    number: 2,
    group: { key: ITEM.id, pageIds: [] },
    item,
    pages: new Map(),
    rotations: {},
    sequence: {},
    destination: { kind: "shipment", recordId: "", documentTypeId: "" },
    onDestinationChange: () => {},
    failure: undefined,
    moveTargets: [],
    pageActions: {
      preview: () => {},
      rotate: () => {},
      moveTo: () => {},
      splitAfter: () => {},
    },
    canEdit: true,
    canFile: true,
    canDiscard: true,
    dirty: false,
    onFile: () => {},
    filing: false,
    discardEffect: "set-aside",
    onDiscard,
  } as unknown as ComponentProps<typeof DocumentCard>;
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={queryClient}>
      <DndContext>
        <DocumentCard {...props} />
      </DndContext>
    </QueryClientProvider>,
  );
}

describe("DocumentCard discard", () => {
  it("names the action for what it does and asks before throwing the document away", async () => {
    const user = userEvent.setup();
    const onDiscard = vi.fn(() => Promise.resolve());
    renderCard(onDiscard);

    expect(screen.queryByRole("button", { name: "Set aside" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Discard document" }));

    expect(onDiscard).not.toHaveBeenCalled();
    const dialog = await screen.findByRole("alertdialog");
    expect(within(dialog).getByText("Discard document 2?")).toBeInTheDocument();

    await user.click(within(dialog).getByRole("button", { name: "Keep it" }));
    await waitFor(() => expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument());
    expect(onDiscard).not.toHaveBeenCalled();
  });

  it("keeps the dialog open until the discard settles and shows why it failed", async () => {
    const user = userEvent.setup();
    const pending = deferred();
    const onDiscard = vi.fn(() => pending.promise);
    renderCard(onDiscard);

    await user.click(screen.getByRole("button", { name: "Discard document" }));
    const dialog = await screen.findByRole("alertdialog");
    await user.click(within(dialog).getByRole("button", { name: "Discard" }));

    expect(onDiscard).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Keep it" })).toBeDisabled();

    pending.reject(
      new GraphQLRequestError({
        kind: "graphql",
        message: "Somebody else changed this document; reload it and try again",
        graphQLErrors: [
          {
            message: "Somebody else changed this document; reload it and try again",
            extensions: {},
            type: "business-rule-violation",
          },
        ],
        status: 200,
      }),
    );

    const alert = await within(dialog).findByRole("alert");
    expect(alert).toHaveTextContent("The document was not discarded");
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });

  it("closes once the discard succeeds", async () => {
    const user = userEvent.setup();
    const onDiscard = vi.fn(() => Promise.resolve());
    renderCard(onDiscard);

    await user.click(screen.getByRole("button", { name: "Discard document" }));
    const dialog = await screen.findByRole("alertdialog");
    await user.click(within(dialog).getByRole("button", { name: "Discard" }));

    await waitFor(() => expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument());
    expect(onDiscard).toHaveBeenCalledTimes(1);
  });
});

describe("DocumentCard fields", () => {
  it("labels the kind picker with the words shown above it", () => {
    renderCard(() => Promise.resolve());

    expect(screen.getByRole("combobox", { name: "File onto" })).toBeInTheDocument();
  });

  it("names what the reader took the document to be instead of its wire value", () => {
    renderCard(() => Promise.resolve(), {
      ...ITEM,
      detectedKind: "ProofOfDelivery",
    } as CaptureItem);

    expect(screen.getByText("Proof of delivery")).toBeInTheDocument();
    expect(screen.queryByText("ProofOfDelivery")).not.toBeInTheDocument();
  });

  it("shows nothing for a kind the reader could not name", () => {
    renderCard(() => Promise.resolve(), { ...ITEM, detectedKind: "Other" } as CaptureItem);

    expect(screen.queryByText("Other")).not.toBeInTheDocument();
  });
});
