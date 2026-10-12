import type { ReplyFile } from "@/lib/triage/reply-files";
import type { UploadState } from "@/types/upload";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ReplyFiles, type RefusedUpload } from "../steps/reply-files";

const KB = 1024;
const MB = 1024 * KB;

function file(documentId: string, fileName: string, byteSize: number): ReplyFile {
  return { documentId, fileName, contentType: "application/pdf", byteSize };
}

function upload(id: string, name: string, status: UploadState["status"]): UploadState {
  return {
    id,
    file: new File(["x"], name, { type: "application/pdf" }),
    progress: status === "success" ? 100 : 40,
    status,
    error: status === "error" ? "The server refused the file" : undefined,
  };
}

const RATE_CON = file("doc_1", "rate-confirmation.pdf", 512 * KB);
const POD = file("doc_2", "proof-of-delivery.pdf", MB);

function renderFiles({
  files = [RATE_CON, POD],
  uploads = [],
  refused = [],
  limitBytes = 10 * MB,
}: {
  files?: readonly ReplyFile[];
  uploads?: readonly UploadState[];
  refused?: readonly RefusedUpload[];
  limitBytes?: number | null;
} = {}) {
  const props = {
    files,
    uploads,
    refused,
    limitBytes,
    onRemove: vi.fn(),
    onCancelUpload: vi.fn(),
    onRetryUpload: vi.fn(),
    onDismissRefusal: vi.fn(),
  };
  return { ...render(<ReplyFiles {...props} />), props };
}

describe("ReplyFiles", () => {
  it("folds files that have all landed into one line saying how many and how much", () => {
    renderFiles();

    const toggle = screen.getByRole("button", { name: "2 files · 1.5 MB of 10.0 MB" });
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByRole("list", { name: "Files" })).not.toBeInTheDocument();
    expect(screen.queryByText("rate-confirmation.pdf")).not.toBeInTheDocument();
  });

  it("names only the total when the email has no limit to measure against", () => {
    renderFiles({ limitBytes: null });

    expect(screen.getByRole("button", { name: "2 files · 1.5 MB" })).toBeInTheDocument();
  });

  it("opens the folded line to list each file with a way to take it off, and folds it again", async () => {
    const user = userEvent.setup();
    const { props } = renderFiles();
    const toggle = screen.getByRole("button", { name: "2 files · 1.5 MB of 10.0 MB" });

    await user.click(toggle);

    expect(toggle).toHaveAttribute("aria-expanded", "true");
    const items = screen.getAllByRole("listitem");
    expect(items.map((item) => item.getAttribute("title"))).toEqual([
      "rate-confirmation.pdf",
      "proof-of-delivery.pdf",
    ]);
    await user.click(screen.getByRole("button", { name: "Remove proof-of-delivery.pdf" }));
    expect(props.onRemove).toHaveBeenCalledWith("doc_2");

    await user.click(toggle);
    expect(screen.queryByRole("list", { name: "Files" })).not.toBeInTheDocument();
  });

  it("lists every file while an upload is still running", () => {
    renderFiles({ uploads: [upload("up_1", "bol.pdf", "uploading")] });

    expect(screen.queryByRole("button", { expanded: false })).not.toBeInTheDocument();
    expect(screen.getAllByRole("listitem").map((item) => item.getAttribute("title"))).toEqual([
      "rate-confirmation.pdf",
      "proof-of-delivery.pdf",
      "bol.pdf",
    ]);
    expect(screen.getByText("2 files · 1.5 MB of 10.0 MB")).toBeInTheDocument();
  });

  it("folds once the running upload lands as a file", () => {
    const landed = file("doc_3", "bol.pdf", 512 * KB);
    const { rerender, props } = renderFiles({
      uploads: [upload("up_1", "bol.pdf", "uploading")],
    });
    expect(screen.getAllByRole("listitem")).toHaveLength(3);

    rerender(
      <ReplyFiles
        {...props}
        files={[RATE_CON, POD, landed]}
        uploads={[upload("up_1", "bol.pdf", "success")]}
      />,
    );

    expect(screen.queryByRole("listitem")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "3 files · 2.0 MB of 10.0 MB" })).toHaveAttribute(
      "aria-expanded",
      "false",
    );
  });

  it("keeps a failed upload listed with its reason until it is retried or removed", () => {
    renderFiles({ uploads: [upload("up_1", "bol.pdf", "error")] });

    expect(screen.getByText("The server refused the file")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument();
  });

  it("keeps a refused file listed with why it would not fit", () => {
    renderFiles({
      refused: [{ key: "scan.pdf:9:1", name: "scan.pdf", reason: "It is 9.0 MB." }],
    });

    expect(screen.getByText("It is 9.0 MB.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Remove scan.pdf" })).toBeInTheDocument();
  });

  it("lists every file when together they are more than the email can carry", () => {
    renderFiles({ limitBytes: MB });

    expect(screen.getAllByRole("listitem")).toHaveLength(2);
    expect(screen.getByText("2 files · 1.5 MB of 1.0 MB")).toBeInTheDocument();
  });

  it("lists an upload on its own before any file has landed, with no count", () => {
    renderFiles({ files: [], uploads: [upload("up_1", "bol.pdf", "uploading")] });

    expect(screen.getAllByRole("listitem")).toHaveLength(1);
    expect(screen.queryByText(/files? ·/)).not.toBeInTheDocument();
  });

  it("draws nothing when the reply carries no files", () => {
    const { container } = renderFiles({ files: [], uploads: [upload("up_1", "a.pdf", "success")] });

    expect(container).toBeEmptyDOMElement();
  });
});
