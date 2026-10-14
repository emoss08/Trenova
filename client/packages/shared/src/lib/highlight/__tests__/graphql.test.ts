import { tokenizeGraphQL } from "@trenova/shared/lib/highlight/graphql";
import { describe, expect, it } from "vitest";

/** Each coloured run as [text, role], in order, so a test reads like the document it checks. */
function roles(code: string): Array<[string, string]> {
  return tokenizeGraphQL(code).map(({ start, end, className }) => [
    code.slice(start, end),
    className,
  ]);
}

describe("tokenizeGraphQL", () => {
  it("colours an operation's keyword, variables, argument names, types and directives", () => {
    const code =
      "query Shipment($id: ID!, $full: Boolean = false) {\n  shipment(id: $id) @include(if: $full) { proNumber status }\n}";

    expect(roles(code)).toEqual([
      ["query", "keyword"],
      ["Shipment", "type"],
      ["$id", "variable"],
      ["ID", "type"],
      ["$full", "variable"],
      ["Boolean", "type"],
      ["false", "literal"],
      ["id", "attr"],
      ["$id", "variable"],
      ["@include", "meta"],
      ["if", "attr"],
      ["$full", "variable"],
    ]);
  });

  it("leaves a field named like a definition keyword plain inside a selection set", () => {
    const code = "query { location { type name } shipment { query input } }";

    expect(roles(code)).toEqual([["query", "keyword"]]);
  });

  it("reads type definitions, field types and an inline fragment's type condition", () => {
    const code =
      "type Location implements Node {\n  type: String!\n  stops: [Stop!]!\n}\nfragment F on Shipment { ... on Move { id } }";

    expect(roles(code)).toEqual([
      ["type", "keyword"],
      ["Location", "type"],
      ["implements", "keyword"],
      ["Node", "type"],
      ["type", "attr"],
      ["String", "type"],
      ["stops", "attr"],
      ["Stop", "type"],
      ["fragment", "keyword"],
      ["F", "type"],
      ["on", "keyword"],
      ["Shipment", "type"],
      ["...", "operator"],
      ["on", "keyword"],
      ["Move", "type"],
    ]);
  });

  it("never reads a comment, field or brace inside a string", () => {
    const code = 'mutation { note(text: "a # b { type }", body: """multi\nline { query }""") }';

    expect(roles(code)).toEqual([
      ["mutation", "keyword"],
      ["text", "attr"],
      ['"a # b { type }"', "string"],
      ["body", "attr"],
      ['"""multi\nline { query }"""', "string"],
    ]);
  });

  it("colours a comment to the end of its line and numbers in either notation", () => {
    const code = "# first page\nquery { list(first: 25, ratio: -1.5e3) }";

    expect(roles(code)).toEqual([
      ["# first page", "comment"],
      ["query", "keyword"],
      ["first", "attr"],
      ["25", "number"],
      ["ratio", "attr"],
      ["-1.5e3", "number"],
    ]);
  });

  it("colours an unterminated string to the end of what has arrived", () => {
    expect(roles('query { a(b: "still typ')).toEqual([
      ["query", "keyword"],
      ["b", "attr"],
      ['"still typ', "string"],
    ]);
    expect(roles('"""open block\nstill open')).toEqual([['"""open block\nstill open', "string"]]);
  });
});
