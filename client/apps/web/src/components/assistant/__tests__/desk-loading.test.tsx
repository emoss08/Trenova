import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeskLoading, DeskLoadingMark } from "../voice/desk-loading";
import { REACH_ANGLE, REACH_LEAN } from "../voice/desk-visitor-motion";

const motion = vi.hoisted(() => ({ reduce: false }));

vi.mock("motion/react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("motion/react")>();
  return { ...actual, useReducedMotion: () => motion.reduce };
});

const MOVING = '[class*="animate-desk-"]';

const PARTS = [
  "walk",
  "fade",
  "turn",
  "bob",
  "lean",
  "leg-front",
  "leg-back",
  "arm-front",
  "arm-back",
  "sheet-x",
  "sheet-y",
  "sheet-fade",
  "pile",
  "pool",
  "cone",
] as const;

function part(container: HTMLElement, name: string) {
  return container.querySelector<SVGElement>(`[data-part="${name}"]`);
}

beforeEach(() => {
  motion.reduce = false;
});

describe("DeskLoadingMark", () => {
  it("moves every part of the visitor on its own fitted animation", () => {
    const { container } = render(<DeskLoadingMark />);

    for (const name of PARTS) {
      expect(part(container, name), name).toHaveClass(`animate-desk-visitor-${name}`);
    }
  });

  it("keeps the desk and the lamp's body still", () => {
    const { container } = render(<DeskLoadingMark />);

    expect(part(container, "desk")?.querySelector(MOVING)).toBeNull();
    expect(part(container, "desk")?.getAttribute("class") ?? "").not.toContain("animate-");
  });

  it("carries the sheet in the clip that stops at the desk top, with the pile", () => {
    const { container } = render(<DeskLoadingMark />);
    const clip = container.querySelector("clipPath");
    const pile = part(container, "pile");
    const sheet = part(container, "sheet");

    expect(pile?.closest("[clip-path]")?.getAttribute("clip-path")).toBe(`url(#${clip?.id})`);
    expect(sheet?.closest("[clip-path]")).toBe(pile?.closest("[clip-path]"));
    expect(pile?.querySelectorAll("rect")).toHaveLength(3);
  });

  it("draws the sheet touching down, and nothing moving, without motion", () => {
    const { container } = render(<DeskLoadingMark animate={false} />);

    expect(container.querySelector(MOVING)).toBeNull();
    expect(part(container, "arm-front")?.style.transform).toBe(`rotate(${REACH_ANGLE}deg)`);
    expect(part(container, "lean")?.style.transform).toContain(`rotate(${REACH_LEAN}deg)`);
    expect(part(container, "walk")?.getAttribute("style")).toBeNull();
    expect(container.querySelector("svg")).toHaveAttribute("data-motion", "still");
  });

  it("gives each drawing its own clip so two on a page never share one", () => {
    const { container } = render(
      <>
        <DeskLoadingMark />
        <DeskLoadingMark />
      </>,
    );
    const ids = Array.from(container.querySelectorAll("clipPath"), (node) => node.id);

    expect(ids).toHaveLength(2);
    expect(new Set(ids).size).toBe(2);
  });

  it("is hidden from assistive technology, takes the desk's light and clips at its edges", () => {
    const { container } = render(<DeskLoadingMark className="h-10" />);
    const svg = container.querySelector("svg");

    expect(svg).toHaveAttribute("aria-hidden", "true");
    expect(svg).toHaveClass("ui-desk-mark", "overflow-hidden", "h-10");
  });
});

describe("DeskLoading", () => {
  it("announces what is loading once, by its words", () => {
    render(<DeskLoading label="Opening the conversation" />);

    const status = screen.getByRole("status");
    expect(status).toHaveTextContent("Opening the conversation");
    expect(status.querySelector('[data-slot="desk-loading-mark"]')).not.toBeNull();
  });

  it("moves unless the person has asked for reduced motion", () => {
    const { container, rerender } = render(<DeskLoading label="Opening the conversation" />);

    expect(container.querySelector(MOVING)).not.toBeNull();

    motion.reduce = true;
    rerender(<DeskLoading label="Opening the conversation" />);

    expect(container.querySelector(MOVING)).toBeNull();
    expect(container.querySelector("svg")).toHaveAttribute("data-motion", "still");
  });
});
