import { translateMessage } from "@trenova/shared/i18n/message";
import { requireCatalog, setLocale, type TranslateFn } from "@trenova/shared/i18n/runtime";
import { afterEach, beforeAll, describe, expect, it } from "vitest";

beforeAll(() => {
  requireCatalog("core", "web");
});

afterEach(async () => {
  await setLocale("en");
});

const echo: TranslateFn = (message, ...args) => `<${message}|${args.join(",")}>`;

describe("translateMessage", () => {
  it("passes the template and its arguments to the translator", () => {
    expect(translateMessage({ message: "Lane {0}", args: [3] }, echo)).toBe("<Lane {0}|3>");
  });

  it("treats missing arguments as none", () => {
    expect(translateMessage({ message: "Any" }, echo)).toBe("<Any|>");
  });

  it("translates a nested message before substituting it, at any depth", () => {
    const message = {
      message: "outer {0} {1}",
      args: [{ message: "middle {0}", args: [{ message: "inner" }] }, "text"],
    };
    expect(translateMessage(message, echo)).toBe("<outer {0} {1}|<middle {0}|<inner|>>,text>");
  });

  it("renders every part of a matrix overlap warning in the reader's language", async () => {
    await setLocale("es");
    const overlap = {
      message: "{0} bands {1} and {2} overlap, so the same shipment matches both.",
      args: [
        "Peso",
        { message: "{0} – under {1}", args: ["0", "1,000"] },
        { message: "{0} and over", args: ["500"] },
      ],
    };
    expect(translateMessage(overlap)).toBe(
      "Las bandas 0 – menos de 1,000 y 500 en adelante de Peso se superponen, por lo que el mismo envío coincide con ambas.",
    );
  });
});
