import { AiMarkdown } from "@/components/elements/ai-markdown";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CitationProvider, SourcesFooter } from "../web-citations";
import type { WebSource } from "../web-sources";

function source(overrides: Partial<WebSource> & { url: string; site: string }): WebSource {
  return {
    title: overrides.site,
    official: false,
    published: "",
    excerpt: "",
    retrievedOn: "2026-09-20",
    cited: false,
    read: false,
    ...overrides,
  };
}

const fmcsa = source({
  url: "https://www.fmcsa.dot.gov/hours-service/elds",
  site: "fmcsa.dot.gov",
  title: "Electronic Logging Devices",
  official: true,
  published: "2023-10-02",
  cited: true,
});
const ecfr = source({
  url: "https://www.ecfr.gov/current/title-49/part-395",
  site: "ecfr.gov",
  title: "49 CFR Part 395",
  official: true,
  cited: true,
});
const trucking = source({
  url: "https://www.truckinginfo.com/eld",
  site: "truckinginfo.com",
  title: "The ELD mandate, explained",
});

describe("CitationProvider", () => {
  it("draws a link to a page the agent found as that site, opening the page", () => {
    render(
      <CitationProvider sources={[fmcsa]}>
        <AiMarkdown
          content={
            "Most carriers need an ELD [FMCSA – Electronic Logging Devices](https://fmcsa.dot.gov/hours-service/elds/)."
          }
        />
      </CitationProvider>,
    );

    const chip = screen.getByRole("link", {
      name: /Electronic Logging Devices on fmcsa\.dot\.gov/u,
    });
    expect(chip).toHaveTextContent("fmcsa.dot.gov");
    expect(chip).toHaveAttribute("href", fmcsa.url);
    expect(chip).toHaveAttribute("target", "_blank");
    expect(chip).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("leaves a link the agent never searched a plain link, words and all", () => {
    render(
      <CitationProvider sources={[fmcsa]}>
        <AiMarkdown content={"See [the carrier's site](https://example.com/rates)."} />
      </CitationProvider>,
    );

    const link = screen.getByRole("link", { name: "the carrier's site" });
    expect(link).toHaveAttribute("href", "https://example.com/rates");
  });
});

describe("SourcesFooter", () => {
  it("names the sites and counts every page, then lists cited pages before the rest", () => {
    render(<SourcesFooter sources={[fmcsa, ecfr, trucking]} />);

    const toggle = screen.getByRole("button", { name: /3 sources/u });
    expect(toggle).toHaveTextContent("fmcsa.dot.gov, ecfr.gov +1");
    expect(toggle).toHaveAttribute("aria-expanded", "false");

    fireEvent.click(toggle);

    const links = screen.getAllByRole("link");
    expect(links.map((link) => link.getAttribute("href"))).toEqual([
      fmcsa.url,
      ecfr.url,
      trucking.url,
    ]);
    expect(screen.getByText("Also found")).toBeInTheDocument();
    expect(links[0]).toHaveTextContent("Official source");
    expect(links[2]).toHaveTextContent("No publish date");
  });

  it("draws nothing when the reply found no pages", () => {
    const { container } = render(<SourcesFooter sources={[]} />);

    expect(container).toBeEmptyDOMElement();
  });
});
