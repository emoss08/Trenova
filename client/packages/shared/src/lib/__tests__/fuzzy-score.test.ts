import { describe, expect, it } from "vitest";
import {
  ALL_ON_TITLE_BONUS,
  FIELD_PREFIX_SCORE,
  INITIALS_SCORE,
  INSIDE_WORD_SCORE,
  isWordStart,
  matchFields,
  matchTerm,
  mergeRanges,
  rankMatches,
  TITLE_PHRASE_SCORE,
  type MatchField,
} from "../fuzzy-score";

describe("matchTerm", () => {
  it("scores the start of the field highest", () => {
    expect(matchTerm("ship", "Shipments")).toEqual({
      score: FIELD_PREFIX_SCORE,
      ranges: [[0, 4]],
    });
  });

  it("matches the start of a later word, a little less the further in", () => {
    const near = matchTerm("cod", "Fleet codes");
    const far = matchTerm("rea", "Service failure reasons");
    expect(near?.ranges).toEqual([[6, 9]]);
    expect(far?.ranges).toEqual([[16, 19]]);
    expect(near!.score).toBeCloseTo(0.9 - 6 * 0.002);
    expect(near!.score).toBeGreaterThan(far!.score);
  });

  it("treats a case change and a letter-to-digit change as a word start", () => {
    expect(matchTerm("queue", "billingQueue")?.ranges).toEqual([[7, 12]]);
    expect(matchTerm("240193", "S-240193")?.ranges).toEqual([[2, 8]]);
    expect(matchTerm("240193", "S240193")?.ranges).toEqual([[1, 7]]);
  });

  it("matches initials on word starts, only across two or more words", () => {
    expect(matchTerm("bq", "Billing queue")).toEqual({
      score: INITIALS_SCORE,
      ranges: [
        [0, 1],
        [8, 9],
      ],
    });
    expect(matchTerm("bq", "Bq")).toEqual({ score: FIELD_PREFIX_SCORE, ranges: [[0, 2]] });
    expect(matchTerm("xq", "Billing queue")).toBeNull();
  });

  it("matches inside a word only for three or more characters", () => {
    expect(matchTerm("ai", "Trailers")).toBeNull();
    expect(matchTerm("ai", "Email logs")).toBeNull();
    expect(matchTerm("ai", "Service failures")).toBeNull();
    expect(matchTerm("ail", "Trailers")).toEqual({ score: INSIDE_WORD_SCORE, ranges: [[2, 5]] });
  });

  it("ignores case and accents without moving the ranges", () => {
    expect(matchTerm("jose", "María José Álvarez")?.ranges).toEqual([[6, 10]]);
    expect(matchTerm("ALV", "María José Álvarez")?.ranges).toEqual([[11, 14]]);
  });

  it("never matches an empty term or field", () => {
    expect(matchTerm("", "Shipments")).toBeNull();
    expect(matchTerm("ship", "")).toBeNull();
  });
});

describe("isWordStart", () => {
  it("marks the starts a person would call words", () => {
    const text = "PRO S-240193 dueDate";
    const starts = Array.from({ length: text.length }, (_, i) => i).filter((i) =>
      isWordStart(text, i),
    );
    expect(starts).toEqual([0, 4, 6, 13, 16]);
  });
});

describe("mergeRanges", () => {
  it("joins overlapping and touching ranges and sorts them", () => {
    expect(
      mergeRanges([
        [6, 8],
        [0, 2],
        [2, 3],
        [7, 10],
      ]),
    ).toEqual([
      [0, 3],
      [6, 10],
    ]);
  });
});

type PageKey = "title" | "trail" | "keyword";

function page(title: string, trail: string, keywords: string[] = []): MatchField<PageKey>[] {
  return [
    { key: "title", text: title },
    ...keywords.map((keyword): MatchField<PageKey> => ({
      key: "keyword",
      text: keyword,
      weight: 0.82,
      via: "Keyword",
    })),
    { key: "trail", text: trail, weight: 0.62 },
  ];
}

