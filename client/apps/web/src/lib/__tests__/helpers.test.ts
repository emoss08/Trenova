import { describe, expect, it } from "vitest";
import {
  decimalStringSchema,
  nullableIntegerSchema,
  nullableStringSchema,
} from "@/types/helpers";

describe("decimalStringSchema", () => {
  it("parses string to float", () => {
    expect(decimalStringSchema.parse("123.45")).toBe(123.45);
  });

  it("converts empty string to null", () => {
    expect(decimalStringSchema.parse("")).toBeNull();
  });

  it("passes null through", () => {
    expect(decimalStringSchema.parse(null)).toBeNull();
  });

  it("passes undefined through", () => {
    expect(decimalStringSchema.parse(undefined)).toBeUndefined();
  });

  it("passes number through", () => {
    expect(decimalStringSchema.parse(42.5)).toBe(42.5);
  });

  it("parses zero string", () => {
    expect(decimalStringSchema.parse("0")).toBe(0);
  });

  it("parses negative string", () => {
    expect(decimalStringSchema.parse("-10.5")).toBe(-10.5);
  });
});

describe("nullableIntegerSchema", () => {
  it("parses string to integer", () => {
    expect(nullableIntegerSchema.parse("42")).toBe(42);
  });

  it("converts empty string to null", () => {
    expect(nullableIntegerSchema.parse("")).toBeNull();
  });

  it("passes null through", () => {
    expect(nullableIntegerSchema.parse(null)).toBeNull();
  });

  it("passes undefined through", () => {
    expect(nullableIntegerSchema.parse(undefined)).toBeUndefined();
  });

  it("passes integer number through", () => {
    expect(nullableIntegerSchema.parse(7)).toBe(7);
  });

  it("truncates decimal string via parseInt", () => {
    expect(nullableIntegerSchema.parse("3.7")).toBe(3);
  });
});

describe("nullableStringSchema", () => {
  it("passes string through", () => {
    expect(nullableStringSchema.parse("hello")).toBe("hello");
  });

  it("converts empty string to null", () => {
    expect(nullableStringSchema.parse("")).toBeNull();
  });

  it("passes null through", () => {
    expect(nullableStringSchema.parse(null)).toBeNull();
  });

  it("passes undefined through", () => {
    expect(nullableStringSchema.parse(undefined)).toBeUndefined();
  });

  it("preserves whitespace strings", () => {
    expect(nullableStringSchema.parse("  ")).toBe("  ");
  });
});
