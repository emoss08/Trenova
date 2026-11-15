import { describe, expect, it } from "vitest";
import { parseSearch, stringifySearch } from "../search";

describe("parseSearch", () => {
  it("reads every value as the string the address holds", () => {
    expect(parseSearch("?id=00123&count=5&open=true&empty=")).toEqual({
      id: "00123",
      count: "5",
      open: "true",
      empty: "",
    });
  });

  it("keeps every value of a repeated key, in order", () => {
    expect(parseSearch("?status=open&status=paid&page=2")).toEqual({
      status: ["open", "paid"],
      page: "2",
    });
  });

  it("decodes plus signs and escapes", () => {
    expect(parseSearch("q=acme+freight&note=a%26b%3Dc&raw=%5B%5D")).toEqual({
      q: "acme freight",
      note: "a&b=c",
      raw: "[]",
    });
  });

  it("reads an empty query as no parameters", () => {
    expect(parseSearch("")).toEqual({});
    expect(parseSearch("?")).toEqual({});
  });
});

describe("stringifySearch", () => {
  it("writes strings as they are, leaving commas, colons and slashes readable", () => {
    expect(
      stringifySearch({
        fieldFilters: "[]",
        q: "acme freight",
        at: "2026-11-15T09:00",
        path: "a/b",
      }),
    ).toBe("?fieldFilters=[]&q=acme+freight&at=2026-11-15T09:00&path=a/b");
  });

  it("escapes only what would end or change a value", () => {
    expect(stringifySearch({ note: "50% off & more = #1? \"yes\" <b> 'x' `y` +z" })).toBe(
      "?note=50%25+off+%26+more+%3D+%231%3F+%22yes%22+%3Cb%3E+%27x%27+%60y%60+%2Bz",
    );
  });

  it("escapes control characters", () => {
    expect(stringifySearch({ line: "a\nb\tc" })).toBe("?line=a%0Ab%09c");
  });

  it("writes a list as the key repeated, and leaves out what has no value", () => {
    expect(stringifySearch({ status: ["open", "paid"], gone: undefined, none: null })).toBe(
      "?status=open&status=paid",
    );
  });

  it("writes numbers, booleans and objects a link was handed as text", () => {
    expect(stringifySearch({ page: 2, open: true, filter: { a: 1 } })).toBe(
      "?page=2&open=true&filter={%22a%22:1}",
    );
  });

  it("writes nothing for an empty query", () => {
    expect(stringifySearch({})).toBe("");
  });

  it("reads back exactly what it wrote", () => {
    const search = {
      q: "acme & sons = 50% #1",
      status: ["open", "paid"],
      id: "00123",
      filters: '[{"field":"name","value":"a b"}]',
    };

    expect(parseSearch(stringifySearch(search))).toEqual(search);
  });
});
