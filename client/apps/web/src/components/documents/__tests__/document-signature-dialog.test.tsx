import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { documentSchema, type Document } from "@trenova/shared/types/document";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DocumentSignatureDialog } from "../document-signature-dialog";

const { setSignature } = vi.hoisted(() => ({ setSignature: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: { documentService: { setSignature } },
}));

afterEach(cleanup);
beforeEach(() => vi.clearAllMocks());

function pod(overrides: Partial<Record<string, unknown>> = {}): Document {
  return documentSchema.parse({
    id: "doc_pod",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    lineageId: "doc_pod",
    versionNumber: 1,
    isCurrentVersion: true,
    fileName: "pod.pdf",
    originalName: "pod-4471.pdf",
    fileSize: 2048,
    fileType: "application/pdf",
    storagePath: "org_1/bu_1/pod.pdf",
    status: "Active",
    resourceId: "shp_1",
    resourceType: "shipment",
    isPublic: false,
    uploadedById: "usr_1",
    previewStatus: "Ready",
    contentStatus: "Indexed",
    hasExtractedText: true,
    shipmentDraftStatus: "Unavailable",
    signatureStatus: "Unknown",
    signedAt: null,
    signatureSource: "",
    version: 7,
    createdAt: 1_792_000_000,
    updatedAt: 1_792_000_000,
    ...overrides,
  });
}

function renderDialog(document: Document) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const invalidate = vi.spyOn(client, "invalidateQueries");
  const onSaved = vi.fn();
  const onOpenChange = vi.fn();
  render(
    <QueryClientProvider client={client}>
      <DocumentSignatureDialog
        document={document}
        open
        onOpenChange={onOpenChange}
        onSaved={onSaved}
      />
    </QueryClientProvider>,
  );
  return { invalidate, onSaved, onOpenChange };
}

describe("DocumentSignatureDialog", () => {
  it("records an unsigned POD against the version the person opened", async () => {
    const saved = pod({ signatureStatus: "Unsigned", signatureSource: "Person", version: 8 });
    setSignature.mockResolvedValue(saved);
    const { invalidate, onSaved, onOpenChange } = renderDialog(pod());

    await userEvent.click(screen.getByRole("radio", { name: "Unsigned" }));
    await userEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(onSaved).toHaveBeenCalledWith(saved));
    expect(setSignature).toHaveBeenCalledWith("doc_pod", {
      status: "Unsigned",
      signedAt: null,
      version: 7,
    });
    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["billingQueue"] });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["documents"] });
  });

  it("drops the signing date when a signed POD is marked not read", async () => {
    setSignature.mockResolvedValue(pod({ signatureSource: "Person", version: 8 }));
    const { onSaved } = renderDialog(
      pod({ signatureStatus: "Signed", signedAt: 1_791_950_000, signatureSource: "Extraction" }),
    );

    expect(screen.getByRole("radio", { name: "Signed" })).toHaveAttribute("aria-checked", "true");
    await userEvent.click(screen.getByRole("radio", { name: "Signature not read" }));
    await userEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(onSaved).toHaveBeenCalled());
    expect(setSignature).toHaveBeenCalledWith("doc_pod", {
      status: "Unknown",
      signedAt: null,
      version: 7,
    });
  });

  it("keeps a signed POD's date", async () => {
    setSignature.mockResolvedValue(pod({ signatureStatus: "Signed", version: 8 }));
    const { onSaved } = renderDialog(
      pod({ signatureStatus: "Signed", signedAt: 1_791_950_000, signatureSource: "Extraction" }),
    );

    await userEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(onSaved).toHaveBeenCalled());
    expect(setSignature).toHaveBeenCalledWith("doc_pod", {
      status: "Signed",
      signedAt: 1_791_950_000,
      version: 7,
    });
  });
});
