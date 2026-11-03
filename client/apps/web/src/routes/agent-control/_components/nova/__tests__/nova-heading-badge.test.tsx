import { cleanup, fireEvent, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { AIControlSegment } from "@/lib/graphql/ai-control";
import { Hero } from "../../kit/hero";
import { NovaSummary } from "../nova-summary";

afterEach(cleanup);

// Two segments, as the summary resolver returns them: plain text and a strong figure.
const SENTENCE: AIControlSegment[] = [
  { text: "22 agents are on, ", strong: false, target: null, providerId: null, tone: null },
  { text: "2 working right now.", strong: true, target: null, providerId: null, tone: null },
];

function badge(container: HTMLElement): HTMLElement {
  const element = container.querySelector<HTMLElement>('[data-slot="nova-badge"]');
  if (!element) throw new Error("no badge rendered");
  return element;
}

function sentence(container: HTMLElement): HTMLElement {
  const element = container.querySelector<HTMLElement>(".say");
  if (!element) throw new Error("no sentence rendered");
  return element;
}

const turning = (container: HTMLElement) =>
  badge(container).classList.contains("animate-nova-spin");

describe("NovaSummary's badge", () => {
  it("turns while the sentence types in and stops once it has landed", () => {
    const { container } = render(
      <NovaSummary context="AI control" segments={SENTENCE} onTarget={vi.fn()} />,
    );

    expect(turning(container)).toBe(true);
    fireEvent.animationEnd(sentence(container));
    expect(turning(container)).toBe(false);
  });

  it("keeps turning until the sentence itself lands, not a link inside it", () => {
    const { container } = render(
      <NovaSummary
        context="AI control"
        segments={[
          {
            text: "10 proposals",
            strong: false,
            target: "watchtower",
            providerId: null,
            tone: null,
          },
        ]}
        onTarget={vi.fn()}
      />,
    );

    const link = container.querySelector<HTMLElement>('[role="link"]');
    if (!link) throw new Error("no link rendered");
    fireEvent.animationEnd(link);
    expect(turning(container)).toBe(true);

    fireEvent.animationEnd(sentence(container));
    expect(turning(container)).toBe(false);
  });

  it("turns again when Nova writes a new sentence", () => {
    const { container, rerender } = render(
      <NovaSummary context="AI control" segments={SENTENCE} onTarget={vi.fn()} />,
    );
    fireEvent.animationEnd(sentence(container));

    rerender(
      <NovaSummary
        context="AI control"
        segments={[
          {
            text: "Every agent is idle.",
            strong: false,
            target: null,
            providerId: null,
            tone: null,
          },
        ]}
        onTarget={vi.fn()}
      />,
    );
    expect(turning(container)).toBe(true);
  });

  it("turns while the summary is still being written, after the old sentence landed", () => {
    const { container } = render(
      <NovaSummary context="AI control" segments={SENTENCE} writing onTarget={vi.fn()} />,
    );
    fireEvent.animationEnd(sentence(container));

    expect(turning(container)).toBe(true);
  });

  it("turns while the sentence is loading", () => {
    const { container } = render(
      <NovaSummary context="AI control" segments={undefined} loading onTarget={vi.fn()} />,
    );

    expect(turning(container)).toBe(true);
  });
});

describe("Hero's badge", () => {
  it("turns while its sentence arrives, then holds still", () => {
    const { container } = render(
      <Hero context="Activity" sentenceKey="quiet">
        Nothing is waiting on you.
      </Hero>,
    );

    expect(turning(container)).toBe(true);
    fireEvent.animationEnd(sentence(container));
    expect(turning(container)).toBe(false);
  });

  it("turns while the sentence is being written", () => {
    const { container } = render(
      <Hero context="Analytics" sentenceKey="" writing>
        Reading the last 30 days.
      </Hero>,
    );
    fireEvent.animationEnd(sentence(container));

    expect(turning(container)).toBe(true);
  });

  it("turns again when the sentence changes", () => {
    const { container, rerender } = render(
      <Hero context="Retrieval" sentenceKey="indexed">
        Every document is indexed.
      </Hero>,
    );
    fireEvent.animationEnd(sentence(container));

    rerender(
      <Hero context="Retrieval" sentenceKey="indexing">
        Indexing 40 documents.
      </Hero>,
    );
    expect(turning(container)).toBe(true);
  });
});
