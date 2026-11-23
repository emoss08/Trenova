import { PENDING_DOCUMENTS_REFETCH_MS } from "@/lib/resource-documents";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Document } from "@trenova/shared/types/document";
import type { UploadState } from "@/types/upload";
import { useState } from "react";
import { beforeEach, describe, expect, it, onTestFinished, vi } from "vitest";
import { documentContentUrl } from "@/services/document";
import { FileDropField, type AttachedFile } from "../file-drop-field";

type HookOptions = {
  resourceId: string;
  resourceType: string;
  onSuccess?: (document: Document, upload: UploadState) => void;
  onError?: (error: Error, file: File) => void;
};

const hook = vi.hoisted(() => ({
  options: null as HookOptions | null,
  uploads: [] as UploadState[],
  uploadFiles: vi.fn(),
  cancelUpload: vi.fn(),
  removeUpload: vi.fn(),
}));

const documentService = vi.hoisted(() => ({ getById: vi.fn() }));

vi.mock("@/services/document", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/document")>()),
  documentService,
}));

vi.mock("@/hooks/use-document-upload", () => ({
  useDocumentUpload: (options: HookOptions) => {
    hook.options = options;
    return {
      uploads: hook.uploads,
      uploadFiles: hook.uploadFiles,
      cancelUpload: hook.cancelUpload,
      removeUpload: hook.removeUpload,
      retryUpload: vi.fn(),
      clearCompleted: vi.fn(),
      clearAll: vi.fn(),
      isUploading: hook.uploads.length > 0,
    };
  },
}));

function pdf(name = "med-card-2026.pdf", bytes = 2048): File {
  return new File([new Uint8Array(bytes)], name, { type: "application/pdf" });
}

function Field({
  value = null,
  onChange = vi.fn(),
  maxFileSize,
  onUploadingChange,
}: {
  value?: AttachedFile | null;
  onChange?: (file: AttachedFile | null) => void;
  maxFileSize?: number;
  onUploadingChange?: (uploading: boolean) => void;
}) {
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false } } }),
  );
  return (
    <QueryClientProvider client={client}>
      <FileDropField
        label="Document"
        note="needed before this credential can be verified"
        value={value}
        onChange={onChange}
        resourceId="wrk_1"
        resourceType="worker"
        maxFileSize={maxFileSize}
        onUploadingChange={onUploadingChange}
      />
    </QueryClientProvider>
  );
}

