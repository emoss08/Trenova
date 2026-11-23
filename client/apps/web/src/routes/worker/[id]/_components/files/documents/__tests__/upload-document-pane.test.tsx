import { WorkPaneHost, WorkPaneProvider } from "@/components/work-pane";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { UploadDocumentPane } from "../upload-document-pane";

const uploader = {
  uploads: [],
  uploadFiles: vi.fn(),
  cancelUpload: vi.fn(),
  retryUpload: vi.fn(),
  removeUpload: vi.fn(),
  clearCompleted: vi.fn(),
  clearAll: vi.fn(),
  isUploading: false,
};

function renderPane(droppedFiles: File[] = []) {
  const onDirtyChange = vi.fn();
  render(
    <WorkPaneProvider onDirtyChange={onDirtyChange}>
      <WorkPaneHost>
        <UploadDocumentPane
          open
          onOpenChange={vi.fn()}
          workerName="Maria"
          lineageId={null}
          canUpload
          droppedFiles={droppedFiles}
          uploader={uploader as never}
          onDone={vi.fn()}
        />
      </WorkPaneHost>
    </WorkPaneProvider>,
  );
  return onDirtyChange;
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("UploadDocumentPane", () => {
  it("counts files chosen and not uploaded as unsaved work", async () => {
    const onDirtyChange = renderPane();
    await screen.findByRole("group", { name: "Files to upload" });
    expect(onDirtyChange).not.toHaveBeenLastCalledWith(true);

    const input = document.querySelector('input[type="file"]') as HTMLInputElement;
    const file = new File(["x"], "med-card.pdf", { type: "application/pdf" });
    fireEvent.change(input, { target: { files: [file] } });

    expect(await screen.findByText("med-card.pdf")).toBeTruthy();
    expect(onDirtyChange).toHaveBeenLastCalledWith(true);
  });

  it("opens with the files dropped on the table already chosen", async () => {
    const file = new File(["x"], "dropped.pdf", { type: "application/pdf" });
    const onDirtyChange = renderPane([file]);

    expect(await screen.findByText("dropped.pdf")).toBeTruthy();
    expect(onDirtyChange).toHaveBeenLastCalledWith(true);
  });

  it("puts the refusal under the drop zone, not pulled up over it", async () => {
    renderPane();
    await screen.findByRole("group", { name: "Files to upload" });

    const input = document.querySelector('input[type="file"]') as HTMLInputElement;
    fireEvent.change(input, {
      target: { files: [new File(["x"], "setup.exe", { type: "application/x-msdownload" })] },
    });

    const alert = await screen.findByRole("alert");
    expect(alert.className).not.toMatch(/-mt-/);
  });
});
