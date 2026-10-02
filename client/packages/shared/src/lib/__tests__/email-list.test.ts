import { describe, expect, it } from "vitest";
import { invalidEmailAddresses, isEmailAddress, splitEmailList } from "../email-list";

describe("splitEmailList", () => {
  it("splits on commas, semicolons, new lines and tabs, dropping blanks and repeats", () => {
    expect(splitEmailList(" AP@Acme.com, ops@acme.com;\nap@acme.com\t\tbilling@acme.com ")).toEqual([
      "ap@acme.com",
      "ops@acme.com",
      "billing@acme.com",
    ]);
  });

  it("reads nothing from an empty field", () => {
    expect(splitEmailList("")).toEqual([]);
    expect(splitEmailList(null)).toEqual([]);
    expect(splitEmailList(" , ;")).toEqual([]);
  });
});

describe("invalidEmailAddresses", () => {
  it("names only the entries that are not addresses", () => {
    expect(invalidEmailAddresses("ap@acme.com, not-an-address, ops@acme")).toEqual([
      "not-an-address",
      "ops@acme",
    ]);
    expect(isEmailAddress("ap@acme.com")).toBe(true);
  });
});
