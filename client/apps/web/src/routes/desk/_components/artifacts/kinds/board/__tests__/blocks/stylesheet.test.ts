// jsdom applies no stylesheet, so these read the forms' rules as text: they guard the two
// layouts that broke in the browser. That the grid and log then sit right needs a browser.
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

/** Vitest runs from the web app's root; CSS imports are emptied in tests, so read the file. */
const SHEET = readFileSync(
  resolve(process.cwd(), "src/routes/desk/_styles/board-beyond.css"),
  "utf8",
);

describe("the forms' stylesheet", () => {
  it("never styles `dk-grid`, the Desk conversation's own layout grid", () => {
    expect(SHEET).not.toMatch(/\.dk-grid\b/);
    expect(SHEET).toMatch(
      /\.dk-bd-grid \{[^}]*grid-template-columns: 136px repeat\(var\(--n, 7\), minmax\(max-content, 1fr\)\)/,
    );
  });

  it("gives a log's entries one time column as wide as the widest time", () => {
    expect(SHEET).toMatch(
      /\.dk-bd-log\.dk-bd-mlog \{[^}]*grid-template-columns: max-content 14px minmax\(0, 1fr\)/,
    );
    expect(SHEET).toMatch(/\.dk-bd-log\.dk-bd-mlog li \{[^}]*grid-template-columns: subgrid/);
  });

  it("lets a row's fact shrink and clip, keeps a name whole beside words up to half the line, and never widens a short subtitle", () => {
    expect(SHEET).toMatch(/\.dk-bd-line > \.dk-bd-at \{[^}]*min-width: 0;[^}]*overflow: hidden/);
    expect(SHEET).toMatch(/\.dk-bd-line:has\(> \.dk-bd-at\) \{[^}]*overflow: hidden/);
    expect(SHEET).toMatch(/\.dk-bd-line:has\(> \.dk-bd-at\) > \.dk-bd-sub \{[^}]*min-width: 0;/);
    expect(SHEET).not.toMatch(/min-width: min\(/);
    expect(SHEET).toMatch(
      /\.dk-bd-line:has\(> \.dk-bd-at\.dk-bd-said\) > \.dk-bd-sub \{\s*flex: none;\s*max-width: 50%;/,
    );
  });
});
