import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactElement } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { LoadDocuments } from "./load-documents";
import { ProfileDocuments } from "./profile-documents";

const {
  fetchMyLoadDocuments,
  fetchMyLoadDocumentTypes,
  fetchMyProfileDocuments,
  fetchMyProfileDocumentTypes,
  fetchMyPortalFeatures,
  uploadMyLoadDocument,
  uploadMyProfileDocument,
} = vi.hoisted(() => ({
  fetchMyLoadDocuments: vi.fn(),
  fetchMyLoadDocumentTypes: vi.fn(),
  fetchMyProfileDocuments: vi.fn(),
  fetchMyProfileDocumentTypes: vi.fn(),
  fetchMyPortalFeatures: vi.fn(),
  uploadMyLoadDocument: vi.fn(),
  uploadMyProfileDocument: vi.fn(),
}));

vi.mock("@trenova/shared/lib/graphql/driver-portal", () => ({
  fetchMyLoadDocuments,
  fetchMyLoadDocumentTypes,
  fetchMyProfileDocuments,
  fetchMyProfileDocumentTypes,
  fetchMyPortalFeatures,
}));

vi.mock("@trenova/shared/lib/portal", () => ({
  uploadMyLoadDocument,
  uploadMyProfileDocument,
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const createdAt = 1_800_000_000;

function renderWithClient(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

function pickFile(container: HTMLElement, name: string) {
  const input = container.querySelector<HTMLInputElement>('input[type="file"]');
  if (!input) {
    throw new Error("file input not rendered");
  }
  fireEvent.change(input, { target: { files: [new File(["scan"], name)] } });
}

describe("LoadDocuments", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("lists the load's documents from the driver's GraphQL query", async () => {
    fetchMyPortalFeatures.mockResolvedValue(undefined);
    fetchMyLoadDocumentTypes.mockResolvedValue([]);
    fetchMyLoadDocuments.mockResolvedValue([
      {
        id: "doc_1",
        fileName: "pod.pdf",
        fileSize: 2048,
        status: "Active",
        documentTypeName: "Proof of Delivery",
        createdAt,
      },
    ]);

    renderWithClient(<LoadDocuments shipmentId="shp_1" />);

    expect(await screen.findByText("pod.pdf")).toBeInTheDocument();
    expect(screen.getByText("Proof of Delivery")).toBeInTheDocument();
    expect(fetchMyLoadDocuments).toHaveBeenCalledWith("shp_1", {
      signal: expect.any(AbortSignal),
    });
  });

  it("tags an upload with a shipment document type and refreshes the list", async () => {
    fetchMyPortalFeatures.mockResolvedValue(undefined);
    fetchMyLoadDocuments.mockResolvedValue([]);
    fetchMyLoadDocumentTypes.mockResolvedValue([
      { id: "dt_pod", code: "POD", name: "Proof of Delivery", color: "#16a34a" },
    ]);
    uploadMyLoadDocument.mockResolvedValue({ id: "doc_2" });

    const { container } = renderWithClient(<LoadDocuments shipmentId="shp_1" />);
    await waitFor(() => expect(fetchMyLoadDocumentTypes).toHaveBeenCalled());

    pickFile(container, "bol.jpg");
    fireEvent.click(await screen.findByRole("button", { name: "Proof of Delivery" }));
    fireEvent.click(screen.getByRole("button", { name: "Upload" }));

    await waitFor(() =>
      expect(uploadMyLoadDocument).toHaveBeenCalledWith("shp_1", expect.any(File), "dt_pod"),
    );
    await waitFor(() => expect(fetchMyLoadDocuments).toHaveBeenCalledTimes(2));
  });
});

describe("ProfileDocuments", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("lists the driver's qualification-file documents", async () => {
    fetchMyPortalFeatures.mockResolvedValue(undefined);
    fetchMyProfileDocumentTypes.mockResolvedValue([]);
    fetchMyProfileDocuments.mockResolvedValue([
      {
        id: "doc_3",
        fileName: "cdl-front.jpg",
        fileSize: 512,
        status: "Pending",
        documentTypeName: "",
        createdAt,
      },
    ]);

    renderWithClient(<ProfileDocuments />);

    expect(await screen.findByText("cdl-front.jpg")).toBeInTheDocument();
    expect(fetchMyProfileDocuments).toHaveBeenCalledWith({ signal: expect.any(AbortSignal) });
  });

  it("tags an upload with a worker document type and refreshes the list", async () => {
    fetchMyPortalFeatures.mockResolvedValue(undefined);
    fetchMyProfileDocuments.mockResolvedValue([]);
    fetchMyProfileDocumentTypes.mockResolvedValue([
      { id: "dt_cdl", code: "CDL", name: "Commercial Driver's License", color: "#2563eb" },
    ]);
    uploadMyProfileDocument.mockResolvedValue({ id: "doc_4" });

    const { container } = renderWithClient(<ProfileDocuments />);
    await waitFor(() => expect(fetchMyProfileDocumentTypes).toHaveBeenCalled());

    pickFile(container, "medical-card.jpg");
    fireEvent.click(await screen.findByRole("button", { name: "Commercial Driver's License" }));
    fireEvent.click(screen.getByRole("button", { name: "Upload" }));

    await waitFor(() =>
      expect(uploadMyProfileDocument).toHaveBeenCalledWith(expect.any(File), "dt_cdl"),
    );
    await waitFor(() => expect(fetchMyProfileDocuments).toHaveBeenCalledTimes(2));
  });
});
