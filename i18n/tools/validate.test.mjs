import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { translationProblems } from "./validate.mjs";

describe("translationProblems", () => {
  it("accepts a sound translation that reorders its placeholders and tags", () => {
    assert.deepEqual(
      translationProblems("You've used <b>{0}</b> of {1}'s budget", "{1} 的预算已用 <b>{0}</b>"),
      [],
    );
  });

  it("accepts a plural that keeps its header and changes its branches", () => {
    assert.deepEqual(
      translationProblems("{0, plural, one {# stop} other {# stops}}", "{0, plural, other {# 个停靠点}}"),
      [],
    );
  });

  it("refuses a dropped or duplicated placeholder", () => {
    assert.match(translationProblems("Sent to {0} on {1}", "Enviado a {0}")[0], /placeholders/);
    assert.match(translationProblems("Sent to {0}", "Enviado a {0} {0}")[0], /placeholders/);
  });

  it("refuses a renamed plural", () => {
    assert.match(
      translationProblems("{0, plural, one {# stop} other {# stops}}", "{0, plurals, other {# paradas}}")[0],
      /plural/,
    );
  });

  it("refuses a lost or broken tag", () => {
    assert.match(translationProblems("Use <link>shared profiles</link>", "Use perfiles compartidos")[0], /tags/);
    assert.match(translationProblems("Press <kbd/> to approve", "Pulse <kbd> para aprobar")[0], /tags/);
  });

  it("does not read a less-than sign in prose as a tag", () => {
    assert.deepEqual(translationProblems("Weight < 10 lb", "Peso < 10 lb"), []);
  });

  it("refuses an empty value and unbalanced braces", () => {
    assert.deepEqual(translationProblems("Save", "  "), ["is empty"]);
    assert.ok(translationProblems("{0} left", "{0} restantes}").includes("unbalanced braces"));
  });
});
