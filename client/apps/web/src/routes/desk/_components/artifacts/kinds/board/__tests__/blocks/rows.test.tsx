// jsdom lays nothing out: these hold which part of a row's second line may give way and that the
// badge stays in the row; that the badge stays whole at the right edge on screen needs a browser.
import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { RecordRows, RecordRowSketch } from "../../board-parts";
import { READ_AT, shipmentRow, words } from "../fixtures";

afterEach(cleanup);

const draw = (rows: Parameters<typeof RecordRows>[0]["rows"]) =>
  render(<RecordRows rows={rows} now={READ_AT} open={vi.fn()} opensInPlace={() => true} />);

describe("a row's second fact", () => {
  it("lets a worded fact give way, whole in its title, and keeps the status badge", () => {
    const sentence =
      "On hold: Rate dispute, Bench: waiting on the broker's corrected rate confirmation";
    const base = shipmentRow();
    const { container } = draw([
      {
        ...base,
        subtitle: "Omar Haddad-Fitzgerald",
        status: "On hold",
        statusTone: "warn",
        measure: base.measure ? { ...base.measure, fact: words(sentence) } : null,
      },
    ]);
    const fact = container.querySelector(".dk-bd-line > .dk-bd-at");
    expect(fact?.className).toBe("dk-bd-at dk-bd-said");
    expect(fact?.getAttribute("title")).toBe(sentence);
    expect(fact?.querySelector(":scope > .dk-bd-words")?.textContent).toBe(sentence);
    expect(fact?.querySelector(".dk-bd-slot")).toBeNull();
    expect(container.querySelector(".dk-bd-line > .dk-bd-st")).not.toBeNull();
    expect(container.querySelector(".dk-bd-line > .dk-bd-sub")?.textContent).toBe(
      "Omar Haddad-Fitzgerald",
    );
  });

  it("keeps a moment in its slot, never cut, with the words around it free to give way", () => {
    const { container } = draw([shipmentRow()]);
    const fact = container.querySelector(".dk-bd-line > .dk-bd-at");
    expect(fact?.className).toBe("dk-bd-at");
    const words = [...(fact?.querySelectorAll(":scope > .dk-bd-words") ?? [])].map(
      (node) => node.textContent,
    );
    expect(words).toEqual(["Due "]);
    const slot = fact?.querySelector(":scope > .dk-bd-slot");
    expect(slot?.textContent).not.toBe("");
    expect(fact?.textContent).toBe(`Due ${slot?.textContent}`);
  });

  it("keeps an amount in its slot whole after the words", () => {
    const base = shipmentRow();
    const { container } = draw([
      {
        ...base,
        measure: base.measure
          ? {
              ...base.measure,
              fact: words("Deductions exceed earnings by {0}", {
                value: { kind: "money", number: "3338.70", currency: "USD", text: "" },
              }),
            }
          : null,
      },
    ]);
    const slot = container.querySelector(".dk-bd-at > .dk-bd-slot");
    expect(slot?.textContent).toBe("$3,338.70");
    expect(container.querySelector(".dk-bd-at > .dk-bd-words")?.textContent).toBe(
      "Deductions exceed earnings by ",
    );
  });

  it("treats a {0} slot holding text as words that give way, and keeps the badge", () => {
    const sentence =
      "Has a critical exception: Deductions exceed earnings; 3338.79 USD carries forward";
    const base = shipmentRow();
    const { container } = draw([
      {
        ...base,
        subtitle: "George Thompson",
        status: "On hold",
        statusTone: "warn",
        measure: base.measure
          ? {
              ...base.measure,
              fact: words("{0}", {
                value: { kind: "text", number: "0", currency: "", text: sentence },
              }),
            }
          : null,
      },
    ]);
    const fact = container.querySelector(".dk-bd-line > .dk-bd-at");
    expect(fact?.className).toBe("dk-bd-at dk-bd-said");
    expect(fact?.getAttribute("title")).toBe(sentence);
    expect(fact?.querySelector(":scope > .dk-bd-words")?.textContent).toBe(sentence);
    expect(fact?.querySelector(".dk-bd-slot")).toBeNull();
    expect(container.querySelector(".dk-bd-line > .dk-bd-st")).not.toBeNull();
    const sub = container.querySelector<HTMLElement>(".dk-bd-line > .dk-bd-sub");
    expect(sub?.className).toBe("dk-bd-sub");
    expect(sub?.getAttribute("style")).toBeNull();
  });

  it("keeps a {0} slot holding an amount or a moment whole", () => {
    const base = shipmentRow();
    const amount = draw([
      {
        ...base,
        measure: base.measure
          ? {
              ...base.measure,
              fact: words("{0}", {
                value: { kind: "money", number: "1844.20", currency: "USD", text: "" },
              }),
            }
          : null,
      },
    ]);
    expect(amount.container.querySelector(".dk-bd-at")?.className).toBe("dk-bd-at");
    expect(amount.container.querySelector(".dk-bd-at > .dk-bd-slot")?.textContent).toBe(
      "$1,844.20",
    );
    expect(amount.container.querySelector(".dk-bd-at > .dk-bd-words")).toBeNull();
    cleanup();
    const moment = draw([
      {
        ...base,
        measure: base.measure
          ? { ...base.measure, fact: words("{0}", { at: READ_AT - 7200 }) }
          : null,
      },
    ]);
    expect(moment.container.querySelector(".dk-bd-at > .dk-bd-slot")?.textContent).not.toBe("");
    expect(moment.container.querySelector(".dk-bd-at > .dk-bd-words")).toBeNull();
  });

  it("leaves the row's sketch as it was", () => {
    const { container } = render(<RecordRowSketch track />);
    expect(container.querySelector(".dk-bd-at")).toBeNull();
    expect(container.querySelector(".dk-bd-row > .dk-bd-track")).not.toBeNull();
  });
});
