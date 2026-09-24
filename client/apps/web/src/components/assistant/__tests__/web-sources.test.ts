import type { AssistantMessage } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { toolEffect, type ToolStep } from "../activity";
import { groupThread } from "../thread-view";
import {
  indexSources,
  replyWebSources,
  sourceFor,
  sourceKey,
  sourcesOfStep,
  webSourcesOf,
} from "../web-sources";

function fenced(value: unknown): string {
  return `<untrusted_data>\n${JSON.stringify(value)}\n</untrusted_data>`;
}

function step(overrides: Partial<ToolStep> & { name: string }): ToolStep {
  return {
    id: `call_${Math.random().toString(36).slice(2, 8)}`,
    arguments: {},
    status: "done",
    content: "",
    effect: toolEffect(overrides.name),
    summary: "",
    durationSeconds: null,
    ...overrides,
  };
}

const FMCSA = "https://www.fmcsa.dot.gov/hours-service/elds/electronic-logging-devices";
const ECFR = "https://www.ecfr.gov/current/title-49/part-395/subpart-B";
const TRUCKING = "https://www.truckinginfo.com/eld-mandate-explained";

/** A web_search result shaped as web_tools.go writes it. */
function search(results: Record<string, unknown>[], query = "ELD mandate") {
  return step({
    name: "web_search",
    arguments: { query },
    content: fenced({ query, retrievedOn: "2026-09-20", results, note: "Cite every fact." }),
  });
}

const fmcsaHit = {
  ref: "r1",
  title: "Electronic Logging Devices",
  url: FMCSA,
  site: "fmcsa.dot.gov",
  source: "official",
  published: "2023-10-02",
  excerpts: ["short", "The ELD rule requires most motor carriers and drivers to use an ELD."],
};
const ecfrHit = {
  ref: "r2",
  title: "49 CFR Part 395 Subpart B",
  url: ECFR,
  site: "ecfr.gov",
  source: "official",
  excerpts: ["Each motor carrier must require its drivers to use an ELD to record duty status."],
};
const truckingHit = {
  ref: "r3",
  title: "The ELD mandate, explained",
  url: TRUCKING,
  site: "truckinginfo.com",
  source: "web",
  published: "2024-01-15",
  author: "Staff",
};

describe("sourcesOfStep", () => {
  it("reads every result of a search, with the fields the result omits left empty", () => {
    const sources = sourcesOfStep(search([fmcsaHit, truckingHit]));

    expect(sources).toEqual([
      {
        url: FMCSA,
        title: "Electronic Logging Devices",
        site: "fmcsa.dot.gov",
        official: true,
        published: "2023-10-02",
        excerpt: "The ELD rule requires most motor carriers and drivers to use an ELD.",
        retrievedOn: "2026-09-20",
        cited: false,
        read: false,
      },
      {
        url: TRUCKING,
        title: "The ELD mandate, explained",
        site: "truckinginfo.com",
        official: false,
        published: "2024-01-15",
        excerpt: "",
        retrievedOn: "2026-09-20",
        cited: false,
        read: false,
      },
    ]);
  });

  it("reads the one page a web_read returned", () => {
    const read = step({
      name: "web_read",
      arguments: { url: ECFR, ref: "r2" },
      content: fenced({
        title: "Part 395 Subpart B",
        url: ECFR,
        site: "ecfr.gov",
        source: "official",
        retrievedOn: "2026-09-21",
        part: 1,
        text: "§ 395.8 Driver's record of duty status.",
        note: "This is the end of the page.",
      }),
    });

    expect(sourcesOfStep(read)).toEqual([
      expect.objectContaining({
        url: ECFR,
        title: "Part 395 Subpart B",
        read: true,
        official: true,
        excerpt: "§ 395.8 Driver's record of duty status.",
        retrievedOn: "2026-09-21",
      }),
    ]);
  });

  it("takes nothing from a running, failed or truncated step, or from another tool", () => {
    expect(sourcesOfStep({ ...search([fmcsaHit]), status: "running" })).toEqual([]);
    expect(
      sourcesOfStep(
        step({
          name: "web_search",
          status: "failed",
          content: 'Tool "web_search" failed: the extension is off',
        }),
      ),
    ).toEqual([]);
    expect(
      sourcesOfStep(
        step({ name: "web_search", content: '<untrusted_data>\n{"results": [\n</untrusted_data>' }),
      ),
    ).toEqual([]);
    expect(
      sourcesOfStep(step({ name: "list_customers", content: fenced({ results: [fmcsaHit] }) })),
    ).toEqual([]);
  });

  it("drops a result whose address is not a web page", () => {
    const sources = sourcesOfStep(
      search([{ ...fmcsaHit, url: "javascript:alert(1)" }, { ...ecfrHit, url: "" }, truckingHit]),
    );

    expect(sources.map((source) => source.url)).toEqual([TRUCKING]);
  });
});

