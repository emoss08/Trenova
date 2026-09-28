import { SectionPanel } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  captureRequestFailureLabel,
  captureRequestStatusAttrs,
  captureRequestsToShow,
  isCaptureRecordKind,
  type CaptureRecordKind,
} from "@/lib/capture";
import {
  cancelCaptureRequest,
  type CaptureRequest,
  type CaptureRequestMode,
} from "@/lib/graphql/capture";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { ButtonGroup } from "@trenova/shared/components/ui/button-group";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { ChevronDownIcon, PrinterIcon, QrCodeIcon, ScanLineIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { Link } from "react-router";
import { CaptureRequestDialog } from "./capture-request-dialog";
import { CoverSheetDialog } from "./cover-sheet-dialog";

const nowInSeconds = () => Math.floor(Date.now() / 1000);
/** A request in flight changes by the minute; the panel re-reads the clock this often. */
const CLOCK_TICK_MS = 15_000;

/**
 * The kind of record capture can file onto, when this record is one and the
 * person may scan and print into it. Null otherwise, including while access
 * is still loading, so nothing capture-related shows until it is known.
 */
export function useCaptureKind(resourceType: string): CaptureRecordKind | null {
  const kind = isCaptureRecordKind(resourceType) ? resourceType : null;
  const accessQuery = useQuery({
    ...queries.capture.access(),
    staleTime: 5 * 60 * 1000,
    enabled: kind !== null,
  });
  const access = accessQuery.data;
  if (kind === null || access === undefined || !access.enabled || !access.canCapture) {
    return null;
  }
  return kind;
}

/**
 * "Scan" beside Upload in the Documents tab, with printing into the record and
 * cover sheets behind its menu. Scanning is the common case, so it is the
 * button; the others are one click further.
 */
export function CaptureButton({
  kind,
  recordId,
  disabled,
}: {
  kind: CaptureRecordKind;
  recordId: string;
  disabled: boolean;
}) {
  const t = useT();
  const [requestMode, setRequestMode] = useState<CaptureRequestMode | null>(null);
  const [coverSheetsOpen, setCoverSheetsOpen] = useState(false);

  return (
    <>
      <ButtonGroup>
        <Button
          type="button"
          size="sm"
          variant="secondary"
          disabled={disabled}
          onClick={() => setRequestMode("Scan")}
        >
          <ScanLineIcon className="size-4" />
          {t("Scan")}
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                type="button"
                size="icon-sm"
                variant="secondary"
                disabled={disabled}
                aria-label={t("More ways to add paper")}
              >
                <ChevronDownIcon className="size-4" />
              </Button>
            }
          />
          <DropdownMenuContent align="end" className="w-72">
            <DropdownMenuItem
              title={t("Scan into this record")}
              description={t("Start a scan on one of your computers")}
              descriptionClassProps="whitespace-normal"
              startContent={<ScanLineIcon className="size-3.5" />}
              onClick={() => setRequestMode("Scan")}
            />
            <DropdownMenuItem
              title={t("Print into this record")}
              description={t("File the next thing you print to Trenova here")}
              descriptionClassProps="whitespace-normal"
              startContent={<PrinterIcon className="size-3.5" />}
              onClick={() => setRequestMode("Print")}
            />
            <DropdownMenuItem
              title={t("Print cover sheets")}
              description={t("A sheet that routes a scanned stack here, from any scanner")}
              descriptionClassProps="whitespace-normal"
              startContent={<QrCodeIcon className="size-3.5" />}
              onClick={() => setCoverSheetsOpen(true)}
            />
          </DropdownMenuContent>
        </DropdownMenu>
      </ButtonGroup>

      {requestMode !== null && (
        <CaptureRequestDialog
          open
          onOpenChange={(open) => {
            if (!open) {
              setRequestMode(null);
            }
          }}
          mode={requestMode}
          kind={kind}
          recordId={recordId}
        />
      )}
      <CoverSheetDialog
        open={coverSheetsOpen}
        onOpenChange={setCoverSheetsOpen}
        kind={kind}
        recordId={recordId}
      />
    </>
  );
}

function minutesLeft(seconds: number): number {
  return Math.max(0, Math.ceil(seconds / 60));
}

