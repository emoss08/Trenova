import { StreamedText, type StreamedSegment } from "@/components/streamed-text";
import { useUserTimezone } from "@/hooks/use-user-timezone";
import { queries } from "@/lib/queries";
import { useShipmentCapabilities } from "@/lib/shipment-board/capabilities";
import { quickFilterLabel } from "@/lib/shipment-board/quick-filters";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { useShipmentBoardUrl } from "../url-state";

const BRIEFING_STALE_MS = 60_000;

/**
 * One sentence about the board, written by the assistant and streamed in.
 * The phrases that name a set of loads filter the table to them.
 */
export function ShipmentBriefing() {
  const t = useT();
  const { ai } = useShipmentCapabilities();
  const timezone = useUserTimezone();
  const [, setUrl] = useShipmentBoardUrl();
  const { data, isLoading, isError } = useQuery({
    ...queries.shipmentBoard.briefing(timezone),
    enabled: ai,
    staleTime: BRIEFING_STALE_MS,
  });

  const segments = useMemo<StreamedSegment[]>(
    () =>
      (data?.segments ?? []).map((segment) => {
        const filter = segment.filter;
        return filter
          ? {
              text: segment.text,
              label: t("Show {0}", quickFilterLabel({ filter }, t)),
              onActivate: () => void setUrl({ qf: [{ filter }], view: "table", expanded: null }),
            }
          : { text: segment.text };
      }),
    [data?.segments, setUrl, t],
  );

  if (!ai || isError) return null;
  if (isLoading || !data) {
    return <Skeleton className="h-5 w-2/3" />;
  }

  return (
    <StreamedText
      streamKey={String(data.generatedAt)}
      segments={segments}
      className="text-foreground text-lg leading-relaxed text-pretty"
    />
  );
}
