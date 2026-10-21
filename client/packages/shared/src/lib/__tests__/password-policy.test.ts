import { describe, expect, it } from "vitest";
import {
  PASSWORD_MAX_BYTES,
  PASSWORD_MIN_LENGTH,
  meetsPasswordLength,
  passwordByteLength,
  passwordLength,
} from "../password-policy";

describe("password policy", () => {
  it("matches the server's limits", () => {
    expect(PASSWORD_MIN_LENGTH).toBe(12);
    expect(PASSWORD_MAX_BYTES).toBe(72);
  });

  it("counts characters the way the server does", () => {
    expect(passwordLength("ñandú-ñandú-")).toBe(12);
    expect(passwordLength("🚚🚚🚚🚚🚚🚚🚚🚚🚚🚚🚚🚚")).toBe(12);
  });

  it("measures the bytes bcrypt reads", () => {
    expect(passwordByteLength("abc")).toBe(3);
    expect(passwordByteLength("ñ")).toBe(2);
    expect(passwordByteLength("🚚")).toBe(4);
  });

  it("accepts twelve characters up to seventy-two bytes", () => {
    expect(meetsPasswordLength("a".repeat(11))).toBe(false);
    expect(meetsPasswordLength("a".repeat(12))).toBe(true);
    expect(meetsPasswordLength("a".repeat(72))).toBe(true);
    expect(meetsPasswordLength("a".repeat(73))).toBe(false);
    expect(meetsPasswordLength("🚚".repeat(18))).toBe(true);
    expect(meetsPasswordLength("🚚".repeat(19))).toBe(false);
  });
});
