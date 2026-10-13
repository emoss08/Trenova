import type { CheckCallKind, CheckCallSource } from "@trenova/graphql/generated/graphql";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";

export type CheckCallLabels = {
  source: (source: CheckCallSource) => string;
  kind: (kind: CheckCallKind) => string;
};

export function useCheckCallLabels(): CheckCallLabels {
  const t = useT();

  return useMemo(() => {
    const sources: Record<CheckCallSource, string> = {
      Dispatcher: t("Dispatcher"),
      Carrier: t("Carrier link"),
      Agent: t("Agent"),
    };
    const kinds: Record<CheckCallKind, string> = {
      Update: t("Position update"),
      Arrival: t("Arrived"),
      Departure: t("Departed"),
    };

    return {
      source: (source) => sources[source],
      kind: (kind) => kinds[kind],
    };
  }, [t]);
}
