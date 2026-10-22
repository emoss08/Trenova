import { translateMessage } from "@trenova/shared/i18n/message";
import { requireCatalog, setLocale, type TranslateFn } from "@trenova/shared/i18n/runtime";
import { afterEach, beforeAll, describe, expect, it } from "vitest";

// A translator that shows what it was handed, so the order and the arguments of every lookup
// are visible in the result.
const calls: { message: string; args: unknown[] }[] = [];
const recording: TranslateFn = (message, ...args) => {
  calls.push({ message: message ?? "", args });
  return `<${message ?? ""}|${args.map(String).join(",")}>`;
};

beforeAll(() => {
  requireCatalog("core", "web");
});

afterEach(async () => {
  calls.length = 0;
  await setLocale("en");
});

describe("translateMessage", () => {
  it("looks a message up by its template and fills its arguments", () => {
    expect(translateMessage({ message: "unknown name {0}", args: ["`foo`"] }, recording)).toBe(
      "<unknown name {0}|`foo`>",
    );
  });

  it("takes a message with no arguments", () => {
    expect(translateMessage({ message: "Save" }, recording)).toBe("<Save|>");
    expect(calls).toEqual([{ message: "Save", args: [] }]);
  });

  it("passes a count through as a number, for a plural to select on", () => {
    translateMessage(
      { message: "{0, plural, one {# item} other {# items}}", args: [3] },
      recording,
    );
    expect(calls[0]?.args).toEqual([3]);
  });

  it("translates a nested message before the message it sits in, at any depth", () => {
    const text = translateMessage(
      {
        message: "{0} or {1}",
        args: [{ message: "a {0}", args: [{ message: "duration" }] }, { message: "a number" }],
      },
      recording,
    );
    expect(text).toBe("<{0} or {1}|<a {0}|<duration|>>,<a number|>>");
    expect(calls.map((call) => call.message)).toEqual([
      "duration",
      "a {0}",
      "a number",
      "{0} or {1}",
    ]);
  });

  it("renders every part of the sentence in the active locale", async () => {
    const message = { message: "Use {0}", args: [{ message: "Save" }] };
    expect(translateMessage(message)).toBe("Use Save");
    await setLocale("es");
    expect(translateMessage(message)).toBe("Utilizar Guardar");
  });
});
