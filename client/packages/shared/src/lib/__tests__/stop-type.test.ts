import { registerCatalogSource, setLocale } from "@trenova/shared/i18n/runtime";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { stopTypeLabel } from "../stop-type";

describe("stopTypeLabel", () => {
  beforeAll(async () => {
    await registerCatalogSource({
      es: async () => ({ "Split Pickup": "Recogida dividida" }),
    });
    await setLocale("es");
  });

  afterAll(async () => {
    await setLocale("en");
  });

  it("reads a stop type in the language on screen", () => {
    expect(stopTypeLabel("SplitPickup")).toBe("Recogida dividida");
  });

  it("keeps a type it does not know as sent", () => {
    expect(stopTypeLabel("Relay")).toBe("Relay");
  });
});
