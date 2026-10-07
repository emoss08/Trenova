import { StreamedText, type StreamedSegment } from "@/components/streamed-text";
import { useUserTimezone } from "@/hooks/use-user-timezone";
import { firstNameOf } from "@/lib/onboarding-copy";
import { queries } from "@/lib/queries";
import { useShipmentCapabilities } from "@/lib/shipment-board/capabilities";
import { quickFilterLabel } from "@/lib/shipment-board/quick-filters";
import type { ShipmentQuickFilter } from "@trenova/graphql/generated/graphql";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { useShipmentBoardUrl } from "../url-state";
import { BriefingLoadsPreview } from "./briefing-loads-preview";

const BRIEFING_STALE_MS = 60_000;

/**
 * The day's brief about the board, written for the organization before the
 * workday and again once everything it flagged is cleared, then streamed in
 * behind a greeting. The phrases that name a set of loads filter the table to
 * them, and hovering one lists the first few.
 */
export function ShipmentBriefing() {
  const t = useT();
  const { ai } = useShipmentCapabilities();
  const timezone = useUserTimezone();
  const userName = useAuthStore((state) => state.user?.name);
  const [, setUrl] = useShipmentBoardUrl();
  const { data, isLoading, isError } = useQuery({
    ...queries.shipmentBoard.briefing(timezone),
    enabled: ai,
    staleTime: BRIEFING_STALE_MS,
  });

  const showOnBoard = useCallback(
    (filter: ShipmentQuickFilter) =>
      void setUrl({ qf: [{ filter }], view: "table", expanded: null }),
    [setUrl],
  );

  const segments = useMemo<StreamedSegment[]>(
    () =>
      (data?.segments ?? []).map((segment) => {
        const filter = segment.filter;
        return filter
          ? {
              text: segment.text,
              label: t("Show {0}", quickFilterLabel({ filter }, t)),
              onActivate: () => showOnBoard(filter),
              preview: (
                <BriefingLoadsPreview filter={filter} onShowAll={() => showOnBoard(filter)} />
              ),
            }
          : { text: segment.text };
      }),
    [data?.segments, showOnBoard, t],
  );

  if (!ai || isError) return null;
  if (isLoading || !data) {
    return <Skeleton className="h-5 w-2/3" />;
  }

  const firstName = firstNameOf(userName);
  const asOf = formatToUserTimezone(data.generatedAt, {
    showDate: false,
    showTimeZone: false,
    showSeconds: false,
  });

  return (
    <div className="flex min-w-0 flex-col gap-1">
      <StreamedText
        streamKey={`${data.generatedAt}:${data.generation}`}
        prefix={firstName ? `${t("Hello {0}.", firstName)} ` : `${t("Hello.")} `}
        segments={segments}
        className="text-foreground text-lg leading-relaxed text-pretty"
      />
      <span className="text-muted-foreground text-xs">{t("Brief as of {0}", asOf)}</span>
    </div>
  );
}