describe("webSourcesOf", () => {
  it("puts the pages the answer cites first, in the order it cites them, then the rest", () => {
    const answer =
      `Carriers must use an ELD ([ecfr.gov](${ECFR})). ` +
      `FMCSA says the same ([fmcsa.dot.gov](${FMCSA.replace("https://www.", "https://")}/)).`;

    const sources = webSourcesOf([search([fmcsaHit, truckingHit, ecfrHit])], answer);

    expect(sources.map((source) => [source.url, source.cited])).toEqual([
      [ECFR, true],
      [FMCSA, true],
      [TRUCKING, false],
    ]);
  });

  it("keeps a page found twice once, marked read when it was also opened", () => {
    const again = search([fmcsaHit], "ELD rule exemptions");
    const read = step({
      name: "web_read",
      arguments: { url: FMCSA, ref: "r1" },
      content: fenced({
        title: "Electronic Logging Devices | FMCSA",
        url: FMCSA,
        site: "fmcsa.dot.gov",
        source: "official",
        retrievedOn: "2026-09-20",
        part: 1,
        text: "Full page text.",
      }),
    });

    const sources = webSourcesOf([search([fmcsaHit]), again, read], "");

    expect(sources).toHaveLength(1);
    expect(sources[0]).toMatchObject({
      read: true,
      title: "Electronic Logging Devices | FMCSA",
      published: "2023-10-02",
      excerpt: "The ELD rule requires most motor carriers and drivers to use an ELD.",
    });
  });

  it("finds the source a link names however the model spelled its address", () => {
    const index = indexSources(webSourcesOf([search([fmcsaHit])], ""));

    expect(
      sourceFor(index, "https://FMCSA.dot.gov/hours-service/elds/electronic-logging-devices/")
        ?.title,
    ).toBe("Electronic Logging Devices");
    expect(sourceFor(index, "https://example.com/elsewhere")).toBeUndefined();
    expect(sourceFor(index, undefined)).toBeUndefined();
    expect(sourceKey("https://www.a.com/x/?q=1")).toBe("a.com/x?q=1");
  });
});

describe("replyWebSources", () => {
  let sequence = 0;
  function message(overrides: Partial<AssistantMessage>): AssistantMessage {
    sequence += 1;
    return {
      id: `msg_${sequence}`,
      threadId: "t1",
      sequence,
      role: "User",
      content: "",
      toolCalls: null,
      toolCallId: "",
      toolName: "",
      toolFailed: false,
      scopeStage: "",
      scopeCategory: "",
      scopeReason: "",
      refused: false,
      model: "",
      inputTokens: 0,
      outputTokens: 0,
      createdAt: 1_700_000_000 + sequence,
      kind: "Message",
      ...overrides,
    };
  }

  const searchResult = fenced({
    query: "ELD mandate",
    retrievedOn: "2026-09-20",
    results: [fmcsaHit, truckingHit],
  });

  it("gives the answer the pages an earlier step of the same reply searched", () => {
    const entries = groupThread([
      message({ role: "User", content: "What is the ELD mandate?" }),
      message({
        role: "Assistant",
        toolCalls: [{ id: "c1", name: "web_search", arguments: { query: "ELD mandate" } }],
      }),
      message({ role: "Tool", toolCallId: "c1", toolName: "web_search", content: searchResult }),
      message({
        id: "answer",
        role: "Assistant",
        content: `Most carriers need one ([fmcsa.dot.gov](${FMCSA})).`,
      }),
      message({ role: "User", content: "Thanks. How many trucks do we run?" }),
      message({ id: "later", role: "Assistant", content: "Twelve." }),
    ]);

    const sources = replyWebSources(entries);

    expect(sources.get("answer")?.sources.map((source) => [source.site, source.cited])).toEqual([
      ["fmcsa.dot.gov", true],
      ["truckinginfo.com", false],
    ]);
    expect(sources.get("answer")?.answer).toBe(true);
    expect(sources.has("later")).toBe(false);
  });

  it("lists the sources once, under the last step that says anything", () => {
    const entries = groupThread([
      message({ role: "User", content: "What is the ELD mandate?" }),
      message({
        id: "interim",
        role: "Assistant",
        content: "Let me check the rule.",
        toolCalls: [{ id: "c1", name: "web_search", arguments: { query: "ELD mandate" } }],
      }),
      message({ role: "Tool", toolCallId: "c1", toolName: "web_search", content: searchResult }),
      message({ id: "answer", role: "Assistant", content: `See ([fmcsa.dot.gov](${FMCSA})).` }),
      message({ id: "silent", role: "Assistant", content: "  " }),
    ]);

    const sources = replyWebSources(entries);

    expect(sources.get("interim")).toMatchObject({ answer: false });
    expect(sources.get("interim")?.sources).toHaveLength(2);
    expect(sources.get("answer")).toMatchObject({ answer: true });
    expect(sources.has("silent")).toBe(false);
  });
});
