// The rules crate's regression corpora (services/crates/rules/testdata/*.json) for TypeScript
// tests, so a test of a wrapper or a screen asserts the contract the crate is pinned to rather
// than whatever the code happens to return. Only tests import this.
//
// A case's `output` is a pattern, matched as the crate's own harness matches it
// (tests/rules_support.rs): an object matches when each of its keys matches (the string
// "$absent" requires the key to be missing), an array when it has the same length and each
// element matches, anything else when it is equal; `exact` requires the whole value.

import allocation from "../../../../../../services/crates/rules/testdata/allocation.json";
import lane from "../../../../../../services/crates/rules/testdata/lane.json";
import matrix from "../../../../../../services/crates/rules/testdata/matrix.json";
import report from "../../../../../../services/crates/rules/testdata/report.json";
import { expect } from "vitest";

export type CorpusCase = {
  readonly name: string;
  readonly op?: string;
  readonly input: unknown;
  readonly output?: unknown;
  readonly exact?: boolean;
  readonly error?: string;
};

type Corpus = { readonly op: string; readonly cases: readonly CorpusCase[] };

const CORPORA = {
  allocation: allocation as unknown as Corpus,
  lane: lane as unknown as Corpus,
  matrix: matrix as unknown as Corpus,
  report: report as unknown as Corpus,
};

/** One named case of an area's corpus. Throws when the corpus has no such case. */
export function corpusCase(area: keyof typeof CORPORA, name: string): CorpusCase {
  const found = CORPORA[area].cases.find((candidate) => candidate.name === name);
  if (!found) throw new Error(`rules corpus ${area} has no case named ${JSON.stringify(name)}`);
  return found;
}

/** Every way `actual` fails the pattern `expected`, as `path: reason`; empty when it matches. */
export function corpusMismatches(expected: unknown, actual: unknown): string[] {
  const found: string[] = [];
  match(expected, actual, true, "ok", found);
  return found;
}

/** Asserts that `actual` is the case's output: equal when the case is exact, else matching. */
export function expectCorpusOutput(testCase: CorpusCase, actual: unknown): void {
  if (testCase.exact) {
    expect(actual).toEqual(testCase.output);
    return;
  }
  expect(corpusMismatches(testCase.output, actual)).toEqual([]);
}

function match(
  expected: unknown,
  actual: unknown,
  present: boolean,
  path: string,
  found: string[],
): void {
  if (expected === "$absent") {
    if (present) found.push(`${path}: expected absent, got ${JSON.stringify(actual)}`);
    return;
  }
  if (!present) {
    found.push(`${path}: missing, expected ${JSON.stringify(expected)}`);
    return;
  }
  if (Array.isArray(expected)) {
    if (!Array.isArray(actual) || actual.length !== expected.length) {
      found.push(`${path}: expected ${expected.length} items, got ${JSON.stringify(actual)}`);
      return;
    }
    expected.forEach((item, index) => match(item, actual[index], true, `${path}[${index}]`, found));
    return;
  }
  if (expected !== null && typeof expected === "object") {
    if (actual === null || typeof actual !== "object" || Array.isArray(actual)) {
      found.push(`${path}: expected an object, got ${JSON.stringify(actual)}`);
      return;
    }
    const record = actual as Record<string, unknown>;
    for (const [key, value] of Object.entries(expected)) {
      match(value, record[key], key in record, `${path}.${key}`, found);
    }
    return;
  }
  if (expected !== actual) {
    found.push(`${path}: expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
}
