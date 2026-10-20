import { describe, expect, it } from "vitest";
import {
  chunk,
  clampInteger,
  limitUtf8Bytes,
  downloadJsonFile,
  downloadTextFile,
  filenameFromDisposition,
  formatCurrency,
  formatCurrencyWhole,
  formatFileSize,
  formatLocation,
  getNameInitials,
  initials,
  metersToMiles,
  toTitleCase,
  truncateText,
  upperFirst,
} from "@trenova/shared/lib/utils";
import { afterEach, vi } from "vitest";

const originalCreateObjectUrlDescriptor = Object.getOwnPropertyDescriptor(URL, "createObjectURL");
const originalRevokeObjectUrlDescriptor = Object.getOwnPropertyDescriptor(URL, "revokeObjectURL");

afterEach(() => {
  if (originalCreateObjectUrlDescriptor) {
    Object.defineProperty(URL, "createObjectURL", originalCreateObjectUrlDescriptor);
  }
  if (originalRevokeObjectUrlDescriptor) {
    Object.defineProperty(URL, "revokeObjectURL", originalRevokeObjectUrlDescriptor);
  }
  vi.restoreAllMocks();
});

describe("toTitleCase", () => {
  it("splits camelCase", () => {
    expect(toTitleCase("firstName")).toBe("First Name");
  });

  it("splits underscores", () => {
    expect(toTitleCase("first_name")).toBe("First Name");
  });

  it("handles technical term ID", () => {
    expect(toTitleCase("userId")).toBe("User ID");
  });

  it("handles technical term URL", () => {
    expect(toTitleCase("requestUrl")).toBe("Request URL");
  });

  it("handles technical term API", () => {
    expect(toTitleCase("apiKey")).toBe("API Key");
  });

  it("handles technical term SQL", () => {
    expect(toTitleCase("sqlQuery")).toBe("SQL Query");
  });

  it("keeps lowercase words in the middle", () => {
    expect(toTitleCase("king_of_the_hill")).toBe("King of the Hill");
  });

  it("capitalizes first word even if lowercase", () => {
    expect(toTitleCase("a_new_hope")).toBe("A New Hope");
  });

  it("capitalizes last word even if lowercase", () => {
    expect(toTitleCase("something_to_think_of")).toBe("Something to Think Of");
  });

  it("handles 'At' after created", () => {
    expect(toTitleCase("createdAt")).toBe("Created At");
  });

  it("handles 'At' after updated", () => {
    expect(toTitleCase("updatedAt")).toBe("Updated At");
  });

  it("returns empty string for empty input", () => {
    expect(toTitleCase("")).toBe("");
  });

  it("handles single word", () => {
    expect(toTitleCase("hello")).toBe("Hello");
  });

  it("handles ALL_CAPS input", () => {
    expect(toTitleCase("FIRST_NAME")).toBe("First Name");
  });
});

describe("metersToMiles", () => {
  it("converts one mile worth of meters", () => {
    expect(metersToMiles(1609.344)).toBe(1);
  });

  it("returns 0 for 0 meters", () => {
    expect(metersToMiles(0)).toBe(0);
  });

  it("converts fractional distances", () => {
    expect(metersToMiles(804.672)).toBeCloseTo(0.5, 10);
  });
});

describe("upperFirst", () => {
  it("capitalizes first character", () => {
    expect(upperFirst("hello")).toBe("Hello");
  });

  it("returns empty string for empty input", () => {
    expect(upperFirst("")).toBe("");
  });

  it("leaves already capitalized unchanged", () => {
    expect(upperFirst("Hello")).toBe("Hello");
  });
});

describe("truncateText", () => {
  it("returns unchanged when under limit", () => {
    expect(truncateText("short", 10)).toBe("short");
  });

  it("truncates and adds ellipsis when over limit", () => {
    expect(truncateText("this is a long string", 7)).toBe("this is...");
  });

  it("returns empty string for empty input", () => {
    expect(truncateText("", 10)).toBe("");
  });
});

describe("formatCurrency", () => {
  it("formats USD by default", () => {
    expect(formatCurrency(1234.56)).toBe("$1,234.56");
  });

  it("formats zero", () => {
    expect(formatCurrency(0)).toBe("$0.00");
  });

  it("formats negative values", () => {
    expect(formatCurrency(-99.99)).toBe("-$99.99");
  });
});

