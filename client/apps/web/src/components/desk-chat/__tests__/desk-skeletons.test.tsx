import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import {
  DeskArtifactBodySkeleton,
  DeskTranscriptSkeleton,
  DeskWorkspaceSkeleton,
} from "../desk-skeletons";

/**
 * The Desk's loading shapes are the outline of what replaces them, drawn in
 * the Desk's palette, and say what is loading to a screen reader without
 * reading the outline out.
 */
describe("Desk skeletons", () => {
  it("lays two turns on the transcript's grid while a conversation is read", () => {
    const { container } = render(
      <div className="dk-grid dk-flow">
        <DeskTranscriptSkeleton />
      </div>,
    );

    const turns = container.querySelectorAll(".dk-r.dk-skt");
    expect(turns).toHaveLength(2);
    expect(turns[0]).toHaveClass("dk-first");
    expect(turns[1]).not.toHaveClass("dk-first");
    for (const turn of turns) {
      expect(turn).toHaveAttribute("aria-hidden", "true");
      expect(turn.querySelector(".dk-g .dk-sk-time")).not.toBeNull();
      expect(turn.querySelector(".dk-c .dk-sk-q")).not.toBeNull();
    }
    expect(screen.getByRole("status")).toHaveTextContent("Reading the conversation…");
  });

  it.each([
    ["table", ".dk-skb-tb"],
    ["document", ".dk-skb-doc"],
    ["record", ".dk-skb-rec"],
    ["lines", ".dk-skb-para"],
  ] as const)("draws a %s artifact in its own shape", (shape, selector) => {
    const { container } = render(<DeskArtifactBodySkeleton shape={shape} />);

    expect(container.querySelector(selector)).not.toBeNull();
    expect(container.querySelector(".dk-skb")).toHaveAttribute("aria-busy", "true");
    expect(screen.getByRole("status")).toHaveTextContent("Loading the artifact…");
  });

  it("draws a table's heading and its rows", () => {
    const { container } = render(<DeskArtifactBodySkeleton shape="table" />);

    expect(container.querySelectorAll(".dk-skb-tr.dk-skb-th")).toHaveLength(1);
    expect(container.querySelectorAll(".dk-skb-tr:not(.dk-skb-th)")).toHaveLength(8);
  });

  it("outlines the workspace's card, contents and foot", () => {
    const { container } = render(<DeskWorkspaceSkeleton shape="record" />);

    const pane = container.querySelector(".dk-apx");
    expect(pane).toHaveAttribute("aria-busy", "true");
    expect(pane?.querySelector(".dk-ax-top .dk-ax-card.dk-front")).not.toBeNull();
    expect(pane?.querySelector(".dk-ax-body .dk-skb-rec")).not.toBeNull();
    expect(pane?.querySelector(".dk-ax-foot .dk-skw-link")).not.toBeNull();
  });

  it("draws everything in the Desk's own skeleton, never the app's", () => {
    const { container } = render(
      <>
        <DeskTranscriptSkeleton />
        <DeskWorkspaceSkeleton shape="table" />
      </>,
    );

    expect(container.querySelectorAll(".dk-sk").length).toBeGreaterThan(0);
    expect(container.querySelector('[data-slot="skeleton"], .ui-shimmer')).toBeNull();
  });
});
