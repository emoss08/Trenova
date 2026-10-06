import { ArtifactLinkContext, AiMarkdown } from "@/components/elements/ai-markdown";
import { artifactRefIds, withArtifactRefs } from "@/lib/artifact-ref";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

const reply =
  "Eleven have no biller, and an item can't be approved without one: [Missing a biller](artifact:aart_01GAPS). Read [the guide](https://example.com/guide).";

describe("artifact links in a reply", () => {
  it("draws a known artifact where the sentence names it", () => {
    render(
      <ArtifactLinkContext
        value={(id, children) =>
          id === "aart_01GAPS" ? <button type="button">{children}</button> : null
        }
      >
        <AiMarkdown content={reply} />
      </ArtifactLinkContext>,
    );

    const badge = screen.getByRole("button", { name: "Missing a biller" });
    expect(badge.closest("p")).toHaveTextContent("without one: Missing a biller.");
  });

  it("leaves the link as its words where nothing can open it", () => {
    render(<AiMarkdown content={reply} />);

    expect(screen.getByText("Missing a biller").tagName).toBe("SPAN");
    expect(screen.queryByRole("link", { name: "Missing a biller" })).toBeNull();
  });

  it("keeps an unknown artifact as words", () => {
    render(
      <ArtifactLinkContext value={() => null}>
        <AiMarkdown content={reply} />
      </ArtifactLinkContext>,
    );

    expect(screen.queryByRole("button")).toBeNull();
    expect(screen.getByText("Missing a biller")).toBeInTheDocument();
  });

  it("does not change other links", () => {
    render(<AiMarkdown content={reply} />);

    expect(screen.getByRole("link", { name: "the guide" })).toHaveAttribute(
      "href",
      "https://example.com/guide",
    );
  });

  it("knows which artifacts a reply names", () => {
    expect([...artifactRefIds(reply)]).toEqual(["aart_01GAPS"]);
    expect(artifactRefIds("No artifacts here.").size).toBe(0);
  });
});

describe("withArtifactRefs", () => {
  const brief = { id: "art_01BRIEF", title: "Storm impact on today's loads" };

  it("names an unnamed artifact at the end of the last sentence", () => {
    expect(withArtifactRefs("I wrote it up. Two loads are late.", [brief])).toBe(
      "I wrote it up. Two loads are late. [Storm impact on today's loads](artifact:art_01BRIEF)",
    );
  });

  it("leaves an artifact the reply already named where it is", () => {
    const named = "See [the brief](artifact:art_01BRIEF) for details.";
    expect(withArtifactRefs(named, [brief])).toBe(named);
  });

  it("starts a line of its own after a table", () => {
    expect(withArtifactRefs("| a | b |\n|---|---|\n| 1 | 2 |", [brief])).toMatch(
      /\| 1 \| 2 \|\n\n\[Storm/,
    );
  });

  it("keeps brackets in a title from breaking the link", () => {
    expect(withArtifactRefs("Done.", [{ id: "art_1", title: "Queue [draft]" }])).toBe(
      "Done. [Queue draft](artifact:art_1)",
    );
  });

  it("names the artifacts of a reply that has no words", () => {
    expect(withArtifactRefs("", [brief])).toBe(
      "[Storm impact on today's loads](artifact:art_01BRIEF)",
    );
  });
});
