import { describe, expect, it } from "vitest";
import { cityStatePostal, mailingAddressLines } from "../mailing-address";

describe("cityStatePostal", () => {
  it("reads like an envelope and leaves out whatever is blank", () => {
    expect(cityStatePostal("Dallas", "TX", "75201")).toBe("Dallas, TX 75201");
    expect(cityStatePostal("Dallas", "", "")).toBe("Dallas");
    expect(cityStatePostal("", "TX", "75201")).toBe("TX 75201");
    expect(cityStatePostal(" Dallas ", null, " 75201 ")).toBe("Dallas, 75201");
    expect(cityStatePostal(null, undefined, "")).toBe("");
  });
});

describe("mailingAddressLines", () => {
  it("stacks the lines a document prints, country last, blanks left out", () => {
    expect(
      mailingAddressLines({
        line1: "10208 Main Blvd",
        line2: "  ",
        city: "Jacksonville",
        state: "FL",
        postalCode: "32202",
        country: "United States",
      }),
    ).toEqual(["10208 Main Blvd", "Jacksonville, FL 32202", "United States"]);
    expect(mailingAddressLines({})).toEqual([]);
  });
});
