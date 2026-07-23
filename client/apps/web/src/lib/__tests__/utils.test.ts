import { describe, expect, it } from "vitest";
import {
  downloadJsonFile,
  downloadTextFile,
  toTitleCase,
  pluralize,
  upperFirst,
  truncateText,
  formatCurrency,
  formatLocation,
  initials,
} from "../utils";
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

describe("pluralize", () => {
  it("returns singular when count is 1", () => {
    expect(pluralize("item", 1)).toBe("item");
  });

  it("returns plural when count is 0", () => {
    expect(pluralize("item", 0)).toBe("items");
  });

  it("returns plural when count is 2", () => {
    expect(pluralize("item", 2)).toBe("items");
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
