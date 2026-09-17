import {
  formatIntelValue,
  humanizeIntelFieldPath,
  INTEL_EMPTY_VALUE,
  type IntelValueLabels,
} from "@/lib/carrier-intelligence";
import type { CarrierIntelEvent } from "@/lib/graphql/carrier-intelligence";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback, useMemo } from "react";

export type EventChange = {
  label: string;
  before: string;
  after: string;
};

export type EventPresentation = {
  title: string;
  change: EventChange | null;
};

type PresentableEvent = Pick<
  CarrierIntelEvent,
  "fieldPath" | "fieldLabel" | "ruleCode" | "ruleLabel" | "priorValue" | "currentValue" | "summary"
>;

const PROVIDER_RECORD_PATH = "identity.found";

export function useEventPresenter(): (event: PresentableEvent) => EventPresentation {
  const t = useT();
  const valueLabels = useMemo<IntelValueLabels>(
    () => ({ yes: t("Yes"), no: t("No"), empty: INTEL_EMPTY_VALUE }),
    [t],
  );

  return useCallback(
    (event: PresentableEvent) => {
      if (event.ruleCode) {
        return { title: event.ruleLabel || event.summary, change: null };
      }

      if (event.fieldPath === PROVIDER_RECORD_PATH) {
        return {
          title:
            event.currentValue === "false"
              ? t("Provider no longer has a record for this USDOT number")
              : t("Provider record is available again"),
          change: null,
        };
      }

      if (!event.fieldPath) {
        return { title: event.summary, change: null };
      }

      const label = event.fieldLabel || humanizeIntelFieldPath(event.fieldPath);
      if (event.priorValue === null && event.currentValue === null) {
        return { title: label, change: null };
      }

      const before = formatIntelValue(event.fieldPath, event.priorValue, valueLabels);
      const after = formatIntelValue(event.fieldPath, event.currentValue, valueLabels);
      return {
        title: t("{0}: {1} → {2}", label, before, after),
        change: { label, before, after },
      };
    },
    [t, valueLabels],
  );
}
