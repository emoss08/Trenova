import assert from "node:assert/strict";
import { mkdir, mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { after, before, describe, it } from "node:test";
import { catalogBundle, WEB_BUNDLE } from "./bundles.mjs";
import {
  extractFromSource,
  extractRust,
  RULES_CLIENT_AREA,
  RustExtractError,
  unescape,
} from "./extract-rust.mjs";

const templates = (source) => extractFromSource(source, "t.rs").messages.map((m) => m.message);

describe("extractFromSource", () => {
  it("reads plain literals with their line", () => {
    const { messages } = extractFromSource(
      `fn f() {\n    let m = message!("unknown name {0}");\n}\n`,
      "t.rs",
    );
    assert.deepEqual(messages, [{ message: "unknown name {0}", line: 2 }]);
  });

  it("decodes escapes as the compiler does", () => {
    assert.deepEqual(
      templates(String.raw`
        message!("say \"{0}\"");
        message!("a back\\slash");
        message!("caf\u{e9} \x41");
        message!("Add ';'");
        message!("Write '{' and '}' with quotes");
      `),
      ['say "{0}"', "a back\\slash", "café A", "Add ';'", "Write '{' and '}' with quotes"],
    );
  });

  it("joins a literal continued across lines with a trailing backslash", () => {
    assert.deepEqual(
      templates(`message!("{0} is not a function; \\\n        did you mean {1}?");`),
      ["{0} is not a function; did you mean {1}?"],
    );
  });

  it("refuses a literal with a newline in it, which no catalog key could match", () => {
    assert.throws(
      () => templates(`message!("first line\nsecond line");`),
      (error) => error instanceof RustExtractError && /newlines/.test(error.message),
    );
    assert.throws(() => templates(String.raw`message!("tab\tseparated");`), RustExtractError);
    assert.throws(() => templates(`message!(" padded");`), RustExtractError);
  });

  it("reads raw strings verbatim", () => {
    assert.deepEqual(
      templates(`
        message!(r"a \\d+ pattern");
        message!(r#"say "{0}" here"#);
        message!(r##"a "# inside"##);
      `),
      ["a \\d+ pattern", 'say "{0}" here', 'a "# inside'],
    );
  });

  it("accepts any macro delimiter and a path-qualified call", () => {
    assert.deepEqual(
      templates(`crate::message!["one"]; $crate::message!{"two"}; message! ("three");`),
      ["one", "two", "three"],
    );
  });

  it("refuses an argument that is not one string literal", () => {
    for (const source of [
      `message!(concat!("a", "b"));`,
      `message!(TEMPLATE);`,
      `message!("{0}", name);`,
      `message!("a",);`,
      `message!();`,
      `message!(b"bytes");`,
      `message!(c"c string");`,
    ]) {
      assert.throws(
        () => templates(source),
        (error) => error instanceof RustExtractError && error.message.startsWith("t.rs:1: "),
        source,
      );
    }
  });

  it("skips comments, doc comments, strings and char literals", () => {
    assert.deepEqual(
      templates(`
        //! Every sentence is written as \`message!("...")\`.
        /// message!("doc")
        // message!("line")
        /* message!("block") /* nested message!("deeper") */ still comment */
        let s = "message!(\\"in a string\\")";
        let r = r#"message!("in a raw string")"#;
        let q = '"'; let e = '\\''; let l: &'static str = "";
        let real = message!("real");
      `),
      ["real"],
    );
  });

  it("skips #[cfg(test)] items and records test modules declared as files", () => {
    const result = extractFromSource(
      `
        pub fn a() -> &'static str { message!("kept") }

        #[cfg(test)]
        mod tests {
            fn t() { let _ = message!("test only"); }
        }

        #[cfg(all(test, feature = "x"))]
        fn helper() { message!("also test only"); }

        #[cfg(not(test))]
        fn production() { message!("not test"); }

        #[cfg(test)]
        mod fixtures;

        #[cfg(test)]
        #[path = "x.rs"]
        mod elsewhere;
      `,
      "t.rs",
    );
    assert.deepEqual(
      result.messages.map((m) => m.message),
      ["kept", "not test"],
    );
    assert.deepEqual(result.testModules, [
      { name: "fixtures", path: null },
      { name: "elsewhere", path: "x.rs" },
    ]);
  });

  it("skips a file that is test-only as a whole", () => {
    assert.deepEqual(templates(`#![cfg(test)]\nfn f() { message!("never"); }\n`), []);
  });

  it("ignores a macro definition and a comparison named message", () => {
    assert.deepEqual(
      templates(`
        macro_rules! message { ($text:literal) => { $text }; }
        if message != other { }
      `),
      [],
    );
  });
});

describe("unescape", () => {
  it("rejects escapes Rust rejects", () => {
    assert.throws(() => unescape(String.raw`\q`), RustExtractError);
    assert.throws(() => unescape(String.raw`\xff`), RustExtractError);
  });
});

describe("extractRust", () => {
  let root;

  before(async () => {
    root = await mkdtemp(join(tmpdir(), "extract-rust-"));
    const src = join(root, "services", "crates", "rules", "src");
    await mkdir(join(src, "formula", "fixtures"), { recursive: true });
    await writeFile(join(src, "lib.rs"), `pub mod formula;\n`);
    await writeFile(
      join(src, "formula.rs"),
      `pub mod check;\n#[cfg(test)]\nmod fixtures;\n`,
    );
    await writeFile(
      join(src, "formula", "check.rs"),
      `pub fn f() { message!("unknown name {0}"); message!("{0}"); }\n`,
    );
    await writeFile(join(src, "formula", "fixtures.rs"), `fn f() { message!("fixture"); }\n`);
    await writeFile(
      join(src, "formula", "fixtures", "more.rs"),
      `fn f() { message!("nested fixture"); }\n`,
    );
  });

  after(async () => {
    await rm(root, { recursive: true, force: true });
  });

  it("sends each template to the Go scope and the web bundle, and skips test module files", async () => {
    const { entries, rejected } = await extractRust(root);
    const file = join("services", "crates", "rules", "src", "formula", "check.rs");

    assert.deepEqual(entries, [
      { message: "unknown name {0}", file, line: 1, kind: "rules-message", area: "rules/formula", scope: "go" },
      { message: "unknown name {0}", file, line: 1, kind: "rules-message", area: RULES_CLIENT_AREA, scope: "ts" },
    ]);
    assert.equal(catalogBundle([RULES_CLIENT_AREA, "rules/formula"]), WEB_BUNDLE);
    assert.deepEqual(
      rejected.map((r) => [r.value, r.reason]),
      [["{0}", "no letters"]],
    );
  });

  it("finds nothing under a root that does not exist", async () => {
    assert.deepEqual(await extractRust(root, ["services/crates/missing/src"]), {
      entries: [],
      rejected: [],
    });
  });
});
