import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PdfFrame } from "../pdf-frame";

const mocks = vi.hoisted(() => ({ download: vi.fn() }));

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...actual, api: { ...actual.api, download: mocks.download } };
});

const ENDPOINT = "/documents/doc_1/view/";

let made: Blob[];

function renderFrame(props: Partial<Parameters<typeof PdfFrame>[0]> = {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <PdfFrame endpoint={ENDPOINT} title="Rate confirmation" {...props} />
    </QueryClientProvider>,
  );
}

function served(body: string, type: string) {
  mocks.download.mockResolvedValue({ blob: new Blob([body], { type }), fileName: "file" });
}

beforeEach(() => {
  made = [];
  vi.spyOn(URL, "createObjectURL").mockImplementation((blob: Blob | MediaSource) => {
    made.push(blob as Blob);
    return `blob:app/${made.length}`;
  });
  vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  mocks.download.mockReset();
});

describe("PdfFrame", () => {
  it("fetches the file through the app's own request helper and shows a busy outline", () => {
    mocks.download.mockImplementation(() => new Promise<never>(() => {}));
    renderFrame();

    expect(screen.getByLabelText("Loading the document")).toHaveAttribute("aria-busy", "true");
    expect(mocks.download).toHaveBeenCalledWith(ENDPOINT, "document.pdf", {
      signal: expect.any(AbortSignal),
    });
  });

  it("opens the file in the browser's viewer at the asked page, named for screen readers", async () => {
    served("%PDF-1.7", "application/pdf");
    renderFrame({ page: 3 });

    const frame = await screen.findByTitle("Rate confirmation");
    expect(frame.tagName).toBe("IFRAME");
    expect(frame).toHaveAttribute("src", "blob:app/1#page=3");
  });

  // A blob URL is the app's own origin: a file the server labelled as HTML must still only
  // ever reach the PDF viewer, never render as a page of the app.
  it("types the file as a PDF whatever the response said", async () => {
    served("<script>alert(1)</script>", "text/html");
    renderFrame();

    await screen.findByTitle("Rate confirmation");
    expect(made.at(-1)?.type).toBe("application/pdf");
  });

  it("says the file could not be shown and fetches it again on request", async () => {
    mocks.download.mockRejectedValueOnce(new Error("offline"));
    renderFrame();

    const retry = await screen.findByRole("button", { name: "Try again" });
    expect(screen.getByText("This document could not be shown.")).toBeInTheDocument();

    served("%PDF-1.7", "application/pdf");
    fireEvent.click(retry);
    await screen.findByTitle("Rate confirmation");
    expect(mocks.download).toHaveBeenCalledTimes(2);
  });

  it("offers the file to open where the browser has no built-in viewer", async () => {
    served("%PDF-1.7", "application/pdf");
    Object.defineProperty(navigator, "pdfViewerEnabled", { configurable: true, get: () => false });
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    try {
      renderFrame();

      fireEvent.click(await screen.findByRole("button", { name: "Open the PDF" }));
      expect(open).toHaveBeenCalledWith("blob:app/1", "_blank");
      expect(screen.queryByTitle("Rate confirmation")).not.toBeInTheDocument();
    } finally {
      Reflect.deleteProperty(navigator, "pdfViewerEnabled");
    }
  });

  it("turns the frame a quarter at a time, swapping its sides to fill the box", async () => {
    served("%PDF-1.7", "application/pdf");
    vi.spyOn(HTMLElement.prototype, "clientWidth", "get").mockReturnValue(800);
    vi.spyOn(HTMLElement.prototype, "clientHeight", "get").mockReturnValue(600);
    renderFrame({ rotate: 90 });

    const frame = await screen.findByTitle("Rate confirmation");
    await waitFor(() => expect(frame.style.width).toBe("600px"));
    expect(frame.style.height).toBe("800px");
    expect(frame.style.transform).toBe("translate(-50%, -50%) rotate(90deg)");
  });
});
