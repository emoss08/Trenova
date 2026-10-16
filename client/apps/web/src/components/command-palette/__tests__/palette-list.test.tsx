import { render } from "@testing-library/react";
import { Command } from "cmdk";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { PaletteItem, PaletteSection } from "../palette-model";
import { PaletteList } from "../palette-list";
import { command, page } from "./palette-test-fixtures";

const ROW_HEIGHT = 46;

function pageRow(id: string): PaletteItem {
  return { kind: "page", key: `page:/${id}`, page: page({ id, title: id, href: `/${id}` }) };
}

function commandRow(id: string): PaletteItem {
  return { kind: "command", key: `command:${id}`, command: command({ id, label: id }) };
}

function listOf(id: string, items: PaletteItem[]): PaletteSection[] {
  return [{ id, heading: id, items }];
}

const PAGES = listOf("pages", [pageRow("invoices"), pageRow("workers")]);
const COMMANDS = listOf("commands", [commandRow("new-shipment"), commandRow("sign-out")]);

function renderList(sections: PaletteSection[], selectedKey: string) {
  const list = (next: PaletteSection[], key: string) => (
    <Command shouldFilter={false} value={key}>
      <PaletteList
        sections={next}
        selectedKey={key}
        onSelect={vi.fn()}
        onRetry={vi.fn()}
        onSeeAll={vi.fn()}
      />
    </Command>
  );
  const view = render(list(sections, selectedKey));
  const fill = document.querySelector<HTMLElement>("[cmdk-list-sizer] > [aria-hidden]")!;
  const styles = new MutationObserver(() => undefined);
  styles.observe(fill, { attributes: true, attributeFilter: ["style"], attributeOldValue: true });
  return {
    fill,
    /** Moves to another list or row, and says every style the fill passed through on the way. */
    show(next: PaletteSection[], key: string): string[] {
      styles.takeRecords();
      view.rerender(list(next, key));
      return [
        ...styles.takeRecords().map((record) => record.oldValue ?? ""),
        fill.getAttribute("style") ?? "",
      ];
    },
  };
}

beforeEach(() => {
  vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(
    function (this: HTMLElement) {
      const rows = [...document.querySelectorAll("[data-palette-row]")];
      const index = rows.indexOf(this);
      return index < 0
        ? new DOMRect(0, 0, 0, 0)
        : new DOMRect(0, index * ROW_HEIGHT, 100, ROW_HEIGHT);
    },
  );
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("PaletteList selection fill", () => {
  it("sits under the selected row", () => {
    const { fill } = renderList(PAGES, "page:/workers");

    expect(fill.style.opacity).toBe("1");
    expect(fill.style.transform).toBe(`translateY(${ROW_HEIGHT}px)`);
    expect(fill.style.height).toBe(`${ROW_HEIGHT}px`);
  });

  it("lands under the first row of a new list in the commit that draws it, without sliding", () => {
    const { fill, show } = renderList(PAGES, "page:/workers");

    const styles = show(COMMANDS, "command:new-shipment");

    expect(fill.style.opacity).toBe("1");
    expect(fill.style.transform).toBe("translateY(0px)");
    expect(styles.some((style) => style.includes("transition: none"))).toBe(true);
    expect(fill.style.transition).toBe("");
  });

  it("slides when the selection moves through a list that stayed put", () => {
    const { fill, show } = renderList(PAGES, "page:/workers");

    const styles = show(PAGES, "page:/invoices");

    expect(fill.style.transform).toBe("translateY(0px)");
    expect(styles.some((style) => style.includes("transition: none"))).toBe(false);
  });

  it("follows its row when the list changes around it, without sliding", () => {
    const { fill, show } = renderList(PAGES, "page:/workers");

    const styles = show(
      listOf("pages", [pageRow("billing"), pageRow("invoices"), pageRow("workers")]),
      "page:/workers",
    );

    expect(fill.style.transform).toBe(`translateY(${ROW_HEIGHT * 2}px)`);
    expect(styles.some((style) => style.includes("transition: none"))).toBe(true);
  });

  it("hides when no row is selected", () => {
    const { fill, show } = renderList(PAGES, "page:/workers");

    show(PAGES, "");

    expect(fill.style.opacity).toBe("0");
  });
});
