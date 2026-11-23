import { stubLayout, stubScrollTo } from "@/test/layout";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { BOUNDED_CELL_CLASS, BoundedTable, type BoundedRow } from "../bounded-table";

let restore: (() => void)[] = [];
beforeEach(() => {
  restore = [stubLayout(320), stubScrollTo()];
});
afterEach(() => {
  for (const undo of restore) undo();
});

const columns = [{ key: "name", label: "Name" }];

function focusableRows(count: number): BoundedRow[] {
  return Array.from({ length: count }, (_, index) => ({
    key: `row-${index}`,
    render: (props) => (
      <tr {...props} tabIndex={0} aria-label={`Entry ${index}`} data-glide-target>
        <td className={BOUNDED_CELL_CLASS}>Entry {index}</td>
      </tr>
    ),
  }));
}

function viewportOf(container: HTMLElement): HTMLElement {
  return container.querySelector('[data-slot="scroll-area-viewport"]') as HTMLElement;
}

async function scrollTo(viewport: HTMLElement, top: number) {
  await act(async () => {
    viewport.scrollTop = top;
    viewport.dispatchEvent(new Event("scroll"));
    await new Promise((resolve) => setTimeout(resolve, 20));
  });
}

describe("a long BoundedTable and keyboard focus", () => {
  it("keeps the focused row mounted, and focused, when it scrolls out of the window", async () => {
    const { container } = render(
      <BoundedTable
        label="Ledger"
        columns={columns}
        maxHeightClassName="max-h-88"
        rows={focusableRows(300)}
      />,
    );
    const first = screen.getByRole("row", { name: "Entry 0" });
    act(() => first.focus());

    await scrollTo(viewportOf(container), 10_000_000);

    expect(screen.getByText("Entry 299")).toBeInTheDocument();
    expect(screen.queryByText("Entry 1")).toBeNull();
    expect(screen.getByRole("row", { name: "Entry 0" })).toBe(document.activeElement);
  });

  it("moves focus down a row with the arrow key, scrolling in a row that was not mounted", async () => {
    render(
      <BoundedTable
        label="Ledger"
        columns={columns}
        maxHeightClassName="max-h-88"
        rows={focusableRows(300)}
      />,
    );
    const mounted = screen.getAllByRole("row").filter((row) => row.hasAttribute("data-index"));
    const last = mounted[mounted.length - 1];
    const lastIndex = Number(last.getAttribute("data-index"));
    act(() => last.focus());

    await act(async () => {
      fireEvent.keyDown(last, { key: "ArrowDown" });
      await new Promise((resolve) => setTimeout(resolve, 50));
    });

    expect(document.activeElement).toBe(
      screen.getByRole("row", { name: `Entry ${lastIndex + 1}` }),
    );
  });

  it("ends a capped list with the line that says only the newest are shown", () => {
    render(
      <BoundedTable
        label="Credentials"
        columns={columns}
        maxHeightClassName="max-h-88"
        rows={focusableRows(100)}
        cap={100}
      />,
    );

    expect(screen.getByText("Showing the newest 100")).toBeInTheDocument();
  });

  it("draws no cap line under the cap", () => {
    render(
      <BoundedTable
        label="Credentials"
        columns={columns}
        maxHeightClassName="max-h-88"
        rows={focusableRows(99)}
        cap={100}
      />,
    );

    expect(screen.queryByText(/Showing the newest/)).toBeNull();
  });
});
