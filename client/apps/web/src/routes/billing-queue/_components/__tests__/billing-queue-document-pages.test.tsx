import type { PageImageSource } from "@/lib/queries/document-page";
import { stubLayout, stubScrollTo } from "@/test/layout";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BillingQueueDocumentPages, pageAtOffset } from "../billing-queue-document-pages";

const VIEW_HEIGHT = 800;

let restore: Array<() => void> = [];
const fetchPage = vi.fn(async (page: number) => new Blob([String(page)], { type: "image/webp" }));

beforeEach(() => {
  restore = [stubLayout(VIEW_HEIGHT, 600), stubScrollTo()];
});

afterEach(() => {
  for (const undo of restore) undo();
  cleanup();
  vi.clearAllMocks();
});

const SOURCE: PageImageSource = {
  image: (page) => ({
    queryKey: ["test-pages", page],
    queryFn: () => fetchPage(page),
  }),
  downloadEndpoint: "/files/packet/",
};

function Viewer({ pageCount, onPage }: { pageCount: number; onPage?: (page: number) => void }) {
  const [page, setPage] = useState(1);
  return (
    <BillingQueueDocumentPages
      source={SOURCE}
      fileName="packet.pdf"
      pageCount={pageCount}
      page={page}
      onPage={(next) => {
        onPage?.(next);
        setPage(next);
      }}
    />
  );
}

function renderViewer(pageCount: number, onPage?: (page: number) => void) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <Viewer pageCount={pageCount} onPage={onPage} />
    </QueryClientProvider>,
  );
}

function fetchedPages(): number[] {
  return fetchPage.mock.calls.map(([page]) => page);
}

function column(): HTMLElement {
  return screen.getByRole("region", { name: /Page Up and Page Down turn pages/ });
}

