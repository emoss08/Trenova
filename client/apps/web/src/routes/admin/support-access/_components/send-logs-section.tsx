import { SectionPanel } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  requestSupportLogs,
  type SupportLogPeriod,
  type SupportLogRequest,
  type SupportLogs,
} from "@/lib/graphql/support-logs";
import { supportLogs } from "@/lib/queries/support-logs";
import { withoutRecordIds } from "@/lib/record-ids";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone, formatUnixMonthDay } from "@trenova/shared/lib/date";
import { useId, useRef, useState } from "react";
import { SendLogsSkeleton } from "./send-logs-skeleton";

const PERIODS: readonly SupportLogPeriod[] = ["Hour", "Day", "Week"];

function periodLabel(t: TranslateFn, period: SupportLogPeriod): string {
  switch (period) {
    case "Hour":
      return t("Last hour");
    case "Week":
      return t("Last 7 days");
    default:
      return t("Last 24 hours");
  }
}

function StatusBadge({ request }: { request: SupportLogRequest }) {
  const t = useT();

  switch (request.status) {
    case "Collecting":
      return (
        <Badge variant="info" dot>
          {t("Collecting")}
        </Badge>
      );
    case "Sent":
      return (
        <Badge variant="success" dot>
          {request.finishedAt != null
            ? t("Sent {0}", formatUnixMonthDay(request.finishedAt))
            : t("Sent")}
        </Badge>
      );
    case "Failed":
      return (
        <Badge variant="danger" dot>
          {t("Failed")}
        </Badge>
      );
    default:
      return (
        <Badge variant="neutral" dot>
          {t("Requested")}
        </Badge>
      );
  }
}

function RequestRow({ request }: { request: SupportLogRequest }) {
  const t = useT();
  const askedAt = formatUnixInUserTimezone(request.requestedAt, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
  const reason =
    request.status === "Failed" && request.failureMessage
      ? withoutRecordIds(request.failureMessage)
      : null;

  return (
    <li className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-0.5 px-3 py-2.5 text-xs">
      <span className="text-sm font-medium text-foreground">{periodLabel(t, request.period)}</span>
      <span className="col-start-2 row-span-2 row-start-1 flex items-center">
        <StatusBadge request={request} />
      </span>
      <span className="col-start-1 truncate text-muted-foreground">
        {t("Asked by {0} on {1}", request.requestedByName, askedAt)}
      </span>
      {reason ? <p className="col-span-2 text-foreground">{reason}</p> : null}
    </li>
  );
}

function holdReason(t: TranslateFn, logs: SupportLogs): string | null {
  if (logs.inFlight) {
    return t("Logs are on their way to Trenova support. You can send more once they arrive.");
  }
  if (logs.remainingToday <= 0) {
    return t("This instance has sent logs 5 times in the last 24 hours. Try again later.");
  }
  return null;
}

function SendLogsBody({ logs }: { logs: SupportLogs }) {
  const t = useT();
  const queryClient = useQueryClient();
  const periodId = useId();
  const [period, setPeriod] = useState<SupportLogPeriod>("Day");
  const sending = useRef(false);

  const mutation = useApiMutation({
    mutationFn: (chosen: SupportLogPeriod) => requestSupportLogs(chosen),
    onSettled: async () => {
      await queryClient.invalidateQueries({ queryKey: supportLogs._def });
    },
    resourceName: "Log request",
  });

  const held = holdReason(t, logs);

  async function send() {
    if (sending.current || held) {
      return;
    }
    sending.current = true;
    try {
      await mutation.mutateAsync(period);
    } catch {
      return;
    } finally {
      sending.current = false;
    }
  }

  return (
    <>
      <div className="flex flex-col gap-3 px-3 py-3">
        <p className="max-w-[72ch] text-xs text-muted-foreground">
          {t(
            "Trenova support receives this instance's logs for the period you choose and keeps them for 30 days.",
          )}
        </p>
        <div className="flex flex-wrap items-center gap-2">
          <label htmlFor={periodId} className="sr-only">
            {t("Period")}
          </label>
          <Select
            value={period}
            onValueChange={(value) => value && setPeriod(value as SupportLogPeriod)}
            items={PERIODS.map((value) => ({ value, label: periodLabel(t, value) }))}
          >
            <SelectTrigger id={periodId} className="w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {PERIODS.map((value) => (
                <SelectItem key={value} value={value}>
                  {periodLabel(t, value)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            type="button"
            size="sm"
            disabled={held !== null}
            isLoading={mutation.isPending}
            loadingText={t("Sending…")}
            onClick={() => void send()}
          >
            {t("Send logs")}
          </Button>
        </div>
        {held ? <p className="text-xs text-muted-foreground">{held}</p> : null}
      </div>
      {logs.requests.length > 0 ? (
        <ul
          aria-label={t("Recent log requests")}
          className="flex flex-col divide-y divide-border-subtle border-t border-border-subtle"
        >
          {logs.requests.map((request) => (
            <RequestRow key={request.id} request={request} />
          ))}
        </ul>
      ) : null}
    </>
  );
}

/**
 * Sends the instance's logs for a chosen period to Trenova support. Shown only on a Managed
 * Trenova instance; the server holds every rule (one request at a time for the whole instance, a
 * daily limit), and the list moves through its states as the instance reports them.
 */
export function SendLogsSection() {
  const t = useT();
  const query = useQuery(supportLogs.recent());

  if (query.isPending) {
    return <SendLogsSkeleton />;
  }
  if (query.isError) {
    return (
      <SectionPanel title={t("Send logs to Trenova support")}>
        <ErrorState
          error={query.error}
          onRetry={() => void query.refetch()}
          layout="compact"
          title={t("Log requests could not be loaded")}
        />
      </SectionPanel>
    );
  }
  if (!query.data.managed) {
    return null;
  }
  return (
    <SectionPanel title={t("Send logs to Trenova support")}>
      <SendLogsBody logs={query.data} />
    </SectionPanel>
  );
}
