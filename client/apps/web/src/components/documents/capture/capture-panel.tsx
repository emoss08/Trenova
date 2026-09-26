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
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
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
/** A request in flight changes by the second; the page re-reads the clock this often. */
const CLOCK_TICK_MS = 15_000;

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
  const what = request.mode === "Scan" ? t("Scan") : t("Print");
  const failure =
    request.failureCode !== null
      ? request.failureMessage !== ""
        ? request.failureMessage
        : captureRequestFailureLabel(t, request.failureCode)
      : null;

  return (
    <li className="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2 text-sm">
      {request.mode === "Scan" ? (
        <ScanLineIcon className="text-foreground-subtle size-4" aria-hidden />
      ) : (
        <PrinterIcon className="text-foreground-subtle size-4" aria-hidden />
      )}
      <span>{what}</span>
      <Badge variant={phaseTone(attrs.phase)} title={attrs.description}>
        {attrs.text}
      </Badge>
      {request.mode === "Print" && request.status === "Pending" && (
        <span className="text-foreground-muted text-xs">
          {t("Waiting for a print to Trenova")} {formatRemaining(t, request.expiresAt - now)}
        </span>
      )}
      {failure !== null && <span className="text-danger text-xs">{failure}</span>}
      {request.batchId !== null && (
        <Link
          to={`/intake?batch=${encodeURIComponent(request.batchId)}&view=all`}
          className="ui-focus-ring text-brand text-xs hover:underline"
        >
          {t("Open in Intake")}
        </Link>
      )}
      <span className="text-foreground-subtle ml-auto text-xs">
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
    </li>
  );
}

function formatRemaining(t: (value: string, ...args: unknown[]) => string, seconds: number) {
  const minutes = Math.max(0, Math.ceil(seconds / 60));
  return t("{0, plural, one {(# minute left)} other {(# minutes left)}}", minutes);
}

/**
 * Scanning and printing into a record, from its Documents tab. Shown only
 * where capture is on and the person may use it, and only on the kinds of
 * record a captured document can be filed onto.
 */
export function CapturePanel({
  resourceType,
  resourceId,
  disabled,
}: {
  resourceType: string;
  resourceId: string;
  disabled: boolean;
}) {
  const kind = isCaptureRecordKind(resourceType) ? resourceType : null;
  const accessQuery = useQuery({ ...queries.capture.access(), staleTime: 5 * 60 * 1000 });
  const access = accessQuery.data;

  if (kind === null || access === undefined || !access.enabled || !access.canCapture) {
    return null;
  }

  return <CaptureControls kind={kind} recordId={resourceId} disabled={disabled} />;
}

function CaptureControls({
  kind,
  recordId,
  disabled,
}: {
  kind: CaptureRecordKind;
  recordId: string;
  disabled: boolean;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [now, setNow] = useState(nowInSeconds);
  const [requestMode, setRequestMode] = useState<CaptureRequestMode | null>(null);
  const [coverSheetsOpen, setCoverSheetsOpen] = useState(false);

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

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        <div className="flex">
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="rounded-r-none"
            disabled={disabled}
            onClick={() => setRequestMode("Scan")}
          >
            <ScanLineIcon className="size-3.5" />
            {t("Scan")}
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  type="button"
                  size="icon-sm"
                  variant="outline"
                  className="-ml-px rounded-l-none"
                  disabled={disabled}
                  aria-label={t("More ways to capture")}
                >
                  <ChevronDownIcon className="size-3.5" />
                </Button>
              }
            />
            <DropdownMenuContent align="start" className="w-64">
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
        </div>
      </div>

      {shown.length > 0 && (
        <ul
          aria-label={t("Scans and prints for this record")}
          className="border-border divide-border-subtle bg-card divide-y rounded-lg border"
        >
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
      )}

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
    </div>
  );
}