describe("matchFields", () => {
  it("matches everything equally on an empty query", () => {
    expect(matchFields("   ", page("Shipments", ""))).toEqual({
      score: 1,
      highlights: {},
      via: null,
    });
  });

  it("does not let 'ai' match inside Service failures, Trailers or Email logs", () => {
    expect(matchFields("ai", page("Service failures", "Shipment management"))).toBeNull();
    expect(matchFields("ai", page("Trailers", "Equipment management"))).toBeNull();
    expect(matchFields("ai", page("Email logs", "Administration › Email & SMS"))).toBeNull();
  });

  it("lets 'ai' match AI control and Aisha Grant on their titles", () => {
    expect(matchFields("ai", page("AI control", "Administration › AI & Automation"))).toEqual({
      score: FIELD_PREFIX_SCORE + ALL_ON_TITLE_BONUS,
      highlights: { title: [[0, 2]] },
      via: null,
    });
    expect(matchFields("ai", [{ key: "title", text: "Aisha Grant" }])?.highlights).toEqual({
      title: [[0, 2]],
    });
  });

  it("matches Document intelligence through a keyword and reports it as the match source", () => {
    const match = matchFields(
      "ai",
      page("Document intelligence", "Administration › Documents", ["ocr", "extraction", "ai"]),
    );
    expect(match).toEqual({
      score: 0.82,
      highlights: {},
      via: { label: "Keyword", text: "ai", ranges: [[0, 2]] },
    });
  });

  it("highlights only the field each word won", () => {
    const match = matchFields("billing inv", [
      { key: "title", text: "Invoices" },
      { key: "trail", text: "Billing", weight: 0.62 },
    ]);
    expect(match?.highlights).toEqual({ title: [[0, 3]], trail: [[0, 7]] });
  });

  it("never highlights a word in a field it did not win", () => {
    const match = matchFields("acme", [
      { key: "title", text: "S-240193" },
      { key: "customer", text: "Acme Foods", weight: 0.92 },
      { key: "route", text: "Acme, TX → Atlanta, GA", weight: 0.75 },
    ]);
    expect(match?.highlights).toEqual({ customer: [[0, 4]] });
  });

  it("requires every word to land somewhere", () => {
    expect(matchFields("new cust", [{ key: "title", text: "New customer" }])).not.toBeNull();
    expect(matchFields("new cust", [{ key: "title", text: "New shipment" }])).toBeNull();
  });

  it("scores a phrase that begins the title above everything, as one run", () => {
    expect(matchFields("new cus", [{ key: "title", text: "New customer" }])).toEqual({
      score: TITLE_PHRASE_SCORE,
      highlights: { title: [[0, 7]] },
      via: null,
    });
  });

  it("adds the title bonus only when every word landed on the title", () => {
    const both = matchFields("ship type", [{ key: "title", text: "Shipment types" }]);
    const split = matchFields("ship mgmt", [
      { key: "title", text: "Shipment types" },
      { key: "trail", text: "mgmt", weight: 0.62 },
    ]);
    expect(both!.score).toBeCloseTo((1 + (0.9 - 9 * 0.002)) / 2 + ALL_ON_TITLE_BONUS);
    expect(split!.score).toBeCloseTo((1 + 0.62) / 2);
  });

  it("names a hidden field once even when several words land there", () => {
    const match = matchFields("leon park", [
      { key: "title", text: "S-240201" },
      { key: "driver", text: "Leon Park", weight: 0.6, via: "Driver" },
    ]);
    expect(match?.via).toEqual({ label: "Driver", text: "Leon Park", ranges: [[0, 4]] });
    expect(match?.highlights).toEqual({});
  });
});

describe("rankMatches", () => {
  const entries = ["Shipments", "Shipment types", "Hazmat segregation rules", "Fleet codes"];
  const fields = (entry: string): MatchField[] => [{ key: "title", text: entry }];

  it("keeps matches only, best first, with their ranges", () => {
    expect(rankMatches("types", entries, fields)).toEqual([
      {
        entry: "Shipment types",
        match: {
          score: 0.9 - 9 * 0.002 + ALL_ON_TITLE_BONUS,
          highlights: { title: [[9, 14]] },
          via: null,
        },
      },
    ]);
  });

  it("returns everything in its original order for an empty query", () => {
    expect(rankMatches("", entries, fields).map((ranked) => ranked.entry)).toEqual(entries);
  });

  it("keeps the original order between equal scores", () => {
    expect(rankMatches("s", ["Sb", "Sa"], fields).map((ranked) => ranked.entry)).toEqual([
      "Sb",
      "Sa",
    ]);
  });
});
