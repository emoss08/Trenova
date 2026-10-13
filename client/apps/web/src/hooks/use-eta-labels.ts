import { useT } from "@trenova/shared/i18n/use-t";
import type { BadgeTone } from "@trenova/shared/components/ui/badge";
import type { EtaVerdict, StopEtaBasis, StopEtaDriveMethod } from "@trenova/shared/types/shipment";
import { useMemo } from "react";

export const ETA_VERDICT_TONE: Record<EtaVerdict, BadgeTone> = {
  OnTime: "success",
  AtRisk: "warning",
  Late: "danger",
  Unknown: "neutral",
};

export type EtaLabels = {
  verdict: (verdict: EtaVerdict) => string;
  basis: (basis: StopEtaBasis) => string;
  driveMethod: (method: StopEtaDriveMethod) => string;
};

export function useEtaLabels(): EtaLabels {
  const t = useT();

  return useMemo(() => {
    const verdicts: Record<EtaVerdict, string> = {
      OnTime: t("On time"),
      AtRisk: t("At risk"),
      Late: t("Projected late"),
      Unknown: t("No projection"),
    };
    const bases: Record<StopEtaBasis, string> = {
      Position: t("Truck position"),
      CheckCall: t("Check call position"),
      CarrierReported: t("Carrier's estimate"),
      AtStop: t("Truck at the stop"),
      LastDeparture: t("Last departure"),
      None: t("Nothing to estimate from"),
    };
    const methods: Record<StopEtaDriveMethod, string> = {
      Routed: t("Routed"),
      Planning: t("Planning speed"),
      Reported: t("Reported by the carrier"),
      None: t("No drive"),
    };

    return {
      verdict: (verdict) => verdicts[verdict],
      basis: (basis) => bases[basis],
      driveMethod: (method) => methods[method],
    };
  }, [t]);
}
