import { describe, expect, it } from "vitest";
import { shellLoadsBuild } from "../manifest";

const manifest = { entry: "/assets/js/main.DcyZLU6s.js" };

describe("shellLoadsBuild", () => {
  it("accepts the HTML of the manifest's build", () => {
    expect(
      shellLoadsBuild(
        '<script type="module" crossorigin src="/assets/js/main.DcyZLU6s.js"></script>',
        manifest,
      ),
    ).toBe(true);
  });

  it("refuses the previous build's HTML that a CDN can still serve during a release", () => {
    expect(
      shellLoadsBuild(
        '<script type="module" crossorigin src="/assets/js/main.Cr7o0Rsn.js"></script>',
        manifest,
      ),
    ).toBe(false);
  });

  it("refuses a page that only mentions the entry's name somewhere else", () => {
    expect(shellLoadsBuild("<p>main.DcyZLU6s.js was the old build</p>", manifest)).toBe(false);
  });
});
