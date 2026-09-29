import { afterEach, describe, expect, it, vi } from "vitest";
import { openCoverSheetsForPrinting } from "../capture-cover-sheet";

// window.open with the "noopener" feature returns null by spec, so a caller
// that asks for it can never tell an opened tab from a blocked one.

describe("openCoverSheetsForPrinting", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  function stubObjectUrls() {
    vi.stubGlobal("URL", {
      ...URL,
      createObjectURL: vi.fn(() => "blob:sheets"),
      revokeObjectURL: vi.fn(),
    });
  }

  it("opens a tab it can detect, cuts its opener, and does not also download", () => {
    stubObjectUrls();
    const tab = { opener: window } as unknown as Window;
    const open = vi.spyOn(window, "open").mockReturnValue(tab);
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});

    const result = openCoverSheetsForPrinting(new Uint8Array([37, 80, 68, 70]), "sheets.pdf");

    expect(open).toHaveBeenCalledTimes(1);
    const features = open.mock.calls[0]?.[2];
    expect(features ?? "").not.toMatch(/noopener/);
    expect(tab.opener).toBeNull();
    expect(click).not.toHaveBeenCalled();
    expect(result).toBe("opened");
  });

  it("downloads the sheets when the browser blocked the tab", () => {
    stubObjectUrls();
    vi.spyOn(window, "open").mockReturnValue(null);
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});

    const result = openCoverSheetsForPrinting(new Uint8Array([37, 80, 68, 70]), "sheets.pdf");

    expect(click).toHaveBeenCalledTimes(1);
    expect(result).toBe("downloaded");
  });
});
