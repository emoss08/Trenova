import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BillingQueueDocumentsTab, type PacketControl } from "../billing-queue-documents-tab";

type UploadOptions = {
  onSuccess?: (doc: unknown, upload: { metadata?: Record<string, string> }) => void;
};

const mocks = vi.hoisted(() => ({
  documents: vi.fn(),
  readiness: vi.fn(),
  uploadFiles: vi.fn(),
  bulkDelete: vi.fn(),
  uploadOptions: { current: null as UploadOptions | null },
}));

vi.mock("@/services/document", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/document")>()),
  documentService: {
    getByResource: mocks.documents,
    delete: vi.fn(),
    bulkDelete: mocks.bulkDelete,
  },
}));
vi.mock("@/lib/queries/shipment", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/queries/shipment")>()),
  shipment: {
    billingReadiness: (id: string) => ({
      queryKey: ["billing-readiness", id],
      queryFn: mocks.readiness,
    }),
  },
}));
vi.mock("@/components/autocomplete-fields", () => ({ DocumentTypeAutocompleteField: () => null }));
vi.mock("@/components/documents/upload-panel", () => ({
  UploadPanel: ({ isOpen }: { isOpen: boolean }) =>
    isOpen ? <div data-testid="upload-panel" /> : null,
}));
vi.mock("@/hooks/use-document-upload", () => ({
  useDocumentUpload: (options: UploadOptions) => {
    mocks.uploadOptions.current = options;
    return {
      uploads: [],
      uploadFiles: mocks.uploadFiles,
      cancelUpload: vi.fn(),
      retryUpload: vi.fn(),
      removeUpload: vi.fn(),
      clearCompleted: vi.fn(),
    };
  },
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));
vi.mock("@/components/image", () => ({
  LazyImage: ({ src, alt }: { src: string; alt: string }) => <img src={src} alt={alt} />,
}));
vi.mock("../billing-queue-packet-preview", () => ({ BillingQueuePacketPreview: () => null }));
vi.mock("@/components/documents/document-signature-dialog", () => ({
  DocumentSignatureDialog: () => null,
}));

const POD_TYPE = { id: "dt_pod", name: "Proof of delivery", requiresSignature: true };
const BOL_TYPE = { id: "dt_bol", name: "Bill of lading", requiresSignature: false };
const LUMPER_TYPE = { id: "dt_lumper", name: "Lumper receipt", requiresSignature: false };

function paper(
  id: string,
  type: { id: string; name: string; requiresSignature: boolean } | null,
  extra: Record<string, unknown> = {},
) {
  return {
    id,
    lineageId: `lin_${id}`,
    documentTypeId: type?.id ?? null,
    documentType: type,
    originalName: `${id}.pdf`,
    fileName: `${id}.pdf`,
    fileType: "application/pdf",
    fileSize: 48_100,
    status: "Active",
    signatureStatus: "Signed",
    previewStatus: "Pending",
    isCurrentVersion: true,
    createdAt: Math.floor(Date.now() / 1000) - 3600,
    ...extra,
  };
}

function requirement(type: typeof POD_TYPE, documentIds: string[]) {
  return {
    documentTypeId: type.id,
    documentTypeCode: type.id.toUpperCase(),
    documentTypeName: type.name,
    satisfied: documentIds.length > 0,
    documentCount: documentIds.length,
    documentIds,
    ineligibleDocuments: [],
  };
}

function packetControl(overrides: Partial<PacketControl> = {}): PacketControl {
  return {
    documentIds: [],
    picked: (overrides.documentIds?.length ?? 0) > 0,
    editable: true,
    onChange: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

function renderTab(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>{ui}</QueryClientProvider>
    </MemoryRouter>,
  );
}

function tab(packet?: PacketControl, isEditable = true) {
  return (
    <BillingQueueDocumentsTab
      shipmentId="shp_1"
      onDocumentSelect={vi.fn()}
      isEditable={isEditable}
      packet={packet}
    />
  );
}

const lane = (name: string) => within(screen.getByRole("region", { name: new RegExp(name) }));

/** The lane's places in order, each found by the name a screen reader gives it. */
function expectCards(laneName: string, names: string[]) {
  const places = lane(laneName)
    .getAllByRole("button")
    .filter((button) => !/^(Send|Leave) /.test(button.getAttribute("aria-label") ?? ""));
  expect(places).toEqual(names.map((name) => lane(laneName).getByRole("button", { name })));
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("documents tab packet board", () => {
  it("lays the default packet out requirement by requirement, numbered, with a slot for what is missing", async () => {
    mocks.documents.mockResolvedValue([
      paper("doc_lumper", LUMPER_TYPE),
      paper("doc_pod", POD_TYPE),
    ]);
    mocks.readiness.mockResolvedValue({
      requirements: [requirement(POD_TYPE, ["doc_pod"]), requirement(BOL_TYPE, [])],
    });
    renderTab(tab(packetControl()));

    await screen.findByRole("region", { name: /Packet/ });
    expectCards("Packet", [
      "Proof of delivery, number 1 in the packet",
      "Upload or drop the Bill of lading",
    ]);
    expectCards("Not sent", ["Lumper receipt", "Add files"]);
  });

  it("follows the biller's own order once they have picked", async () => {
    mocks.documents.mockResolvedValue([
      paper("doc_pod", POD_TYPE),
      paper("doc_lumper", LUMPER_TYPE),
    ]);
    mocks.readiness.mockResolvedValue({ requirements: [requirement(POD_TYPE, ["doc_pod"])] });
    renderTab(tab(packetControl({ documentIds: ["doc_lumper", "doc_pod"] })));

    await screen.findByRole("region", { name: /Packet/ });
    expectCards("Packet", [
      "Lumper receipt, number 1 in the packet",
      "Proof of delivery, number 2 in the packet",
    ]);
  });

  it("sends a document with the invoice from its arrow, holding the card until the save settles", async () => {
    let settle: () => void = () => undefined;
    const onChange = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          settle = resolve;
        }),
    );
    mocks.documents.mockResolvedValue([
      paper("doc_pod", POD_TYPE),
      paper("doc_lumper", LUMPER_TYPE),
    ]);
    mocks.readiness.mockResolvedValue({ requirements: [requirement(POD_TYPE, ["doc_pod"])] });
    const user = userEvent.setup();
    renderTab(tab(packetControl({ onChange })));

    await user.click(
      await screen.findByRole("button", { name: "Send Lumper receipt with the invoice" }),
    );

    expect(onChange).toHaveBeenCalledWith(["doc_pod", "doc_lumper"]);
    expect(screen.getByRole("status", { name: "Saving" })).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Send Lumper receipt with the invoice" }),
    ).not.toBeInTheDocument();
    settle();
    await waitFor(() =>
      expect(screen.queryByRole("status", { name: "Saving" })).not.toBeInTheDocument(),
    );
  });

  it("moves a packet document one place with the arrow keys", async () => {
    const onChange = vi.fn().mockResolvedValue(undefined);
    mocks.documents.mockResolvedValue([
      paper("doc_pod", POD_TYPE),
      paper("doc_lumper", LUMPER_TYPE),
    ]);
    mocks.readiness.mockResolvedValue({ requirements: [] });
    const user = userEvent.setup();
    renderTab(tab(packetControl({ documentIds: ["doc_pod", "doc_lumper"], onChange })));

    const lumper = await screen.findByRole("button", {
      name: "Lumper receipt, number 2 in the packet",
    });
    lumper.focus();
    await user.keyboard("{ArrowLeft}");

    expect(onChange).toHaveBeenCalledWith(["doc_lumper", "doc_pod"]);
  });

  it("does not move a document past either end of the packet", async () => {
    const onChange = vi.fn().mockResolvedValue(undefined);
    mocks.documents.mockResolvedValue([paper("doc_pod", POD_TYPE)]);
    mocks.readiness.mockResolvedValue({ requirements: [] });
    const user = userEvent.setup();
    renderTab(tab(packetControl({ documentIds: ["doc_pod"], onChange })));

    (
      await screen.findByRole("button", { name: "Proof of delivery, number 1 in the packet" })
    ).focus();
    await user.keyboard("{ArrowLeft}{ArrowRight}");

    expect(onChange).not.toHaveBeenCalled();
  });

  it("offers no moves once the packet can no longer change", async () => {
    mocks.documents.mockResolvedValue([
      paper("doc_pod", POD_TYPE),
      paper("doc_lumper", LUMPER_TYPE),
    ]);
    mocks.readiness.mockResolvedValue({ requirements: [requirement(POD_TYPE, ["doc_pod"])] });
    renderTab(tab(packetControl({ editable: false }), false));

    await screen.findByRole("region", { name: /Packet/ });
    expect(screen.queryByRole("button", { name: /with the invoice|out of the packet/ })).toBeNull();
    expect(screen.queryByRole("button", { name: "Add files" })).toBeNull();
  });

  it("uploads a file dropped on a missing slot as that slot's type", async () => {
    mocks.documents.mockResolvedValue([paper("doc_pod", POD_TYPE)]);
    mocks.readiness.mockResolvedValue({
      requirements: [requirement(POD_TYPE, ["doc_pod"]), requirement(BOL_TYPE, [])],
    });
    renderTab(tab(packetControl()));

    const slot = (await screen.findByRole("button", { name: "Upload or drop the Bill of lading" }))
      .parentElement!;
    const file = new File(["%PDF"], "bol.pdf", { type: "application/pdf" });
    fireEvent.dragOver(slot, { dataTransfer: { types: ["Files"], files: [file] } });
    fireEvent.drop(slot, { dataTransfer: { types: ["Files"], files: [file] } });

    expect(mocks.uploadFiles).toHaveBeenCalledWith([file], {
      documentTypeId: "dt_bol",
      packetSlot: "dt_bol",
    });
  });

  it("refuses a dropped file it cannot upload", async () => {
    mocks.documents.mockResolvedValue([]);
    mocks.readiness.mockResolvedValue({ requirements: [requirement(BOL_TYPE, [])] });
    renderTab(tab(packetControl()));

    const slot = (await screen.findByRole("button", { name: "Upload or drop the Bill of lading" }))
      .parentElement!;
    const file = new File(["MZ"], "setup.exe", { type: "application/x-msdownload" });
    fireEvent.drop(slot, { dataTransfer: { types: ["Files"], files: [file] } });

    expect(mocks.uploadFiles).not.toHaveBeenCalled();
  });

  it("adds a document uploaded into a missing slot to a packet the biller picked", async () => {
    const onChange = vi.fn().mockResolvedValue(undefined);
    mocks.documents.mockResolvedValue([paper("doc_pod", POD_TYPE)]);
    mocks.readiness.mockResolvedValue({
      requirements: [requirement(POD_TYPE, ["doc_pod"]), requirement(BOL_TYPE, [])],
    });
    renderTab(tab(packetControl({ documentIds: ["doc_pod"], onChange })));
    await screen.findByRole("region", { name: /Packet/ });

    mocks.uploadOptions.current?.onSuccess?.(paper("doc_bol", BOL_TYPE), {
      metadata: { documentTypeId: "dt_bol", packetSlot: "dt_bol" },
    });
    await waitFor(() => expect(onChange).toHaveBeenCalledWith(["doc_pod", "doc_bol"]));

    onChange.mockClear();
    mocks.uploadOptions.current?.onSuccess?.(paper("doc_photo", null), { metadata: {} });
    expect(onChange).not.toHaveBeenCalled();
  });

  it("shows the chosen document's file and the one fix it needs", async () => {
    mocks.documents.mockResolvedValue([
      paper("doc_lumper", LUMPER_TYPE),
      paper("doc_pod", POD_TYPE, { signatureStatus: "Unsigned" }),
    ]);
    mocks.readiness.mockResolvedValue({ requirements: [requirement(POD_TYPE, ["doc_pod"])] });
    const user = userEvent.setup();
    renderTab(tab(packetControl()));

    expect(await screen.findByText(/^Select a document to see its file/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /^Proof of delivery, Unsigned/ }));

    expect(screen.getByText(/^doc_pod\.pdf · /)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Record signature" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Lumper receipt" }));

    expect(screen.getByText(/^doc_lumper\.pdf · /)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Record signature" })).toBeNull();
  });

  it("lays every document in one lane when the item sends no packet", async () => {
    mocks.documents.mockResolvedValue([
      paper("doc_pod", POD_TYPE),
      paper("doc_lumper", LUMPER_TYPE),
    ]);
    mocks.readiness.mockResolvedValue({ requirements: [requirement(POD_TYPE, ["doc_pod"])] });
    renderTab(tab(undefined, false));

    await screen.findByRole("region", { name: "Shipment documents" });
    expectCards("Shipment documents", ["Proof of delivery", "Lumper receipt"]);
    expect(screen.queryByRole("region", { name: /Packet/ })).toBeNull();
  });

  it("leaves every ticked document out of the packet in one save from the dock", async () => {
    const onChange = vi.fn().mockResolvedValue(undefined);
    mocks.documents.mockResolvedValue([
      paper("doc_pod", POD_TYPE),
      paper("doc_bol", BOL_TYPE),
      paper("doc_lumper", LUMPER_TYPE),
    ]);
    mocks.readiness.mockResolvedValue({ requirements: [] });
    const user = userEvent.setup();
    renderTab(tab(packetControl({ documentIds: ["doc_pod", "doc_bol", "doc_lumper"], onChange })));

    await user.click(await screen.findByRole("checkbox", { name: "Select Proof of delivery" }));
    await user.click(screen.getByRole("checkbox", { name: "Select Lumper receipt" }));
    expect(screen.getByText("2 selected")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Send with invoice" })).toBeNull();
    await user.click(screen.getByRole("button", { name: "Leave out" }));

    expect(onChange).toHaveBeenCalledWith(["doc_bol"]);
    await waitFor(() => expect(screen.queryByText("2 selected")).not.toBeInTheDocument());
  });

  it("sends ticked documents with the invoice after the ones already in it", async () => {
    const onChange = vi.fn().mockResolvedValue(undefined);
    mocks.documents.mockResolvedValue([
      paper("doc_pod", POD_TYPE),
      paper("doc_bol", BOL_TYPE),
      paper("doc_lumper", LUMPER_TYPE),
    ]);
    mocks.readiness.mockResolvedValue({ requirements: [] });
    const user = userEvent.setup();
    renderTab(tab(packetControl({ documentIds: ["doc_pod"], onChange })));

    await user.click(await screen.findByRole("checkbox", { name: "Select Lumper receipt" }));
    await user.click(screen.getByRole("checkbox", { name: "Select Bill of lading" }));
    await user.click(screen.getByRole("button", { name: "Send with invoice" }));

    expect(onChange).toHaveBeenCalledWith(["doc_pod", "doc_bol", "doc_lumper"]);
  });

  it("ticks every card between the last one ticked and a shift-clicked one in the same lane", async () => {
    mocks.documents.mockResolvedValue([
      paper("doc_pod", POD_TYPE),
      paper("doc_bol", BOL_TYPE),
      paper("doc_lumper", LUMPER_TYPE),
    ]);
    mocks.readiness.mockResolvedValue({ requirements: [] });
    const user = userEvent.setup();
    renderTab(tab(packetControl({ documentIds: ["doc_pod", "doc_bol", "doc_lumper"] })));

    await user.click(await screen.findByRole("checkbox", { name: "Select Proof of delivery" }));
    await user.keyboard("{Shift>}");
    await user.click(screen.getByRole("checkbox", { name: "Select Lumper receipt" }));
    await user.keyboard("{/Shift}");

    expect(screen.getByRole("checkbox", { name: "Select Bill of lading" })).toBeChecked();
    expect(screen.getByText("3 selected")).toBeInTheDocument();
  });

  it("asks before deleting the ticked documents, then deletes them together", async () => {
    mocks.bulkDelete.mockResolvedValue({ deletedCount: 2, errorCount: 0 });
    mocks.documents.mockResolvedValue([
      paper("doc_pod", POD_TYPE),
      paper("doc_lumper", LUMPER_TYPE),
    ]);
    mocks.readiness.mockResolvedValue({ requirements: [] });
    const user = userEvent.setup();
    renderTab(tab(packetControl({ documentIds: ["doc_pod"] })));

    await user.click(await screen.findByRole("checkbox", { name: "Select Proof of delivery" }));
    await user.click(screen.getByRole("checkbox", { name: "Select Lumper receipt" }));
    await user.click(screen.getByRole("button", { name: "Delete" }));

    expect(mocks.bulkDelete).not.toHaveBeenCalled();
    const dialog = await screen.findByRole("alertdialog", { name: "Delete 2 documents?" });
    await user.click(within(dialog).getByRole("button", { name: "Delete" }));

    expect(mocks.bulkDelete).toHaveBeenCalledWith(["doc_pod", "doc_lumper"]);
    await waitFor(() => expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument());
  });

  it("offers no ticking once nothing about the documents can change", async () => {
    mocks.documents.mockResolvedValue([paper("doc_pod", POD_TYPE)]);
    mocks.readiness.mockResolvedValue({ requirements: [] });
    renderTab(tab(packetControl({ editable: false }), false));

    await screen.findByRole("region", { name: /Packet/ });
    expect(screen.queryByRole("checkbox")).toBeNull();
  });

  it("adds a card with ⌘ or Ctrl, swaps the line for the dock, and a plain click goes back to one", async () => {
    mocks.documents.mockResolvedValue([
      paper("doc_pod", POD_TYPE),
      paper("doc_lumper", LUMPER_TYPE),
    ]);
    mocks.readiness.mockResolvedValue({ requirements: [] });
    const user = userEvent.setup();
    renderTab(tab(packetControl({ documentIds: ["doc_pod"] })));

    const pod = await screen.findByRole("button", {
      name: "Proof of delivery, number 1 in the packet",
    });
    const lumper = screen.getByRole("button", { name: "Lumper receipt" });
    await user.click(pod);
    expect(pod).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByText(/^doc_pod\.pdf · /)).toBeInTheDocument();

    await user.keyboard("{Control>}");
    await user.click(lumper);
    await user.keyboard("{/Control}");

    expect(lumper).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("checkbox", { name: "Select Proof of delivery" })).toBeChecked();
    expect(screen.getByText("2 selected")).toBeInTheDocument();
    expect(screen.queryByText(/^doc_pod\.pdf · /)).toBeNull();

    await user.click(lumper);

    expect(pod).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByText(/^doc_lumper\.pdf · /)).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByText("2 selected")).not.toBeInTheDocument());
  });

  it("selects only one card at a time when nothing about the documents can change", async () => {
    mocks.documents.mockResolvedValue([
      paper("doc_pod", POD_TYPE),
      paper("doc_lumper", LUMPER_TYPE),
    ]);
    mocks.readiness.mockResolvedValue({ requirements: [] });
    const user = userEvent.setup();
    renderTab(tab(undefined, false));

    await user.click(await screen.findByRole("button", { name: "Proof of delivery" }));
    await user.keyboard("{Control>}");
    await user.click(screen.getByRole("button", { name: "Lumper receipt" }));
    await user.keyboard("{/Control}");

    expect(screen.getByRole("button", { name: "Proof of delivery" })).toHaveAttribute(
      "aria-pressed",
      "false",
    );
    expect(screen.queryByText(/selected$/)).toBeNull();
  });

  it("asks before deleting the one selected document too", async () => {
    mocks.bulkDelete.mockResolvedValue({ deletedCount: 1, errorCount: 0 });
    mocks.documents.mockResolvedValue([paper("doc_pod", POD_TYPE)]);
    mocks.readiness.mockResolvedValue({ requirements: [] });
    const user = userEvent.setup();
    renderTab(tab(packetControl({ documentIds: ["doc_pod"] })));

    await user.click(
      await screen.findByRole("button", { name: "Proof of delivery, number 1 in the packet" }),
    );
    await user.click(screen.getByRole("button", { name: "More for Proof of delivery" }));
    await user.click(await screen.findByRole("menuitem", { name: "Delete" }));

    const dialog = await screen.findByRole("alertdialog", { name: "Delete 1 document?" });
    await user.click(within(dialog).getByRole("button", { name: "Delete" }));

    expect(mocks.bulkDelete).toHaveBeenCalledWith(["doc_pod"]);
  });

  it("clears the selection with Escape", async () => {
    mocks.documents.mockResolvedValue([
      paper("doc_pod", POD_TYPE),
      paper("doc_lumper", LUMPER_TYPE),
    ]);
    mocks.readiness.mockResolvedValue({ requirements: [] });
    const user = userEvent.setup();
    renderTab(tab(packetControl({ documentIds: ["doc_pod"] })));

    await user.click(await screen.findByRole("checkbox", { name: "Select Proof of delivery" }));
    await user.click(screen.getByRole("checkbox", { name: "Select Lumper receipt" }));
    expect(screen.getByText("2 selected")).toBeInTheDocument();
    await user.keyboard("{Escape}");

    await waitFor(() => expect(screen.queryByText("2 selected")).not.toBeInTheDocument());
    expect(screen.getByRole("checkbox", { name: "Select Proof of delivery" })).not.toBeChecked();
  });

  it("takes the last document out of the packet as a pick of none", async () => {
    const onChange = vi.fn().mockResolvedValue(undefined);
    mocks.documents.mockResolvedValue([paper("doc_pod", POD_TYPE)]);
    mocks.readiness.mockResolvedValue({ requirements: [requirement(POD_TYPE, ["doc_pod"])] });
    const user = userEvent.setup();
    renderTab(tab(packetControl({ onChange })));

    await user.click(
      await screen.findByRole("button", { name: "Leave Proof of delivery out of the packet" }),
    );

    expect(onChange).toHaveBeenCalledWith([]);
  });
});
