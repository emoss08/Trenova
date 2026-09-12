// These cases are deliberately the same table as TestFormatPositional / TestFormatPlural in
// shared/i18n/i18n_test.go. The two formatters render the same catalog entries, so they have
// to agree; if one side is changed alone, one of these pairs starts failing.
import { formatMessage } from "@trenova/shared/i18n/format-message";
import { describe, expect, it } from "vitest";

describe("formatMessage — positional", () => {
  it("returns the message untouched when there are no arguments", () => {
    expect(formatMessage("en", "Save", [])).toBe("Save");
  });

  it("substitutes a single placeholder", () => {
    expect(formatMessage("en", 'Delete "{0}"?', ["Load 42"])).toBe('Delete "Load 42"?');
  });

  it("substitutes multiple placeholders", () => {
    expect(formatMessage("en", "{0} of {1}", [3, 9])).toBe("3 of 9");
  });

  it("repeats a placeholder used twice", () => {
    expect(formatMessage("en", "{0} and {0}", ["x"])).toBe("x and x");
  });

  it("leaves an out-of-range index intact rather than rendering undefined", () => {
    expect(formatMessage("en", "{3}", ["a"])).toBe("{3}");
  });

  it("treats an unmatched brace as literal text", () => {
    expect(formatMessage("en", "100% { done", ["a"])).toBe("100% { done");
  });

  it("leaves a non-numeric placeholder intact", () => {
    expect(formatMessage("en", "{name}", ["a"])).toBe("{name}");
  });

  it("renders a float without trailing zeros", () => {
    expect(formatMessage("en", "{0} mi", [12.5])).toBe("12.5 mi");
  });
});

describe("formatMessage — plurals", () => {
  const message = "{0, plural, one {# shipment} other {# shipments}}";

  it("selects the one-form in English", () => {
    expect(formatMessage("en", message, [1])).toBe("1 shipment");
  });

  it("selects the other-form for zero and many", () => {
    expect(formatMessage("en", message, [0])).toBe("0 shipments");
    expect(formatMessage("en", message, [7])).toBe("7 shipments");
  });

  it("applies Spanish plural rules", () => {
    const spanish = "{0, plural, one {# envío} other {# envíos}}";
    expect(formatMessage("es", spanish, [1])).toBe("1 envío");
    expect(formatMessage("es", spanish, [4])).toBe("4 envíos");
  });

  it("uses the single Chinese form even for a count of one", () => {
    const chinese = "{0, plural, other {# 個運單}}";
    expect(formatMessage("zh-TW", chinese, [1])).toBe("1 個運單");
    expect(formatMessage("zh-CN", "{0, plural, other {# 个运单}}", [1])).toBe("1 个运单");
  });

  it("falls back to the other-form when one is absent", () => {
    expect(formatMessage("en", "{0, plural, other {# items}}", [1])).toBe("1 items");
  });

  it("keeps surrounding text", () => {
    const sentence = "You have {0, plural, one {# message} other {# messages}} waiting";
    expect(formatMessage("en", sentence, [1])).toBe("You have 1 message waiting");
    expect(formatMessage("en", sentence, [3])).toBe("You have 3 messages waiting");
  });

  it("ignores the plural form when the argument is not a number", () => {
    expect(formatMessage("en", message, ["many"])).toBe("many");
  });
});
