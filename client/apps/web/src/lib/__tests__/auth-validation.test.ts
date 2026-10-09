import { describe, expect, it } from "vitest";
import { emailSchema, scorePassword } from "../auth-validation";

describe("emailSchema", () => {
  const schema = emailSchema(() => "Enter a valid work email.");

  it("accepts an address and trims what surrounds it", () => {
    expect(schema.parse("  jordan@granitefreight.com ")).toBe("jordan@granitefreight.com");
  });

  it.each(["", "   ", "jordan", "jordan@", "jordan@granitefreight", "@granitefreight.com", "jo rdan@x.com"])(
    "refuses %j with the caller's message",
    (value) => {
      const result = schema.safeParse(value);
      expect(result.success).toBe(false);
      expect(result.error?.issues[0]?.message).toBe("Enter a valid work email.");
    },
  );

  it("reads the message when it validates, not when the schema is built", () => {
    let message = "first";
    const lazy = emailSchema(() => message);
    message = "second";
    expect(lazy.safeParse("nope").error?.issues[0]?.message).toBe("second");
  });
});

describe("scorePassword", () => {
  it("scores nothing for an empty password", () => {
    expect(scorePassword("")).toBe(0);
  });

  it("gives a point for each rule met", () => {
    expect(scorePassword("abcdefgh")).toBe(1);
    expect(scorePassword("Abcdefgh")).toBe(2);
    expect(scorePassword("Abcdefg1")).toBe(3);
    expect(scorePassword("Abcdef1!")).toBe(4);
  });

  it("counts mixed case only when both cases appear", () => {
    expect(scorePassword("ABCDEFGH")).toBe(1);
    expect(scorePassword("abcdefgH")).toBe(2);
  });

  it("scores the rules independently of length", () => {
    expect(scorePassword("aB1!")).toBe(3);
  });

  it("lets length stand in for a symbol at 14 characters", () => {
    expect(scorePassword("abcdefghijklm")).toBe(1);
    expect(scorePassword("abcdefghijklmn")).toBe(2);
  });

  it("takes the length point at the caller's minimum, counting exactly", () => {
    expect(scorePassword("abcdefghijk", 12)).toBe(0);
    expect(scorePassword("abcdefghijkl", 12)).toBe(1);
    expect(scorePassword("abcdefg", 8)).toBe(0);
  });
});
