import { ageFromDays } from "@/lib/carrier-intelligence";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback } from "react";

export type IntelAgeFormatter = (days: number | null | undefined) => string | null;

export function useIntelAgeFormatter(): IntelAgeFormatter {
  const t = useT();

  return useCallback(
    (days: number | null | undefined) => {
      const age = ageFromDays(days);
      if (!age) {
        return null;
      }
      if (age.unit === "year") {
        return t("{0, plural, one {# yr} other {# yrs}}", age.value);
      }
      return age.value < 1 ? t("< 1 mo") : t("{0, plural, one {# mo} other {# mos}}", age.value);
    },
    [t],
  );
}