describe("formatLocation", () => {
  it("returns empty string for undefined", () => {
    expect(formatLocation(undefined)).toBe("");
  });

  it("formats full address", () => {
    const location = {
      addressLine1: "123 Main St",
      addressLine2: "Suite 100",
      city: "Denver",
      state: { abbreviation: "CO" },
      postalCode: "80202",
    } as any;
    const result = formatLocation(location);
    expect(result).toBe("123 Main St, Suite 100, Denver, CO 80202");
  });

  it("handles missing addressLine2", () => {
    const location = {
      addressLine1: "123 Main St",
      addressLine2: null,
      city: "Denver",
      state: { abbreviation: "CO" },
      postalCode: "80202",
    } as any;
    const result = formatLocation(location);
    expect(result).toBe("123 Main St, Denver, CO 80202");
  });

  it("assembles all parts correctly", () => {
    const location = {
      addressLine1: "456 Elm Ave",
      addressLine2: "Apt 2",
      city: "Austin",
      state: { abbreviation: "TX" },
      postalCode: "73301",
    } as any;
    const result = formatLocation(location);
    expect(result).toBe("456 Elm Ave, Apt 2, Austin, TX 73301");
  });

  it("does not produce 'undefined' when state is null", () => {
    const location = {
      addressLine1: "123 Main St",
      addressLine2: null,
      city: "Denver",
      state: null,
      postalCode: "80202",
    } as any;
    const result = formatLocation(location);
    expect(result).not.toContain("undefined");
    expect(result).toBe("123 Main St, Denver 80202");
  });

  it("does not produce 'undefined' when state is undefined", () => {
    const location = {
      addressLine1: "123 Main St",
      addressLine2: "Suite 100",
      city: "Denver",
      state: undefined,
      postalCode: "80202",
    } as any;
    const result = formatLocation(location);
    expect(result).not.toContain("undefined");
    expect(result).toBe("123 Main St, Suite 100, Denver 80202");
  });

  it("does not produce 'undefined' when city is missing", () => {
    const location = {
      addressLine1: "123 Main St",
      addressLine2: null,
      city: undefined,
      state: { abbreviation: "CO" },
      postalCode: "80202",
    } as any;
    const result = formatLocation(location);
    expect(result).not.toContain("undefined");
    expect(result).toBe("123 Main St, CO 80202");
  });

  it("handles missing city and state gracefully", () => {
    const location = {
      addressLine1: "123 Main St",
      addressLine2: null,
      city: undefined,
      state: null,
      postalCode: "80202",
    } as any;
    const result = formatLocation(location);
    expect(result).not.toContain("undefined");
    expect(result).toBe("123 Main St, 80202");
  });

  it("handles missing postalCode gracefully", () => {
    const location = {
      addressLine1: "123 Main St",
      addressLine2: null,
      city: "Denver",
      state: { abbreviation: "CO" },
      postalCode: undefined,
    } as any;
    const result = formatLocation(location);
    expect(result).not.toContain("undefined");
    expect(result).toBe("123 Main St, Denver, CO");
  });
});

describe("initials", () => {
  it("returns both initials uppercased", () => {
    expect(initials("john", "doe")).toBe("JD");
  });

  it("returns one initial when last is undefined", () => {
    expect(initials("john", undefined)).toBe("J");
  });

  it("returns bullet when both undefined", () => {
    expect(initials(undefined, undefined)).toBe("•");
  });

  it("uppercases lowercase input", () => {
    expect(initials("alice", "bob")).toBe("AB");
  });
});

describe("download helpers", () => {
  it("downloads text files and revokes the object URL", () => {
    const click = vi.fn();
    const appendChild = vi.spyOn(document.body, "appendChild");
    const removeChild = vi.spyOn(document.body, "removeChild");
    const createObjectURL = vi.fn(() => "blob:test");
    const revokeObjectURL = vi.fn();
    URL.createObjectURL = createObjectURL;
    URL.revokeObjectURL = revokeObjectURL;
    vi.spyOn(document, "createElement").mockImplementation((tagName) => {
      const element = document.createElementNS("http://www.w3.org/1999/xhtml", tagName);
      if (tagName === "a") {
        Object.defineProperty(element, "click", { value: click });
      }
      return element as HTMLElement;
    });

    downloadTextFile("test.txt", "contents");

    expect(createObjectURL).toHaveBeenCalledWith(expect.any(Blob));
    expect(appendChild).toHaveBeenCalledWith(expect.objectContaining({ download: "test.txt" }));
    expect(click).toHaveBeenCalledOnce();
    expect(removeChild).toHaveBeenCalled();
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:test");
  });

  it("serializes JSON downloads with application/json", async () => {
    let blob: Blob | undefined;
    URL.createObjectURL = vi.fn((value) => {
      blob = value as Blob;
      return "blob:json";
    });
    URL.revokeObjectURL = vi.fn();
    vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});

    downloadJsonFile("data.json", { ok: true });

    expect(blob?.type).toBe("application/json");
    await expect(blob?.text()).resolves.toBe(JSON.stringify({ ok: true }, null, 2));
  });
});

