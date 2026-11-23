import { documentContentUrl } from "@/services/document";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { DocumentPanePreview, type PaneDocument } from "../document-pane-preview";

const license: PaneDocument = {
  id: "doc_cdl",
  originalName: "01_BRF-260184_commercial-drivers-license-front.jpg",
  fileType: "image/jpeg",
  fileSize: 421_888,
  previewStatus: "Ready",
};

describe("DocumentPanePreview", () => {
  it("shows the server's preview, the whole file name, its size and a way to open it", () => {
    render(<DocumentPanePreview document={license} />);

    expect(screen.getByAltText(license.originalName)).toHaveAttribute(
      "src",
      documentContentUrl("doc_cdl", "preview"),
    );
    expect(screen.getByText(license.originalName)).toBeInTheDocument();
    expect(screen.getByText("412 KB")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open the document" })).toHaveAttribute(
      "href",
      documentContentUrl("doc_cdl", "view"),
    );
  });

  it("opens the document from the preview the same way as the link, in a new tab", () => {
    render(<DocumentPanePreview document={license} />);

    const preview = screen.getByRole("link", { name: `Open ${license.originalName}` });
    expect(preview).toHaveAttribute("href", documentContentUrl("doc_cdl", "view"));
    expect(preview).toHaveAttribute("target", "_blank");
    expect(preview).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("falls back to the file's icon when the preview fails to load, never a broken image", () => {
    render(<DocumentPanePreview document={license} />);

    fireEvent.error(screen.getByAltText(license.originalName));

    expect(screen.queryByAltText(license.originalName)).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: `Open ${license.originalName}` })).toBeInTheDocument();
  });

  it("shows that the preview is being drawn and asks for no image yet", () => {
    render(<DocumentPanePreview document={{ ...license, previewStatus: "Pending" }} />);

    expect(screen.queryByAltText(license.originalName)).not.toBeInTheDocument();
    expect(screen.getByTitle("Generating thumbnail...")).toBeInTheDocument();
  });

  it("asks for no preview of a kind of file the server cannot draw", () => {
    render(
      <DocumentPanePreview
        document={{
          id: "doc_letter",
          originalName: "employment-verification.docx",
          fileType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
          fileSize: 18_000,
          previewStatus: "Unsupported",
        }}
      />,
    );

    expect(screen.queryByAltText("employment-verification.docx")).not.toBeInTheDocument();
    expect(screen.getByText("employment-verification.docx")).toBeInTheDocument();
  });

  it("draws no image of a preview that failed, only the file's icon", () => {
    render(<DocumentPanePreview document={{ ...license, previewStatus: "Failed" }} />);

    expect(screen.queryByAltText(license.originalName)).not.toBeInTheDocument();
    expect(screen.getByTitle("Preview unavailable")).toBeInTheDocument();
  });

  it("swaps the spinner for the image once the read says the preview is ready", () => {
    const { rerender } = render(
      <DocumentPanePreview document={{ ...license, previewStatus: "Pending" }} />,
    );
    expect(screen.queryByAltText(license.originalName)).not.toBeInTheDocument();

    rerender(<DocumentPanePreview document={license} />);

    expect(screen.queryByTitle("Generating thumbnail...")).not.toBeInTheDocument();
    expect(screen.getByAltText(license.originalName)).toHaveAttribute(
      "src",
      documentContentUrl("doc_cdl", "preview"),
    );
  });

  it("leaves the open link to a pane whose footer opens the document, and an unknown size off", () => {
    render(<DocumentPanePreview document={{ ...license, fileSize: 0 }} openLink={false} />);

    expect(screen.queryByRole("link", { name: "Open the document" })).not.toBeInTheDocument();
    expect(screen.queryByText("0 B")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: `Open ${license.originalName}` })).toBeInTheDocument();
  });

  it("says why there is no document, and draws nothing when given nothing to say", () => {
    const { container, rerender } = render(
      <DocumentPanePreview document={null} empty="No document attached." />,
    );

    expect(screen.getByText("No document attached.")).toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();

    rerender(<DocumentPanePreview document={null} />);
    expect(container).toBeEmptyDOMElement();
  });
});