function RequestRow({
  request,
  now,
  onCancel,
  canceling,
}: {
  request: CaptureRequest;
  now: number;
  onCancel: (id: string) => void;
  canceling: boolean;
}) {
  const t = useT();
  const attrs = captureRequestStatusAttrs(t)[request.status];
  const failure =
    request.failureCode === null
      ? null
      : request.failureMessage !== ""
        ? request.failureMessage
        : captureRequestFailureLabel(t, request.failureCode);
  const waitingForPrint = request.mode === "Print" && request.status === "Pending";

  return (
    <li className="flex flex-col gap-2 px-3 py-2">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
        {request.mode === "Scan" ? (
          <ScanLineIcon className="text-foreground-subtle size-4" aria-hidden />
        ) : (
          <PrinterIcon className="text-foreground-subtle size-4" aria-hidden />
        )}
        <span>{request.mode === "Scan" ? t("Scan") : t("Print")}</span>
        <Badge variant={phaseTone(attrs.phase)} title={attrs.description}>
          {attrs.text}
        </Badge>
        {waitingForPrint && (
          <span className="text-foreground-muted text-xs">
            {t(
              "{0, plural, one {Waiting for a print to Trenova, # minute left} other {Waiting for a print to Trenova, # minutes left}}",
              minutesLeft(request.expiresAt - now),
            )}
          </span>
        )}
        {request.batchId !== null && (
          <Link
            to={`/intake?batch=${encodeURIComponent(request.batchId)}&view=all`}
            className="ui-focus-ring text-brand text-xs hover:underline"
          >
            {t("Open in Intake")}
          </Link>
        )}
        <span className="text-foreground-subtle ml-auto text-xs tabular-nums">
          {formatSecondsAgo(Math.max(0, now - request.createdAt))}
        </span>
        {request.isOpen && (
          <Button
            type="button"
            size="xs"
            variant="ghost"
            onClick={() => onCancel(request.id)}
            isLoading={canceling}
          >
            {t("Cancel")}
          </Button>
        )}
      </div>
      {failure !== null && (
        <Alert variant="destructive" size="sm">
          <AlertDescription>{failure}</AlertDescription>
        </Alert>
      )}
    </li>
  );
}

/**
 * Scans and prints asked for on this record, while they matter: those still
 * in flight, and recent ones that finished or failed. Hidden when there are
 * none.
 */
export function CaptureRequestsPanel({
  kind,
  recordId,
}: {
  kind: CaptureRecordKind;
  recordId: string;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [now, setNow] = useState(nowInSeconds);
  const requestsQuery = useQuery(queries.capture.requests(kind, recordId));
  const shown = captureRequestsToShow(requestsQuery.data ?? [], now);
  const anyOpen = shown.some((request) => request.isOpen);

  useEffect(() => {
    if (!anyOpen) {
      return;
    }
    const timer = window.setInterval(() => setNow(nowInSeconds()), CLOCK_TICK_MS);
    return () => window.clearInterval(timer);
  }, [anyOpen]);

  const cancel = useApiMutation({
    mutationFn: (id: string) => cancelCaptureRequest(id),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: queries.capture.requests(kind, recordId).queryKey,
      });
    },
    resourceName: "Capture request",
  });

  if (requestsQuery.isError) {
    return (
      <Alert variant="destructive" size="sm">
        <AlertDescription className="flex flex-wrap items-center justify-between gap-2">
          <span>{t("Scans and prints for this record could not be loaded.")}</span>
          <Button
            type="button"
            size="xs"
            variant="outline"
            onClick={() => void requestsQuery.refetch()}
            isLoading={requestsQuery.isRefetching}
          >
            {t("Try again")}
          </Button>
        </AlertDescription>
      </Alert>
    );
  }

  if (shown.length === 0) {
    return null;
  }

  return (
    <SectionPanel
      title={t("Scans and prints")}
      icon={<ScanLineIcon aria-hidden />}
      count={shown.length}
    >
      <ul className="divide-border-subtle divide-y">
        {shown.map((request) => (
          <RequestRow
            key={request.id}
            request={request}
            now={now}
            onCancel={(id) => cancel.mutate(id)}
            canceling={cancel.isPending && cancel.variables === request.id}
          />
        ))}
      </ul>
    </SectionPanel>
  );
}