describe("BillingQueueDocumentPages", () => {
  it("fetches only the pages near the viewport of a hundred-page file", async () => {
    renderViewer(100);

    expect(await screen.findByAltText("Page 1 of packet.pdf")).toBeInTheDocument();
    await waitFor(() => expect(fetchedPages()).toContain(1));
    expect(fetchedPages().length).toBeLessThan(10);
    expect(fetchedPages()).not.toContain(100);
    expect(screen.queryByRole("button", { name: "Page 100" })).not.toBeInTheDocument();
  });

  it("fetches a page once for both its thumbnail and the column", async () => {
    renderViewer(3);

    await screen.findByAltText("Page 1 of packet.pdf");
    await screen.findByRole("button", { name: "Page 1" });

    await waitFor(() => expect(fetchedPages().filter((page) => page === 1)).toHaveLength(1));
  });

  it("goes to a page clicked in the rail and outlines its thumbnail", async () => {
    const onPage = vi.fn();
    renderViewer(5, onPage);
    await screen.findByAltText("Page 1 of packet.pdf");

    fireEvent.click(screen.getByRole("button", { name: "Page 3" }));

    expect(onPage).toHaveBeenLastCalledWith(3);
    expect(screen.getByText("3 of 5")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Page 3" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("button", { name: "Page 1" })).not.toHaveAttribute("aria-current");
    await waitFor(() => expect(column().scrollTop).toBeGreaterThan(0));
  });

  it("moves through the thumbnails with the arrow keys, Home and End", async () => {
    renderViewer(5);
    const first = await screen.findByRole("button", { name: "Page 1" });
    first.focus();

    fireEvent.keyDown(first, { key: "ArrowDown" });
    await waitFor(() => expect(screen.getByRole("button", { name: "Page 2" })).toHaveFocus());
    expect(screen.getByText("2 of 5")).toBeInTheDocument();

    fireEvent.keyDown(screen.getByRole("button", { name: "Page 2" }), { key: "End" });
    await waitFor(() => expect(screen.getByRole("button", { name: "Page 5" })).toHaveFocus());
    expect(screen.getByText("5 of 5")).toBeInTheDocument();
  });

  it("follows the page being read as the column is scrolled", async () => {
    const onPage = vi.fn();
    renderViewer(5, onPage);
    await screen.findByAltText("Page 1 of packet.pdf");
    const well = column();

    act(() => {
      fireEvent.wheel(well);
      well.scrollTop = VIEW_HEIGHT * 2;
      fireEvent.scroll(well);
    });

    expect(onPage).toHaveBeenLastCalledWith(3);
    expect(screen.getByText("3 of 5")).toBeInTheDocument();
  });

  it("keeps a page turned to as current while the column scrolls to it", async () => {
    const onPage = vi.fn();
    renderViewer(5, onPage);
    await screen.findByAltText("Page 1 of packet.pdf");

    fireEvent.click(screen.getByRole("button", { name: "Page 5" }));
    const well = column();
    act(() => {
      well.scrollTop = VIEW_HEIGHT * 3;
      fireEvent.scroll(well);
    });

    expect(onPage).toHaveBeenLastCalledWith(5);
    expect(screen.getByText("5 of 5")).toBeInTheDocument();
  });

  it("turns one page at a time with Page Down and Page Up", async () => {
    const onPage = vi.fn();
    renderViewer(5, onPage);
    await screen.findByAltText("Page 1 of packet.pdf");

    fireEvent.keyDown(column(), { key: "PageDown" });
    fireEvent.keyDown(column(), { key: "PageDown" });
    fireEvent.keyDown(column(), { key: "PageUp" });

    expect(onPage.mock.calls.map(([page]) => page)).toEqual([2, 3, 2]);
  });
});

describe("BillingQueueDocumentPages, toolbar", () => {
  it("goes to a page typed into the page count", async () => {
    const onPage = vi.fn();
    renderViewer(100, onPage);
    await screen.findByAltText("Page 1 of packet.pdf");

    fireEvent.click(screen.getByRole("button", { name: "Page 1 of 100. Go to a page." }));
    const field = screen.getByRole("textbox", { name: "Go to page" });
    fireEvent.change(field, { target: { value: "57" } });
    fireEvent.keyDown(field, { key: "Enter" });

    expect(onPage).toHaveBeenLastCalledWith(57);
    expect(screen.getByText("57 of 100")).toBeInTheDocument();
    expect(screen.queryByRole("textbox", { name: "Go to page" })).not.toBeInTheDocument();
  });

  it("goes no further than the last page and leaves the field on Escape", async () => {
    const onPage = vi.fn();
    renderViewer(5, onPage);
    await screen.findByAltText("Page 1 of packet.pdf");

    fireEvent.click(screen.getByRole("button", { name: "Page 1 of 5. Go to a page." }));
    let field = screen.getByRole("textbox", { name: "Go to page" });
    fireEvent.change(field, { target: { value: "90" } });
    fireEvent.keyDown(field, { key: "Enter" });
    expect(onPage).toHaveBeenLastCalledWith(5);

    fireEvent.click(screen.getByRole("button", { name: "Page 5 of 5. Go to a page." }));
    field = screen.getByRole("textbox", { name: "Go to page" });
    fireEvent.change(field, { target: { value: "2" } });
    fireEvent.keyDown(field, { key: "Escape" });
    expect(onPage).toHaveBeenCalledTimes(1);
  });

  it("opens fitted to the width and steps to a scale with plus and minus", async () => {
    renderViewer(3);
    await screen.findByAltText("Page 1 of packet.pdf");

    expect(screen.getByRole("button", { name: "Fit width" })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(screen.getByText("Fit width", { selector: "span" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Fit page" }));
    expect(screen.getByRole("button", { name: "Fit page" })).toHaveAttribute(
      "aria-pressed",
      "true",
    );

    fireEvent.click(screen.getByRole("button", { name: "Zoom in" }));
    expect(screen.getByText(/^\d+%$/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Fit width" })).toHaveAttribute(
      "aria-pressed",
      "false",
    );

    fireEvent.keyDown(column(), { key: "0" });
    expect(screen.getByRole("button", { name: "Fit width" })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
  });

  it("zooms with Ctrl and the scroll wheel", async () => {
    renderViewer(3);
    await screen.findByAltText("Page 1 of packet.pdf");

    fireEvent.wheel(column(), { ctrlKey: true, deltaY: -100 });

    expect(screen.getByText(/^\d+%$/)).toBeInTheDocument();
  });

  it("turns one page on its own and every page together", async () => {
    renderViewer(3);
    const first = await screen.findByAltText("Page 1 of packet.pdf");
    const second = await screen.findByAltText("Page 2 of packet.pdf");
    const turn = (image: HTMLElement) => image.parentElement?.style.transform ?? "";

    fireEvent.click(screen.getByRole("button", { name: "Rotate page 2" }));
    expect(turn(second)).toContain("rotate(90deg)");
    expect(turn(first)).not.toContain("rotate(");

    fireEvent.click(screen.getByRole("button", { name: "Rotate all pages" }));
    expect(turn(first)).toContain("rotate(90deg)");
    expect(turn(second)).toContain("rotate(180deg)");
  });
});

describe("pageAtOffset", () => {
  const rows = [
    { start: 24, end: 824 },
    { start: 824, end: 1624 },
    { start: 1624, end: 2424 },
  ];

  it("is the page whose row crosses the offset", () => {
    expect(pageAtOffset(rows, 900)).toBe(2);
    expect(pageAtOffset(rows, 1624)).toBe(3);
  });

  it("is the first page above the first row and the last below the last", () => {
    expect(pageAtOffset(rows, 0)).toBe(1);
    expect(pageAtOffset(rows, 9000)).toBe(3);
  });
});