describe("FileDropField", () => {
  beforeEach(() => {
    hook.options = null;
    hook.uploads = [];
    hook.uploadFiles.mockReset();
    hook.cancelUpload.mockReset();
    hook.removeUpload.mockReset();
    documentService.getById.mockReset();
  });

  it("offers a zone to drop or choose the document, with the types it takes and why it is wanted", () => {
    render(<Field />);

    expect(
      screen.getByRole("group", { name: "Document needed before this credential can be verified" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Drop the document here")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "choose a file" })).toBeInTheDocument();
    expect(screen.getByText("PDF, JPG, PNG or WEBP.")).toBeInTheDocument();
    expect(hook.options).toMatchObject({ resourceId: "wrk_1", resourceType: "worker" });
  });

  it("uploads the one file chosen", async () => {
    const user = userEvent.setup();
    render(<Field />);
    const file = pdf();

    await user.upload(screen.getByLabelText("Choose a file"), file);

    expect(hook.uploadFiles).toHaveBeenCalledTimes(1);
    expect(hook.uploadFiles).toHaveBeenCalledWith([file]);
  });

  it("uploads only the first of several files dropped on it", () => {
    render(<Field />);
    const first = pdf("first.pdf");

    fireEvent.drop(screen.getByText("Drop the document here").closest("div")!, {
      dataTransfer: { files: [first, pdf("second.pdf")] },
    });

    expect(hook.uploadFiles).toHaveBeenCalledWith([first]);
  });

  it("refuses a type it does not take and says so, uploading nothing", () => {
    render(<Field />);

    fireEvent.drop(screen.getByText("Drop the document here").closest("div")!, {
      dataTransfer: { files: [new File(["x"], "setup.exe", { type: "application/x-msdownload" })] },
    });

    expect(hook.uploadFiles).not.toHaveBeenCalled();
    expect(screen.getByRole("alert")).toHaveTextContent(
      "setup.exe is not a type this field takes.",
    );
  });

  it("refuses a file larger than the field takes", () => {
    render(<Field maxFileSize={1024} />);

    fireEvent.drop(screen.getByText("Drop the document here").closest("div")!, {
      dataTransfer: { files: [pdf("scan.pdf", 4096)] },
    });

    expect(hook.uploadFiles).not.toHaveBeenCalled();
    expect(screen.getByRole("alert")).toHaveTextContent("scan.pdf is larger than 1.0 KB.");
  });

  it("shows an upload in flight with its progress, and cancels it", async () => {
    const user = userEvent.setup();
    hook.uploads = [{ id: "up_1", file: pdf(), progress: 61.6, status: "uploading" }];
    render(<Field />);

    expect(screen.getByText("med-card-2026.pdf")).toBeInTheDocument();
    expect(screen.getByText("2.0 KB · 62%")).toBeInTheDocument();
    expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "61.6");
    expect(screen.queryByText("Drop the document here")).not.toBeInTheDocument();

    await user.click(
      screen.getByRole("button", { name: "Cancel the upload of med-card-2026.pdf" }),
    );

    expect(hook.cancelUpload).toHaveBeenCalledWith("up_1");
  });

  it("attaches the document an upload became", () => {
    const onChange = vi.fn();
    render(<Field onChange={onChange} />);
    const upload: UploadState = { id: "up_1", file: pdf(), progress: 100, status: "success" };

    act(() => {
      hook.options?.onSuccess?.(
        {
          id: "doc_1",
          originalName: "med-card-2026.pdf",
          fileSize: 421_888,
          fileType: "application/pdf",
          previewStatus: "Pending",
        } as Document,
        upload,
      );
    });

    expect(onChange).toHaveBeenCalledWith({
      id: "doc_1",
      name: "med-card-2026.pdf",
      size: 421_888,
      fileType: "application/pdf",
      previewStatus: "Pending",
    });
    expect(hook.removeUpload).toHaveBeenCalledWith("up_1");
  });

  it("shows the attached document and takes it off the form when removed", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <Field
        value={{
          id: "doc_1",
          name: "med-card-2026.pdf",
          size: 421_888,
          fileType: "application/pdf",
          previewStatus: "Ready",
        }}
        onChange={onChange}
      />,
    );

    expect(screen.getByText("med-card-2026.pdf")).toBeInTheDocument();
    expect(screen.getByText("412 KB")).toBeInTheDocument();
    expect(screen.queryByText("Drop the document here")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Remove med-card-2026.pdf" }));

    expect(onChange).toHaveBeenCalledWith(null);
  });

  it("leaves the size off an attached document whose size is not known", () => {
    render(
      <Field
        value={{
          id: "doc_1",
          name: "Attached document",
          size: 0,
          fileType: null,
          previewStatus: null,
        }}
      />,
    );

    expect(screen.getByText("Attached document")).toBeInTheDocument();
    expect(screen.queryByText("0 B")).not.toBeInTheDocument();
  });

  it("says why an upload failed and offers the zone again", () => {
    render(<Field />);

    act(() => {
      hook.options?.onError?.(new Error("Network unreachable"), pdf());
    });

    expect(screen.getByRole("alert")).toHaveTextContent("The upload failed: Network unreachable");
    expect(screen.getByText("Drop the document here")).toBeInTheDocument();
  });

  it("says when the virus scanner blocked the file", () => {
    hook.uploads = [
      {
        id: "up_1",
        file: pdf(),
        progress: 100,
        status: "quarantined",
        error: "Blocked: the virus scanner found Eicar-Test-Signature in this file",
      },
    ];
    render(<Field />);

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Blocked: the virus scanner found Eicar-Test-Signature in this file",
    );
  });

  it("tells the form while a file uploads and again once it has landed", () => {
    const onUploadingChange = vi.fn<(uploading: boolean) => void>();
    const { rerender } = render(<Field onUploadingChange={onUploadingChange} />);
    expect(onUploadingChange).toHaveBeenLastCalledWith(false);

    hook.uploads = [{ id: "up_1", file: pdf(), progress: 10, status: "uploading" }];
    rerender(<Field onUploadingChange={onUploadingChange} />);
    expect(onUploadingChange).toHaveBeenLastCalledWith(true);

    hook.uploads = [{ id: "up_1", file: pdf(), progress: 100, status: "verifying" }];
    rerender(<Field onUploadingChange={onUploadingChange} />);
    expect(onUploadingChange).toHaveBeenLastCalledWith(true);

    hook.uploads = [];
    rerender(
      <Field
        value={{
          id: "doc_1",
          name: "med-card-2026.pdf",
          size: 2048,
          fileType: "application/pdf",
          previewStatus: "Ready",
        }}
        onUploadingChange={onUploadingChange}
      />,
    );
    expect(onUploadingChange).toHaveBeenLastCalledWith(false);
  });

  it("lets the form go when it is taken away mid-upload", () => {
    const onUploadingChange = vi.fn<(uploading: boolean) => void>();
    hook.uploads = [{ id: "up_1", file: pdf(), progress: 10, status: "uploading" }];
    const { unmount } = render(<Field onUploadingChange={onUploadingChange} />);
    expect(onUploadingChange).toHaveBeenLastCalledWith(true);

    unmount();

    expect(onUploadingChange).toHaveBeenLastCalledWith(false);
  });

  it("does not count a blocked file as still uploading", () => {
    const onUploadingChange = vi.fn<(uploading: boolean) => void>();
    hook.uploads = [{ id: "up_1", file: pdf(), progress: 100, status: "quarantined" }];
    render(<Field onUploadingChange={onUploadingChange} />);

    expect(onUploadingChange).toHaveBeenLastCalledWith(false);
    expect(onUploadingChange).not.toHaveBeenCalledWith(true);
  });

  it("shows a stored document's preview from the server, and its icon when there is none", () => {
    render(
      <Field
        value={{
          id: "doc_7",
          name: "cdl-front.jpg",
          size: 88_000,
          fileType: "image/jpeg",
          previewStatus: "Ready",
        }}
      />,
    );

    const preview = screen.getByAltText("cdl-front.jpg");
    expect(preview).toHaveAttribute("src", documentContentUrl("doc_7", "preview"));

    fireEvent.error(preview);

    expect(screen.queryByAltText("cdl-front.jpg")).not.toBeInTheDocument();
    expect(screen.getByText("cdl-front.jpg")).toBeInTheDocument();
  });

  it("shows a picked image from its own bytes while it uploads, and no picture for a PDF", () => {
    const createObjectURL = vi.fn(() => "blob:picked-1");
    const revokeObjectURL = vi.fn();
    const original = { create: URL.createObjectURL, revoke: URL.revokeObjectURL };
    URL.createObjectURL = createObjectURL;
    URL.revokeObjectURL = revokeObjectURL;
    onTestFinished(() => {
      URL.createObjectURL = original.create;
      URL.revokeObjectURL = original.revoke;
    });
    const photo = new File([new Uint8Array(512)], "medical-card.png", { type: "image/png" });
    hook.uploads = [{ id: "up_2", file: photo, progress: 40, status: "uploading" }];

    const { unmount } = render(<Field />);

    expect(screen.getByAltText("medical-card.png")).toHaveAttribute("src", "blob:picked-1");
    expect(createObjectURL).toHaveBeenCalledWith(photo);
    unmount();
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:picked-1");

    hook.uploads = [{ id: "up_3", file: pdf(), progress: 40, status: "uploading" }];
    render(<Field />);
    expect(screen.queryByAltText("med-card-2026.pdf")).not.toBeInTheDocument();
  });

  it("shows a spinner for a stored document whose preview is being drawn, and the image once it is", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    onTestFinished(() => {
      vi.useRealTimers();
    });
    documentService.getById
      .mockResolvedValueOnce({ id: "doc_8", previewStatus: "Pending" } as Document)
      .mockResolvedValueOnce({ id: "doc_8", previewStatus: "Ready" } as Document);
    render(
      <Field
        value={{
          id: "doc_8",
          name: "medical-card.pdf",
          size: 2048,
          fileType: "application/pdf",
          previewStatus: "Pending",
        }}
      />,
    );

    expect(screen.getByTitle("Generating thumbnail...")).toBeInTheDocument();
    expect(screen.queryByAltText("medical-card.pdf")).not.toBeInTheDocument();
    await waitFor(() => expect(documentService.getById).toHaveBeenCalledTimes(1));
    expect(documentService.getById).toHaveBeenCalledWith("doc_8", {
      signal: expect.any(AbortSignal),
    });

    await act(() => vi.advanceTimersByTimeAsync(PENDING_DOCUMENTS_REFETCH_MS));

    await waitFor(() =>
      expect(screen.getByAltText("medical-card.pdf")).toHaveAttribute(
        "src",
        documentContentUrl("doc_8", "preview"),
      ),
    );
    expect(documentService.getById).toHaveBeenCalledTimes(2);
    await act(() => vi.advanceTimersByTimeAsync(PENDING_DOCUMENTS_REFETCH_MS * 2));
    expect(documentService.getById).toHaveBeenCalledTimes(2);
  });

  it("asks the server nothing for a stored document whose preview is settled", () => {
    render(
      <Field
        value={{
          id: "doc_9",
          name: "notes.docx",
          size: 2048,
          fileType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
          previewStatus: "Unsupported",
        }}
      />,
    );

    expect(screen.queryByAltText("notes.docx")).not.toBeInTheDocument();
    expect(documentService.getById).not.toHaveBeenCalled();
  });

  it("tries the preview of a stored document whose read did not say where it stands", () => {
    render(
      <Field
        value={{
          id: "doc_10",
          name: "cdl-back.jpg",
          size: 0,
          fileType: "image/jpeg",
          previewStatus: null,
        }}
      />,
    );

    expect(screen.getByAltText("cdl-back.jpg")).toHaveAttribute(
      "src",
      documentContentUrl("doc_10", "preview"),
    );
  });
});
