import { registerCatalogSource, setLocale } from "@trenova/shared/i18n/runtime";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import {
  JURISDICTION_VERIFICATION_LABELS,
  describeJurisdictionVerification,
} from "../jurisdiction-verification";

describe("describeJurisdictionVerification", () => {
  it("names the state alone when it was never settled on a day", () => {
    expect(describeJurisdictionVerification("Unverified", null)).toBe("Unverified");
  });

  it("reads the state and its day as one sentence", () => {
    expect(describeJurisdictionVerification("Verified", "Mar 4, 2026")).toBe(
      "Verified on Mar 4, 2026",
    );
    expect(describeJurisdictionVerification("Disputed", "Mar 4, 2026")).toBe(
      "Disputed on Mar 4, 2026",
    );
  });

  describe("in Spanish", () => {
    beforeAll(async () => {
      await registerCatalogSource({
        es: async () => ({
          Verified: "Verificada",
          "Verified on {0}": "Verificada el {0}",
        }),
      });
      await setLocale("es");
    });

    afterAll(async () => {
      await setLocale("en");
    });

    it("translates the state rather than showing the stored value", () => {
      expect(JURISDICTION_VERIFICATION_LABELS.Verified).toBe("Verificada");
      expect(describeJurisdictionVerification("Verified", "4 mar 2026")).toBe(
        "Verificada el 4 mar 2026",
      );
    });
  });
});
