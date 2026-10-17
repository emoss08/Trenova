import { ArtifactOpenerProvider } from "@/components/assistant/artifact-opener";
import type { AssistantMessageAttachment } from "@/types/assistant";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DeskMessageAttachments } from "../desk-uploads";

afterEach(cleanup);

const RATE_CON: AssistantMessageAttachment = {
  documentId: "doc_1",
  fileName: "rate-con.pdf",
  contentType: "application/pdf",
  fileSize: 2048,
};

describe("DeskMessageAttachments", () => {
  it("opens what was read off a sent file once its card exists", () => {
    const open = vi.fn();
    render(
      <ArtifactOpenerProvider onOpen={open} attachmentCards={new Map([["doc_1", "art_1"]])}>
        <DeskMessageAttachments attachments={[RATE_CON]} />
      </ArtifactOpenerProvider>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Open what was read from rate-con.pdf" }));

    expect(open).toHaveBeenCalledWith("art_1");
  });

  it("names a file with no card yet without offering to open it", () => {
    render(
      <ArtifactOpenerProvider onOpen={vi.fn()} attachmentCards={new Map()}>
        <DeskMessageAttachments attachments={[RATE_CON]} />
      </ArtifactOpenerProvider>,
    );

    expect(screen.getByText("rate-con.pdf")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("names the file where there is no pane to open a card in", () => {
    render(<DeskMessageAttachments attachments={[RATE_CON]} />);

    expect(screen.getByText("rate-con.pdf")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
