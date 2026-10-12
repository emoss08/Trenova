import { ArtifactLinkContext, AiMarkdown } from "@/components/elements/ai-markdown";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

const reply =
  "Eleven have no biller, and an item can't be approved without one: [Missing a biller](artifact:aart_01GAPS). Read [the guide](https://example.com/guide).";

describe("artifact links in a reply", () => {
  it("draws a known artifact where the sentence names it", () => {
    render(
      <ArtifactLinkContext
        value={(id) =>
          id === "aart_01GAPS" ? <button type="button">Billing queue items</button> : null
        }
      >
        <AiMarkdown content={reply} />
      </ArtifactLinkContext>,
    );

    const badge = screen.getByRole("button", { name: "Billing queue items" });
    expect(badge.closest("p")).toHaveTextContent("without one: Billing queue items.");
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
});
