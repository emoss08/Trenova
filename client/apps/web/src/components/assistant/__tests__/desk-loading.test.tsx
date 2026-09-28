import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeskLoading, DeskLoadingMark } from "../voice/desk-loading";

const motion = vi.hoisted(() => ({ reduce: false }));

vi.mock("motion/react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("motion/react")>();
  return { ...actual, useReducedMotion: () => motion.reduce };
});

const MOVING = '[class*="animate-desk-"]';

function part(container: HTMLElement, name: string) {
  return container.querySelector(`[data-part="${name}"]`);
}

beforeEach(() => {
  motion.reduce = false;
});

describe("DeskLoadingMark", () => {
  it("opens the drawer, files a sheet and settles the stack while it moves", () => {
    const { container } = render(<DeskLoadingMark />);

    expect(part(container, "drawer")).toHaveClass("animate-desk-drawer");
    expect(part(container, "sheet")).toHaveClass("animate-desk-file");
    expect(part(container, "stack")).toHaveClass("animate-desk-file-stack");
    expect(part(container, "desk")?.getAttribute("class") ?? "").not.toContain("animate-");
  });

  it("draws one still frame, the drawer open and a sheet half in, without motion", () => {
    const { container } = render(<DeskLoadingMark animate={false} />);

    expect(container.querySelector(MOVING)).toBeNull();
    expect(part(container, "drawer")).toHaveClass("translate-x-[7.6px]");
    expect(part(container, "sheet")).toHaveClass("-translate-y-[3.2px]");
    expect(container.querySelector("svg")).toHaveAttribute("data-motion", "still");
  });

  it("files the sheet behind the drawer's side, clipped at the drawer floor", () => {
    const { container } = render(<DeskLoadingMark />);
    const drawer = part(container, "drawer");
    const clip = container.querySelector("clipPath");
    const clipped = drawer?.querySelector("[clip-path]");

    expect(clip).not.toBeNull();
    expect(clipped?.getAttribute("clip-path")).toBe(`url(#${clip?.id})`);
    expect(clipped?.contains(part(container, "sheet"))).toBe(true);
    expect(drawer?.lastElementChild?.contains(part(container, "sheet"))).toBe(false);
  });

  it("gives each drawing its own clip and mask so two on a page never share one", () => {
    const { container } = render(
      <>
        <DeskLoadingMark />
        <DeskLoadingMark />
      </>,
    );
    const ids = Array.from(container.querySelectorAll("clipPath, mask"), (node) => node.id);

    expect(new Set(ids).size).toBe(4);
  });

  it("is hidden from assistive technology and takes the desk's light", () => {
    const { container } = render(<DeskLoadingMark className="h-10" />);
    const svg = container.querySelector("svg");

    expect(svg).toHaveAttribute("aria-hidden", "true");
    expect(svg).toHaveClass("ui-desk-mark", "h-10");
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
