import type {
  StopVisitKind,
  TrackingEventOutcome,
  TrackingEventSource,
} from "@trenova/graphql/generated/graphql";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";

export type TrackingEventLabels = {
  source: (source: TrackingEventSource) => string;
  outcome: (outcome: TrackingEventOutcome) => string;
  kind: (kind: StopVisitKind) => string;
};

export function useTrackingEventLabels(): TrackingEventLabels {
  const t = useT();

  return useMemo(() => {
    const sources: Record<TrackingEventSource, string> = {
      Dispatcher: t("Dispatcher"),
      Agent: t("Agent"),
      Driver: t("Driver"),
      Telematics: t("Telematics"),
      EDI: t("EDI"),
    };
    const outcomes: Record<TrackingEventOutcome, string> = {
      Applied: t("Used"),
      Pending: t("Waiting"),
      Duplicate: t("Already reported"),
      Superseded: t("Not used"),
      Refused: t("Rejected"),
    };
    const kinds: Record<StopVisitKind, string> = {
      Arrival: t("Arrival"),
      Departure: t("Departure"),
    };

    return {
      source: (source) => sources[source],
      outcome: (outcome) => outcomes[outcome],
      kind: (kind) => kinds[kind],
    };
  }, [t]);
}
