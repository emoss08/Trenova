import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { DeskWebCite, DeskWebLive, DeskWebSources } from "../desk-web";
import type { DeskWebSource } from "../web-cites";

const page = (site: string, title: string, age = "Sep 28"): DeskWebSource => ({
  url: `https://${site}/page`,
  site,
  title,
  snippet: `What ${site} says.`,
  age,
});

const sources = [
  page("eia.gov", "Gasoline and Diesel Fuel Update"),
  page("ohgo.com", "OHGO", "Today"),
  page("dat.com", "Weekly freight market update"),
  page("freightwaves.com", "Diesel ticks up"),
  page("weather.gov", "Columbus forecast"),
];

describe("DeskWebCite", () => {
  it("names the first site without its .gov and counts the others", () => {
    const { container } = render(<DeskWebCite ids={[1, 4]} sources={sources} />);
    const pill = container.querySelector("a.dk-wc")!;

    expect(pill.textContent).toBe("eia+1");
    expect(pill.getAttribute("href")).toBe("https://eia.gov/page");
    expect(pill.getAttribute("target")).toBe("_blank");
  });

  it("lists every cited page while it is rested on", () => {
    const { container } = render(<DeskWebCite ids={[1, 4]} sources={sources} />);
    fireEvent.mouseEnter(container.querySelector(".dk-wc-w")!);

    // The popover hangs outside the reply's text; nothing is added beside the pill.
    expect(container.querySelector(".dk-wc-w")?.children).toHaveLength(1);
    const entries = [...document.querySelectorAll(".dk-wc-it")];
    expect(entries.map((entry) => entry.getAttribute("href"))).toEqual([
      "https://eia.gov/page",
      "https://freightwaves.com/page",
    ]);
    expect(entries[0].textContent).toContain("Sep 28");
    expect(entries[0].textContent).toContain("What eia.gov says.");
  });

  it("draws nothing when none of its pages are known", () => {
    const { container } = render(<DeskWebCite ids={[9]} sources={sources} />);

    expect(container.innerHTML).toBe("");
  });
});

describe("DeskWebLive", () => {
  it("shows the query and a chip for each page reported so far", () => {
    const { container } = render(
      <DeskWebLive
        label="Searching the web"
        query="DOE diesel average"
        sources={sources.slice(0, 2)}
      />,
    );

    expect(container.querySelector(".dk-wl-q")?.textContent).toBe("DOE diesel average");
    expect([...container.querySelectorAll(".dk-wl-it")].map((chip) => chip.textContent)).toEqual([
      "Eeia.gov",
      "Oohgo.com",
    ]);
  });
});

describe("DeskWebSources", () => {
  it("stacks at most four sites and opens to the numbered pages", () => {
    const { container, getByRole } = render(
      <DeskWebSources sources={sources} query="DOE diesel average" />,
    );

    expect(container.querySelectorAll(".dk-ws-st .dk-wf")).toHaveLength(4);
    expect(getByRole("button").textContent).toContain("5 sources");
    fireEvent.click(getByRole("button"));
    expect(container.querySelector(".dk-ws")?.classList.contains("dk-open")).toBe(true);
    expect(container.querySelectorAll(".dk-ws-l li")).toHaveLength(5);
    expect(container.querySelector(".dk-ws-d")?.textContent).toBe("eia.gov · Sep 28");
  });

  it("is not drawn when the reply found no pages", () => {
    const { container } = render(<DeskWebSources sources={[]} query="" />);

    expect(container.innerHTML).toBe("");
  });
});