describe("getNameInitials", () => {
  it("takes one letter per word by default", () => {
    expect(getNameInitials("Alpha Logistics")).toBe("AL");
    expect(getNameInitials("Meridian Freight Group")).toBe("MF");
  });

  it("leaves a one-word name as a single letter by default", () => {
    expect(getNameInitials("Trenova")).toBe("T");
  });

  it("falls back when there is no name", () => {
    expect(getNameInitials(undefined)).toBe("U");
    expect(getNameInitials("", "ORG")).toBe("ORG");
  });

  // The sign-in screen's organization tile is a fixed-width three-letter code, so a
  // short name is padded out of the first word rather than rendered half empty.
  it("pads a short result from the first word when asked", () => {
    expect(getNameInitials("Meridian Freight Group", "ORG", { maxLength: 3, pad: true })).toBe(
      "MFG",
    );
    expect(getNameInitials("Trenova Logistics", "ORG", { maxLength: 3, pad: true })).toBe("TRL");
    expect(getNameInitials("Trenova Transportation", "ORG", { maxLength: 3, pad: true })).toBe(
      "TRT",
    );
    expect(getNameInitials("Trenova", "ORG", { maxLength: 3, pad: true })).toBe("TRE");
  });

  it("collapses runs of whitespace rather than emitting a blank initial", () => {
    expect(getNameInitials("Alpha   Beta Gamma", "ORG", { maxLength: 3, pad: true })).toBe("ABG");
  });
});

describe("chunk", () => {
  it("splits into consecutive groups of the given size, keeping order", () => {
    expect(chunk(["a", "b", "c", "d", "e"], 2)).toEqual([["a", "b"], ["c", "d"], ["e"]]);
  });

  it("returns one group when the size covers every item", () => {
    expect(chunk([1, 2, 3], 3)).toEqual([[1, 2, 3]]);
    expect(chunk([1, 2, 3], 10)).toEqual([[1, 2, 3]]);
  });

  it("returns no groups for no items", () => {
    expect(chunk([], 5)).toEqual([]);
  });

  it("rejects a size that could never make progress", () => {
    expect(() => chunk([1], 0)).toThrow(RangeError);
    expect(() => chunk([1], -1)).toThrow(RangeError);
    expect(() => chunk([1], 1.5)).toThrow(RangeError);
  });
});

describe("formatFileSize", () => {
  it("picks the unit and keeps one decimal below 100", () => {
    expect(formatFileSize(0)).toBe("0 B");
    expect(formatFileSize(512)).toBe("512 B");
    expect(formatFileSize(1536)).toBe("1.5 KB");
    expect(formatFileSize(150 * 1024 * 1024)).toBe("150 MB");
  });
});

describe("formatCurrencyWhole", () => {
  it("drops the cents when there are none, and keeps both digits when there are", () => {
    expect(formatCurrencyWhole(185)).toBe("$185");
    expect(formatCurrencyWhole(1180)).toBe("$1,180");
    expect(formatCurrencyWhole(185.5)).toBe("$185.50");
    expect(formatCurrencyWhole(0)).toBe("$0");
  });
});

describe("filenameFromDisposition", () => {
  it("reads a quoted file name", () => {
    expect(
      filenameFromDisposition('attachment; filename="Statement-ACME-2026-08-31.pdf"', "x.pdf"),
    ).toBe("Statement-ACME-2026-08-31.pdf");
  });

  it("reads an unquoted file name", () => {
    expect(filenameFromDisposition("attachment; filename=report.xlsx", "x.xlsx")).toBe(
      "report.xlsx",
    );
  });

  it("prefers the encoded name a server sends for non-ASCII names", () => {
    expect(
      filenameFromDisposition(
        "attachment; filename*=utf-8''Statement-%C3%89lan-2026-08-31.pdf",
        "x.pdf",
      ),
    ).toBe("Statement-Élan-2026-08-31.pdf");
  });

  it("falls back when there is no header or no name", () => {
    expect(filenameFromDisposition(null, "fallback.pdf")).toBe("fallback.pdf");
    expect(filenameFromDisposition("attachment", "fallback.pdf")).toBe("fallback.pdf");
    expect(filenameFromDisposition("attachment; filename*=utf-8''%E0%A4%A", "f.pdf")).toBe("f.pdf");
  });
});

describe("limitUtf8Bytes", () => {
  it("keeps text that fits and cuts the rest between whole characters", () => {
    expect(limitUtf8Bytes("Northline Foods", 80)).toBe("Northline Foods");
    expect(limitUtf8Bytes("abcdef", 4)).toBe("abcd");
    expect(limitUtf8Bytes("瞭望台佇列", 7)).toBe("瞭望");
    expect(new TextEncoder().encode(limitUtf8Bytes("é".repeat(100), 61)).length).toBe(60);
    expect(limitUtf8Bytes("a😀b", 4)).toBe("a");
    expect(limitUtf8Bytes("abc", 0)).toBe("");
  });
});

describe("clampInteger", () => {
  it("rounds a number inside the bounds to a whole one", () => {
    expect(clampInteger(231.6, 200, 340, 232)).toBe(232);
  });

  it("holds a number outside the bounds at the nearer one", () => {
    expect(clampInteger(120, 200, 340, 232)).toBe(200);
    expect(clampInteger(900, 200, 340, 232)).toBe(340);
  });

  it("reads anything that is not a number as the fallback", () => {
    expect(clampInteger(Number.NaN, 200, 340, 232)).toBe(232);
    expect(clampInteger(Number.POSITIVE_INFINITY, 200, 340, 232)).toBe(232);
  });
});
