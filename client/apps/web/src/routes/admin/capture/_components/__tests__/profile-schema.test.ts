import { describe, expect, it } from "vitest";
import { newProfileDefaults, profileFormSchema, profileInput } from "../profile-schema";

function parse(overrides: Record<string, unknown>) {
  return profileFormSchema.safeParse({ ...newProfileDefaults(), name: "Paperwork", ...overrides });
}

describe("profileFormSchema", () => {
  it("accepts the defaults once named", () => {
    expect(parse({}).success).toBe(true);
  });

  it("refuses an inactive default, as the server does", () => {
    const result = parse({ isDefault: true, status: "Inactive" });
    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.path).toEqual(["isDefault"]);
  });

  it("needs a page count when splitting by count, and not otherwise", () => {
    expect(parse({ separatorStrategies: ["FixedPageCount"], fixedPageCount: 0 }).success).toBe(
      false,
    );
    expect(parse({ separatorStrategies: ["FixedPageCount"], fixedPageCount: 2 }).success).toBe(
      true,
    );
    expect(parse({ separatorStrategies: ["PatchCode"], fixedPageCount: 0 }).success).toBe(true);
  });

  it("reads an empty checkbox group as no separators", () => {
    const result = parse({ separatorStrategies: null });
    expect(result.success).toBe(true);
    expect(result.data?.separatorStrategies).toEqual([]);
  });

  it("keeps image quality where the server does", () => {
    expect(parse({ jpegQuality: 29 }).success).toBe(false);
    expect(parse({ jpegQuality: 96 }).success).toBe(false);
  });
});

describe("profileInput", () => {
  it("drops a page count the server would refuse and sends no blank description", () => {
    const values = profileFormSchema.parse({
      ...newProfileDefaults(),
      name: "  Legal  ",
      description: "   ",
      separatorStrategies: ["PatchCode"],
      fixedPageCount: 4,
    });
    expect(profileInput(values)).toMatchObject({
      name: "Legal",
      description: null,
      separatorStrategies: ["PatchCode"],
      fixedPageCount: 0,
    });
  });
});
