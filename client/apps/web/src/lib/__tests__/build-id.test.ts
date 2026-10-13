import { describe, expect, it, vi } from "vitest";
import { buildId, deployedBuildId } from "../build-id";

const html = (entry: string) =>
  `<!doctype html><html><head><script type="module" crossorigin src="${entry}"></script></head></html>`;

function documentWith(markup: string): Pick<Document, "querySelector"> {
  const doc = document.implementation.createHTMLDocument("");
  doc.documentElement.innerHTML = markup;
  return doc;
}

describe("buildId", () => {
  it("names the running build by its entry script", () => {
    expect(buildId(documentWith(html("/assets/js/main.abc.js")))).toBe("/assets/js/main.abc.js");
  });

  it("reads 'development' on the dev server, whose entry never changes", () => {
    expect(buildId(documentWith(html("/src/main.tsx")))).toBe("development");
    expect(buildId(documentWith("<head></head>"))).toBe("development");
  });
});

describe("deployedBuildId", () => {
  it("reads the deployed build from the server's index.html, past every cache", async () => {
    const fetchPage = vi.fn(async () => new Response(html("/assets/js/main.new.js")));

    expect(await deployedBuildId(fetchPage, "/")).toBe("/assets/js/main.new.js");
    expect(fetchPage).toHaveBeenCalledWith("/", { cache: "no-store", credentials: "same-origin" });
  });

  it("knows nothing when the server cannot be asked or answers with an error", async () => {
    expect(
      await deployedBuildId(async () => Promise.reject(new TypeError("offline")), "/"),
    ).toBeNull();
    expect(await deployedBuildId(async () => new Response("", { status: 502 }), "/")).toBeNull();
    expect(await deployedBuildId(async () => new Response("<html></html>"), "/")).toBeNull();
  });
});
