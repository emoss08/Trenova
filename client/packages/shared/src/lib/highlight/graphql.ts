import {
  defineLanguage,
  type HighlightTokenClass,
  type TokenRange,
} from "@tanstack/highlight/core";

// TanStack Highlight ships no GraphQL grammar. This one reads a document left to right in one
// pass, so a "#" inside a string is never a comment and a name inside a block string is never
// a field. Definition keywords ("type", "query", "input") only count at the top level, where a
// definition can start: inside a selection set "type" is a field a shipment or location has.

const TOKEN =
  /(?<comment>#[^\n]*)|(?<blockString>"""(?:\\"""|[\s\S])*?(?:"""|$))|(?<string>"(?:\\.|[^"\\\n])*"?)|(?<variable>\$[_A-Za-z][_0-9A-Za-z]*)|(?<directive>@[_A-Za-z][_0-9A-Za-z]*)|(?<number>-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)|(?<spread>\.\.\.)|(?<name>[_A-Za-z][_0-9A-Za-z]*)|(?<open>\{)|(?<close>\})/g;

const DEFINITION_KEYWORDS = new Set([
  "directive",
  "enum",
  "extend",
  "fragment",
  "input",
  "interface",
  "mutation",
  "query",
  "scalar",
  "schema",
  "subscription",
  "type",
  "union",
]);

const CLAUSE_KEYWORDS = new Set(["implements", "on", "repeatable"]);

const LITERALS = new Set(["false", "null", "true"]);

const FOLLOWED_BY_COLON = /^\s*:/;

function nameClass(name: string, rest: string, depth: number): HighlightTokenClass | undefined {
  if (LITERALS.has(name)) return "literal";
  if (depth === 0 && DEFINITION_KEYWORDS.has(name)) return "keyword";
  if (CLAUSE_KEYWORDS.has(name) && /^\s+[_A-Za-z]/.test(rest)) return "keyword";
  if (FOLLOWED_BY_COLON.test(rest)) return "attr";
  if (/^[A-Z]/.test(name)) return "type";
  return undefined;
}

export function tokenizeGraphQL(code: string): TokenRange[] {
  const ranges: TokenRange[] = [];
  let depth = 0;
  const push = (className: HighlightTokenClass | undefined, start: number, end: number) => {
    if (className) ranges.push({ className, start, end });
  };

  for (const match of code.matchAll(TOKEN)) {
    const groups = match.groups ?? {};
    const start = match.index;
    const end = start + match[0].length;

    if (groups.open !== undefined) {
      depth += 1;
    } else if (groups.close !== undefined) {
      depth = Math.max(0, depth - 1);
    } else if (groups.comment !== undefined) {
      push("comment", start, end);
    } else if (groups.blockString !== undefined || groups.string !== undefined) {
      push("string", start, end);
    } else if (groups.variable !== undefined) {
      push("variable", start, end);
    } else if (groups.directive !== undefined) {
      push("meta", start, end);
    } else if (groups.number !== undefined) {
      push("number", start, end);
    } else if (groups.spread !== undefined) {
      push("operator", start, end);
    } else if (groups.name !== undefined) {
      push(nameClass(groups.name, code.slice(end, end + 64), depth), start, end);
    }
  }
  return ranges;
}

export const graphql = defineLanguage({
  name: "graphql",
  aliases: ["gql", "graphqls"],
  tokenize: tokenizeGraphQL,
});
