import { describe, expect, it } from "vitest";
import { sizedSvg } from "../svg-source";

function svgFile(markup: string): File {
  return new File([markup], "mark.svg", { type: "image/svg+xml" });
}

async function rootOf(file: File): Promise<Element> {
  return new DOMParser().parseFromString(await file.text(), "image/svg+xml").documentElement;
}

describe("a vector picture given a size to be drawn at", () => {
  it("takes a large size in its own shape when it has only a view box", async () => {
    const sized = await sizedSvg(
      svgFile(
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 48 24"><rect width="48" height="24"/></svg>',
      ),
    );
    const root = await rootOf(sized);

    expect(root.getAttribute("width")).toBe("2048");
    expect(root.getAttribute("height")).toBe("1024");
    expect(root.getAttribute("viewBox")).toBe("0 0 48 24");
    expect(sized.name).toBe("mark.svg");
    expect(sized.type).toBe("image/svg+xml");
  });

  it("is enlarged from an icon's size, with a view box added so its drawing scales", async () => {
    const sized = await sizedSvg(
      svgFile(
        '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24px"><circle r="12"/></svg>',
      ),
    );
    const root = await rootOf(sized);

    expect(root.getAttribute("viewBox")).toBe("0 0 24 24");
    expect(root.getAttribute("width")).toBe("2048");
    expect(root.getAttribute("height")).toBe("2048");
  });

  it("follows the view box, not a size stated in other units", async () => {
    const sized = await sizedSvg(
      svgFile(
        '<svg xmlns="http://www.w3.org/2000/svg" width="100%" height="5em" viewBox="0 0 10 40"/>',
      ),
    );
    const root = await rootOf(sized);

    expect(root.getAttribute("width")).toBe("512");
    expect(root.getAttribute("height")).toBe("2048");
  });

  it("is left as it is with neither a view box nor a size in pixels", async () => {
    const file = svgFile('<svg xmlns="http://www.w3.org/2000/svg" width="50%"><rect/></svg>');

    expect(await sizedSvg(file)).toBe(file);
  });

  it("refuses a file that is not an SVG document", async () => {
    await expect(sizedSvg(svgFile("<html><body>not a picture</body></html>"))).rejects.toThrow();
  });
});
