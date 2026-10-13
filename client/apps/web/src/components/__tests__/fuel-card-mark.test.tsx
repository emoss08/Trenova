import { existsSync } from "node:fs";
import { resolve } from "node:path";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { FuelCardMark } from "../fuel-card-mark";

afterEach(cleanup);

const PUBLIC_DIR = resolve(__dirname, "../../../public");

function markImage(card: HTMLElement): Element | null {
  return card.querySelector("svg image");
}

describe("fuel card mark", () => {
  it.each([
    ["WEX", "/integrations/logos/wex-logo-dark.svg", "#d00a2c"],
    ["Comdata", "/integrations/logos/comdata-logo-dark.png", "#0b2a45"],
    ["EFS", "/integrations/logos/efs-logo-light.svg", "#ffffff"],
  ] as const)("draws %s cards in the network's colour with its logo", (provider, logo, colour) => {
    render(<FuelCardMark provider={provider} />);

    const card = screen.getByRole("img", { name: provider });
    expect(card.style.backgroundColor.toLowerCase()).toBe(colour);
    expect(markImage(card)?.getAttribute("href")).toBe(logo);
  });

  it("ships every network logo it draws", () => {
    const { container } = render(
      <>
        <FuelCardMark provider="WEX" />
        <FuelCardMark provider="Comdata" />
        <FuelCardMark provider="EFS" />
      </>,
    );

    const hrefs = Array.from(container.querySelectorAll("svg image"), (image) =>
      image.getAttribute("href"),
    );
    expect(hrefs).toHaveLength(3);
    for (const href of hrefs) {
      expect(existsSync(`${PUBLIC_DIR}${href}`), `${href} is missing from public/`).toBe(true);
    }
  });

  // Rows and buttons resize any svg without a size class, and a letterboxed
  // crop shows what lies outside it, so the mark carries its own proportions.
  it("draws the mark at its crop's proportions whatever row it sits in", () => {
    render(<FuelCardMark provider="Comdata" />);

    const svg = screen.getByRole("img", { name: "Comdata" }).querySelector("svg");
    expect(svg?.getAttribute("viewBox")).toBe("19 26 286 58");
    const style = svg?.getAttribute("style") ?? "";
    expect(style).toContain("width: 100%");
    expect(style).toContain("height: auto");
    expect(style).toContain("aspect-ratio: 286 / 58");
  });

  it("gives a card on another network a plain face with no brand", () => {
    render(<FuelCardMark provider="Other" />);

    const card = screen.getByRole("img", { name: "Other" });
    expect(card.style.backgroundColor).toBe("");
    expect(markImage(card)).toBeNull();
  });
});
