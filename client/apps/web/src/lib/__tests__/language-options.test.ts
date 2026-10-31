import { filterLanguageOptions, languageOptions } from "@/lib/language-options";
import { describe, expect, it } from "vitest";

const names = (options: { nativeName: string; displayName: string }[]) =>
  options.map((option) => [option.nativeName, option.displayName]);

describe("languageOptions", () => {
  it("names every language in itself and in the interface's language", () => {
    expect(names(languageOptions("en"))).toEqual([
      ["English", "English"],
      ["Español", "Spanish"],
      ["繁體中文", "Chinese (Traditional)"],
      ["简体中文", "Chinese (Simplified)"],
    ]);
    expect(names(languageOptions("zh-CN"))).toEqual([
      ["English", "英语"],
      ["Español", "西班牙语"],
      ["繁體中文", "中文（繁体）"],
      ["简体中文", "中文（简体）"],
    ]);
    expect(names(languageOptions("es"))).toEqual([
      ["English", "inglés"],
      ["Español", "español"],
      ["繁體中文", "chino (tradicional)"],
      ["简体中文", "chino (simplificado)"],
    ]);
  });
});

describe("filterLanguageOptions", () => {
  const options = languageOptions("es");
  const found = (query: string) =>
    filterLanguageOptions(options, query).map((option) => option.locale);

  it("keeps every language for a blank query", () => {
    expect(found("   ")).toEqual(["en", "es", "zh-TW", "zh-CN"]);
  });

  it("ignores case and accents", () => {
    expect(found("ESPANOL")).toEqual(["es"]);
    expect(found("SIMPLIFICADO")).toEqual(["zh-CN"]);
  });

  it("finds a language by its English name whatever language is on screen", () => {
    expect(found("spanish")).toEqual(["es"]);
    expect(found("chinese")).toEqual(["zh-TW", "zh-CN"]);
    expect(found("traditional")).toEqual(["zh-TW"]);
  });

  it("finds a language by its native name and its locale code", () => {
    expect(found("简体")).toEqual(["zh-CN"]);
    expect(found("zh-tw")).toEqual(["zh-TW"]);
  });

  it("finds nothing for a language Trenova is not translated into", () => {
    expect(found("deutsch")).toEqual([]);
  });
});
