import { describe, expect, it } from "vitest";
import { formatPhoneForDisplay } from "../phone";

describe("formatPhoneForDisplay", () => {
  it("groups a North American number stored in E.164", () => {
    expect(formatPhoneForDisplay("+18155550142")).toBe("(815) 555-0142");
  });

  it("groups ten bare digits and an already grouped number the same way", () => {
    expect(formatPhoneForDisplay("8155550142")).toBe("(815) 555-0142");
    expect(formatPhoneForDisplay("(815) 555-0142")).toBe("(815) 555-0142");
  });

  it("leaves a number from another country as it is stored", () => {
    expect(formatPhoneForDisplay("+442071838750")).toBe("+442071838750");
    expect(formatPhoneForDisplay("+5215512345678")).toBe("+5215512345678");
  });

  it("leaves anything that is not ten national digits alone", () => {
    expect(formatPhoneForDisplay("555-0142")).toBe("555-0142");
    expect(formatPhoneForDisplay("")).toBe("");
  });
});
