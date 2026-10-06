import type { TurnState } from "@/components/assistant/turn-stream";
import { describe, expect, it } from "vitest";
import {
  citedSources,
  citeLabel,
  deskWebSources,
  liveWebSearch,
  siteHue,
  sourceAge,
  webCiteIds,
  webQueryOf,
  withWebCites,
} from "../web-cites";

const result = (url: string, title: string, published = "") => ({
  url,
  title,
  site: new URL(url).hostname.replace(/^www\./u, ""),
  source: "web",
  published,
  excerpts: ["Weekly U.S. on-highway diesel average, released Mondays, for every region."],
});

const search = (query: string, results: ReturnType<typeof result>[]) => ({
  name: "web_search",
  status: "done" as const,
  arguments: { query },
  // Shaped as the runtime saves a tool result: the JSON inside its untrusted-data fence.
  content: `<untrusted_data>\n${JSON.stringify({ query, retrievedOn: "2026-10-04", results })}\n</untrusted_data>`,
});

const today = new Date(2026, 9, 4);

describe("deskWebSources", () => {
  it("lists each page once, in the order the searches first returned it", () => {
    const steps = [
      search("DOE diesel average", [
        result("https://www.eia.gov/petroleum/gasdiesel/", "Gasoline and Diesel Fuel Update"),
        result("https://freightwaves.com/news/diesel", "Diesel ticks up"),
      ]),
      search("I-70 Ohio closures", [
        result("https://eia.gov/petroleum/gasdiesel", "Gasoline and Diesel Fuel Update"),
        result("https://ohgo.com/", "OHGO"),
      ]),
    ];

    const sources = deskWebSources(steps, today);

    expect(sources.map((source) => source.site)).toEqual([
      "eia.gov",
      "freightwaves.com",
      "ohgo.com",
    ]);
    expect(sources[0].snippet).toContain("diesel average");
  });

  it("skips steps that are not web searches or have not returned", () => {
    expect(
      deskWebSources(
        [
          { name: "get_shipment", status: "done", content: "{}" },
          { ...search("x", [result("https://eia.gov/", "EIA")]), status: "running", content: "" },
        ],
        today,
      ),
    ).toEqual([]);
  });
});

describe("sourceAge", () => {
  it("reads today as Today, this year as a month and day, and another year with it", () => {
    expect(sourceAge("2026-10-04", today)).toBe("Today");
    expect(sourceAge("2026-09-28", today)).toBe("Sep 28");
    expect(sourceAge("2025-09-28", today)).toBe("Sep 28, 2025");
    expect(sourceAge("", today)).toBe("");
  });
});

describe("withWebCites", () => {
  const sources = deskWebSources(
    [
      search("diesel", [
        result("https://eia.gov/petroleum", "EIA"),
        result("https://ohgo.com/", "OHGO"),
        result("https://dat.com/", "DAT"),
        result("https://freightwaves.com/diesel", "FreightWaves"),
      ]),
    ],
    today,
  );

  it("numbers a cited page by its place among the pages found", () => {
    expect(withWebCites("Diesel is up 3¢ [ohgo.com](https://www.ohgo.com).", sources)).toBe(
      "Diesel is up 3¢ [2](#dk-web-2).",
    );
  });

  it("makes links side by side one citation, each page once", () => {
    expect(
      withWebCites(
        "Up 3¢ [eia.gov](https://eia.gov/petroleum), [freightwaves.com](https://freightwaves.com/diesel) [eia.gov](https://eia.gov/petroleum/).",
        sources,
      ),
    ).toBe("Up 3¢ [1, 4](#dk-web-1.4).");
  });

  it("leaves a link to a page the agent never found, and code, as written", () => {
    const text =
      "See [example.com](https://example.com) and `[eia.gov](https://eia.gov/petroleum)`.";
    expect(withWebCites(text, sources)).toBe(text);
  });

  it("changes nothing when the reply found no pages", () => {
    const text = "See [eia.gov](https://eia.gov/petroleum).";
    expect(withWebCites(text, [])).toBe(text);
  });

  it("reads a citation's numbers back, and drops numbers with no page", () => {
    expect(webCiteIds("#dk-web-1.4")).toEqual([1, 4]);
    expect(webCiteIds("https://eia.gov")).toBeNull();
    expect(citedSources([2, 9], sources).map((source) => source.site)).toEqual(["ohgo.com"]);
    expect(citedSources([9], sources)).toEqual([]);
  });
});

describe("citeLabel", () => {
  it("names the site without a trailing .com, .gov or .org", () => {
    expect(citeLabel("eia.gov")).toBe("eia");
    expect(citeLabel("freightwaves.com")).toBe("freightwaves");
    expect(citeLabel("fmcsa.dot.gov")).toBe("fmcsa.dot");
    expect(citeLabel("ttnews.co.uk")).toBe("ttnews.co.uk");
  });
});

describe("siteHue", () => {
  it("gives a site the same colour every time", () => {
    expect(siteHue("eia.gov")).toBe(siteHue("eia.gov"));
    expect(siteHue("eia.gov")).toBeGreaterThanOrEqual(0);
    expect(siteHue("eia.gov")).toBeLessThan(360);
    expect(siteHue("eia.gov")).not.toBe(siteHue("dat.com"));
  });
});

describe("liveWebSearch", () => {
  const turn = (segments: TurnState["segments"], status: TurnState["status"] = "working") =>
    ({ status, segments }) as TurnState;
  const searchSegment = (status: "running" | "done", content = "") => ({
    kind: "tool" as const,
    callId: "call_1",
    name: "web_search",
    arguments: { query: "DOE diesel average" },
    status,
    content,
  });

  it("shows the query while the search runs, and its pages once they are reported", () => {
    expect(liveWebSearch(turn([searchSegment("running")]))).toEqual({
      query: "DOE diesel average",
      sources: [],
    });
    const done = searchSegment("done", search("x", [result("https://eia.gov/", "EIA")]).content);
    expect(liveWebSearch(turn([done]))?.sources.map((source) => source.site)).toEqual(["eia.gov"]);
  });

  it("goes once the agent writes, takes another step, or the turn ends", () => {
    const done = searchSegment("done");
    expect(liveWebSearch(turn([done, { kind: "text", text: "The", closed: false }]))).toBeNull();
    expect(liveWebSearch(turn([done, { ...done, callId: "call_2", name: "web_read" }]))).toBeNull();
    expect(liveWebSearch(turn([done], "done"))).toBeNull();
  });
});

describe("webQueryOf", () => {
  it("is the first search's query", () => {
    expect(
      webQueryOf([
        { name: "get_shipment", arguments: { query: "no" } },
        { name: "web_search", arguments: { query: " DOE diesel average " } },
      ]),
    ).toBe("DOE diesel average");
    expect(webQueryOf([])).toBe("");
  });
});
