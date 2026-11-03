import { hosClockTone } from "@/components/hos/hos-clock-gauges";
import { recordPath } from "@/config/record-links";
import type { DueCheck } from "@/routes/desk/_components/artifacts/record-card/due-checks";
import {
  dispatchStanding,
  hosClocks,
  licenseLine,
  qualificationChecks,
  type HosClock,
} from "@/routes/desk/_components/artifacts/cards/worker/worker-card";
import { RingGauge } from "@trenova/shared/components/ui/ring-gauge";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatClockDurationMs, formatCompactAge } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { PALETTE_ENTITY_ICONS } from "../../palette-entities";
import { workerRoleText } from "../../palette-fields";
import type { StatusState } from "../../palette-parts";
import { shipmentStatus, workerStatus } from "../../palette-status";
import { recordShipmentLane } from "./customer-preview";
import {
  PreviewError,
  PreviewFields,
  PreviewHead,
  PreviewSection,
  PreviewSkeleton,
  PreviewSource,
  RelatedRow,
  StatePill,
  type PreviewFieldRow,
} from "./preview-frame";
import { openShipmentsQuery, workerHosQuery, workerPreviewQuery } from "./preview-queries";
import type { RecordPreviewProps } from "./record-preview-props";

/** How big the clocks' rings are drawn in the preview's narrow column. */
const RING_SIZE = 56;
const RING_STROKE = 5;

const CHECK_STATE: Record<DueCheck["state"], StatusState | undefined> = {
  ok: undefined,
  info: undefined,
  warn: "warn",
  bad: "bad",
};

/**
 * The driver's four clocks as the Desk's worker card draws them: a ring each,
 * filled with what is left of its limit and coloured as the worker's page
 * colours it, the time left inside and the limit under it.
 */
export function HosRings({ clocks }: { clocks: readonly HosClock[] }) {
  const t = useT();
  return (
    <div className="bg-sunken rounded-surface grid grid-cols-4 gap-1 px-2 py-3">
      {clocks.map((clock) => (
        <div key={clock.key} className="flex min-w-0 flex-col items-center gap-1 text-center">
          <RingGauge
            value={clock.share}
            size={RING_SIZE}
            strokeWidth={RING_STROKE}
            tone={hosClockTone(clock.remainingMs)}
            aria-label={t("{0} time remaining", clock.label)}
          >
            <span
              className={cn(
                "text-xs font-medium tabular-nums",
                clock.state === "bad"
                  ? "text-danger-subtle-foreground"
                  : clock.state === "warn"
                    ? "text-warning-subtle-foreground"
                    : "text-foreground",
              )}
            >
              {formatClockDurationMs(clock.remainingMs)}
            </span>
          </RingGauge>
          <span className="text-foreground w-full truncate text-sm">{clock.label}</span>
          <span className="text-foreground-subtle w-full truncate text-xs tabular-nums">
            {t("of {0}", formatClockDurationMs(clock.limitMs))}
          </span>
        </div>
      ))}
    </div>
  );
}

/** The worker's qualification dates as rows, worst first: what lapsed or falls due marked. */
export function qualificationRows(checks: readonly DueCheck[]): PreviewFieldRow[] {
  return checks.map((check) => ({
    label: check.title,
    value: check.detail,
    state: CHECK_STATE[check.state],
  }));
}

export function WorkerPreview({ record, onNavigate }: RecordPreviewProps) {
  const t = useT();
  const now = useNow();
  const worker = useQuery(workerPreviewQuery(record.id));
  const hos = useQuery(workerHosQuery(record.id));
  const load = useQuery(openShipmentsQuery({ kind: "worker", id: record.id }, 1));

  if (worker.isLoading) {
    return <PreviewSkeleton />;
  }
  if (worker.isError || !worker.data) {
    return <PreviewError />;
  }

  const data = worker.data;
  const name = `${data.firstName} ${data.lastName}`.trim();
  const role = workerRoleText(data.type ?? undefined, data.driverType ?? undefined);
  const standing = dispatchStanding(data, t);
  // Where the worker cannot be dispatched, that is what the pill says; otherwise
  // it says what the list row says, driving, assigned or available.
  const status =
    standing.tone === "warn"
      ? { label: standing.label, state: "warn" as const }
      : workerStatus(record, t);
  const state = hos.data ?? null;
  const current = load.data?.rows[0];
  const home = [data.city, data.state?.abbreviation].filter(Boolean).join(", ");
  const profile = data.profile;
  const checks = profile ? qualificationChecks(profile, now, t) : [];

  return (
    <>
      <PreviewSource
        kind="worker"
        facts={[state ? t("ELD · {0} ago", formatCompactAge(now - state.recordedAt)) : null]}
      />
      <PreviewHead
        title={name || record.title}
        subtitle={[data.externalId, role].filter(Boolean).join(" · ")}
        status={status ? <StatePill label={status.label} state={status.state} /> : undefined}
      />
      {standing.reason && (
        <p className="text-foreground-muted text-base first-letter:uppercase">{standing.reason}</p>
      )}
      {state && <HosRings clocks={hosClocks(state, t)} />}
      {current && (
        <PreviewSection label={t("On load")}>
          <RelatedRow
            icon={PALETTE_ENTITY_ICONS.shipment}
            title={current.proNumber ?? current.id}
            mono
            sub={recordShipmentLane(current)}
            note={shipmentStatus(current.status, t)?.label}
            onOpen={() => onNavigate(recordPath("shipment", current.id))}
          />
        </PreviewSection>
      )}
      {checks.length > 0 && (
        <PreviewSection label={t("Qualifications")}>
          <PreviewFields rows={qualificationRows(checks)} />
        </PreviewSection>
      )}
      <PreviewFields
        rows={[
          profile ? { label: t("License"), value: licenseLine(profile, t) } : null,
          home ? { label: t("Home terminal"), value: home } : null,
        ]}
      />
    </>
  );
}
