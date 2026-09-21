import { useT } from "@trenova/shared/i18n/use-t";
import { IntelInlineError } from "@/components/carrier-intelligence/intel-inline-error";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import {
  CARRIER_INTEL_RAW_PAYLOAD_KEY,
  fetchCarrierIntelRawPayload,
  type CarrierIntelSnapshotSummary,
} from "@/lib/graphql/carrier-intelligence";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { ShikiCodeBlock } from "@trenova/shared/components/ui/shiki-code-block";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { downloadJsonFile } from "@trenova/shared/lib/utils";
import { CopyIcon, DownloadIcon, LockKeyholeIcon } from "lucide-react";
import { useMemo } from "react";

export type RawPayloadDialogProps = {
  carrierId: string;
  snapshot: Pick<CarrierIntelSnapshotSummary, "id" | "provider" | "fetchedAt" | "dotNumber"> | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function RawPayloadDialog({
  carrierId,
  snapshot,
  open,
  onOpenChange,
}: RawPayloadDialogProps) {
  const t = useT();
  const { copy } = useCopyToClipboard();

  const payloadQuery = useQuery({
    queryKey: [CARRIER_INTEL_RAW_PAYLOAD_KEY, carrierId, snapshot?.id],
    queryFn: ({ signal }) => fetchCarrierIntelRawPayload(carrierId, snapshot?.id ?? "", { signal }),
    enabled: open && snapshot !== null,
    staleTime: 5 * 60 * 1000,
    gcTime: 60 * 1000,
  });

  const pretty = useMemo(
    () =>
      payloadQuery.data === null || payloadQuery.data === undefined
        ? null
        : JSON.stringify(payloadQuery.data, null, 2),
    [payloadQuery.data],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="xl">
        <DialogHeader>
          <DialogTitle>{t("Raw provider payload")}</DialogTitle>
          <DialogDescription>
            {snapshot
              ? t(
                  "Exactly what {0} returned on {1}.",
                  carrierIntelProviderLabel(snapshot.provider),
                  formatUnixDateTimeMedium(snapshot.fetchedAt),
                )
              : null}
          </DialogDescription>
        </DialogHeader>
        <p className="text-muted-foreground flex items-start gap-2 text-xs">
          <LockKeyholeIcon className="mt-0.5 size-3.5 shrink-0" aria-hidden />
          {t(
            "Confidential. Licensed provider data that may contain personal contact details and must not be shared outside your organization. Viewing it is recorded.",
          )}
        </p>
        {payloadQuery.isPending ? (
          <Skeleton className="h-80 w-full" />
        ) : payloadQuery.isError ? (
          <IntelInlineError
            error={payloadQuery.error}
            title={t("The payload could not be loaded")}
            onRetry={() => void payloadQuery.refetch()}
          />
        ) : pretty === null ? (
          <p className="text-muted-foreground py-6 text-center text-xs">
            {t(
              "No raw payload is retained for this snapshot. Payloads are removed once the retention period passes.",
            )}
          </p>
        ) : (
          <ScrollArea className="h-[60vh] rounded-md border">
            <ShikiCodeBlock code={pretty} lang="json" className="text-xs" />
          </ScrollArea>
        )}
        <DialogFooter>
          {pretty !== null && snapshot ? (
            <>
              <Button
                type="button"
                variant="outline"
                onClick={() => void copy(pretty, { withToast: true })}
              >
                <CopyIcon />
                {t("Copy")}
              </Button>
              <Button
                type="button"
                variant="outline"
                onClick={() =>
                  downloadJsonFile(
                    `carrier-intel-${snapshot.dotNumber}-${snapshot.id}.json`,
                    payloadQuery.data,
                  )
                }
              >
                <DownloadIcon />
                {t("Download")}
              </Button>
            </>
          ) : null}
          <Button type="button" onClick={() => onOpenChange(false)}>
            {t("Close")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
