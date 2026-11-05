import { afterEach, describe, expect, it, vi } from "vitest";
import { openInNewTab } from "../open-in-new-tab";

afterEach(() => vi.restoreAllMocks());

describe("openInNewTab", () => {
  it("opens the address in a new tab with no access back to Trenova", () => {
    const open = vi.spyOn(window, "open").mockReturnValue(null);

    openInNewTab("https://storage.example.com/bol.pdf?signature=abc");

    expect(open).toHaveBeenCalledTimes(1);
    expect(open).toHaveBeenCalledWith(
      "https://storage.example.com/bol.pdf?signature=abc",
      "_blank",
      "noopener,noreferrer",
    );
  });
});
