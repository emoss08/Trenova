import { useT } from "@trenova/shared/i18n/use-t";
import type {
  CarrierIntelRiskLevel,
  CarrierSourcingSort,
} from "@trenova/graphql/generated/graphql";
import { useIntelAgeFormatter } from "@/components/carrier-intelligence/use-intel-age";
import {
  type AuthorityAgePreset,
  type PowerUnitRange,
  type SourcingScreen,
} from "@/lib/carrier-sourcing";
import { useMemo } from "react";

export type SourcingLabels = {
  risk: Record<CarrierIntelRiskLevel, string>;
  sort: Record<CarrierSourcingSort, string>;
  screen: Record<SourcingScreen, string>;
  powerUnits: Record<PowerUnitRange, string>;
  authorityAge: Record<AuthorityAgePreset, string>;
  riskLabel: (level: string | null | undefined) => string;
  age: (days: number | null | undefined) => string | null;
};

export function useSourcingLabels(): SourcingLabels {
  const t = useT();

  const age = useIntelAgeFormatter();

  return useMemo(() => {
    const risk: Record<CarrierIntelRiskLevel, string> = {
      Low: t("Low"),
      Moderate: t("Moderate"),
      Elevated: t("Elevated"),
      High: t("High"),
      VeryHigh: t("Very high"),
      Unknown: t("Unknown"),
    };
    return {
      risk,
      sort: {
        BestMatch: t("Best match"),
        FleetSizeDesc: t("Fleet size"),
        AuthorityAgeDesc: t("Authority age"),
      },
      screen: {
        hazmat: t("Hazmat carriers only"),
        hideBlocked: t("Hide blocked carriers"),
        hideExisting: t("Hide carriers in Trenova"),
      },
      powerUnits: {
        "1-10": t("1–10 trucks"),
        "11-50": t("11–50 trucks"),
        "51-250": t("51–250 trucks"),
        "251+": t("251+ trucks"),
      },
      authorityAge: {
        under1: t("Under 1 yr"),
        "1-3": t("1–3 yrs"),
        "3-5": t("3–5 yrs"),
        "5+": t("5+ yrs"),
      },
      riskLabel: (level) =>
        level && level in risk ? risk[level as CarrierIntelRiskLevel] : risk.Unknown,
      age,
    };
  }, [age, t]);
}
