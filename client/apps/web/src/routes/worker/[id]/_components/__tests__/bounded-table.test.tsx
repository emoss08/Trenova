import { fireEvent, render, screen } from "@testing-library/react";
import { stubLayout } from "@/test/layout";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  BOUNDED_CELL_CLASS,
  BOUNDED_VIRTUAL_FROM,
  BoundedTable,
  type BoundedMore,
  type BoundedRow,
  type BoundedTableHandle,
} from "../bounded-table";
import { useEffect, useRef } from "react";

const columns = [{ key: "name", label: "Name" }];

function rowsOf(count: number): BoundedRow[] {
  return Array.from({ length: count }, (_, index) => ({
    key: `row-${index}`,
    render: (props) => (
      <tr {...props} data-glide-target>
        <td className={BOUNDED_CELL_CLASS}>Entry {index}</td>
      </tr>
    ),
  }));
}

const settled: BoundedMore = {
  hasMore: false,
  loadMore: () => undefined,
  loadingMore: false,
  loadMoreFailed: false,
  error: null,
};

// jsdom lays nothing out; a windowed body needs a size to open its window.
let restoreLayout = () => {};
beforeEach(() => {
  restoreLayout = stubLayout(320);
});
afterEach(() => restoreLayout());

describe("BoundedTable", () => {
  it("draws a short table whole, with no spacer rows", () => {
    render(
      <BoundedTable
        label="Ledger"
        columns={columns}
        maxHeightClassName="max-h-88"
        rows={rowsOf(5)}
      />,
    );

    expect(screen.getAllByRole("row")).toHaveLength(6);
    expect(screen.getByText("Entry 4")).toBeInTheDocument();
  });

  it("mounts only the rows near the view once the table is long", () => {
    render(
      <BoundedTable
        label="Ledger"
        columns={columns}
        maxHeightClassName="max-h-88"
        rows={rowsOf(500)}
        more={settled}
      />,
    );

    expect(screen.getByText("Entry 0")).toBeInTheDocument();
    expect(screen.queryByText("Entry 499")).toBeNull();
    expect(screen.getAllByRole("row").length).toBeLessThan(80);
    expect(BOUNDED_VIRTUAL_FROM).toBe(30);
  });

  it("holds a loading line at the end while the next page is read", () => {
    render(
      <BoundedTable
        label="Ledger"
        columns={columns}
        maxHeightClassName="max-h-88"
        rows={rowsOf(3)}
        more={{ ...settled, hasMore: true, loadingMore: true }}
      />,
    );

    expect(screen.getByLabelText("Loading more")).toHaveAttribute("aria-busy", "true");
  });

  it("says a page failed and asks for it again on retry", () => {
    const loadMore = vi.fn();
    render(
      <BoundedTable
        label="Ledger"
        columns={columns}
        maxHeightClassName="max-h-88"
        rows={rowsOf(3)}
        more={{ ...settled, hasMore: true, loadMoreFailed: true, loadMore, error: new Error("x") }}
      />,
    );

    expect(screen.getByText("More could not be loaded")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /try again/i }));
    expect(loadMore).toHaveBeenCalledTimes(1);
  });

  it("scrolls a short table to a row on the first commit, as a deep link asks", () => {
    const scrolled: string[] = [];
    const original = Element.prototype.scrollIntoView;
    Element.prototype.scrollIntoView = function scrollIntoView(this: Element) {
      scrolled.push(this.getAttribute("data-row-key") ?? "");
    };
    function DeepLinked() {
      const handle = useRef<BoundedTableHandle>(null);
      useEffect(() => {
        handle.current?.scrollToKey("row-3");
      }, []);
      return (
        <BoundedTable
          label="Ledger"
          columns={columns}
          maxHeightClassName="max-h-88"
          handleRef={handle}
          rows={rowsOf(5)}
        />
      );
    }
    try {
      render(<DeepLinked />);
      expect(scrolled).toEqual(["row-3"]);
    } finally {
      Element.prototype.scrollIntoView = original;
    }
  });
});
