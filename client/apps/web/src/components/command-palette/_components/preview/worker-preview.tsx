import { recordPath } from "@/config/record-links";
import { useNow } from "@trenova/shared/hooks/use-now";
import { durationLabel } from "@/routes/desk/_components/artifacts/cards/shipment/shipment-card";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatCompactAge, formatUnixDateMedium } from "@trenova/shared/lib/date";
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
} from "./preview-frame";
import { openShipmentsQuery, workerHosQuery, workerPreviewQuery } from "./preview-queries";
import type { RecordPreviewProps } from "./record-preview-props";

const MS_PER_MINUTE = 60_000;
const TIGHT_MS = 2 * 60 * MS_PER_MINUTE;
const SOON_SECONDS = 30 * 86_400;
const DEFAULT_LIMITS = { drive: 11, shift: 14, cycle: 70 } as const;

interface Clock {
  label: string;
  remainingMs: number;
  limitMs: number;
}

function HosStrip({ clocks }: { clocks: readonly Clock[] }) {
  const t = useT();
  return (
    <div className="bg-sunken rounded-surface divide-border-subtle grid grid-cols-3 divide-x">
      {clocks.map((clock) => {
        const share =
          clock.limitMs > 0 ? Math.min(1, Math.max(0, clock.remainingMs / clock.limitMs)) : 0;
        const tight = clock.remainingMs < TIGHT_MS;
        return (
          <div key={clock.label} className="flex min-w-0 flex-col gap-1.5 px-3.5 py-3">
            <span className="text-foreground-subtle truncate text-sm">{clock.label}</span>
            <span
              className={cn(
                "text-xl font-medium tabular-nums",
                tight ? "text-warning-subtle-foreground" : "text-foreground",
              )}
            >
              {durationLabel(clock.remainingMs / MS_PER_MINUTE, t)}
            </span>
            <span className="bg-border-subtle h-1 overflow-hidden rounded-full" aria-hidden>
              <span
                className={cn(
                  "cp-grow-x block h-full rounded-full",
                  tight ? "bg-warning" : "bg-foreground-muted",
                )}
                style={{ width: `${Math.round(share * 100)}%` }}
              />
            </span>
          </div>
        );
      })}
    </div>
  );
}

function expiry(
  at: number | null | undefined,
  t: ReturnType<typeof useT>,
): { value: string; state: StatusState } | null {
  if (!at || at <= 0) {
    return null;
  }
  const left = at - Date.now() / 1000;
  const date = formatUnixDateMedium(at);
  if (left < 0) {
    return { value: t("Expired {0}", date), state: "bad" };
  }
  if (left < SOON_SECONDS) {
    return { value: t("Expires {0}", date), state: "warn" };
  }
  return { value: t("Through {0}", date), state: "neutral" };
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
  const status = workerStatus(record, t);
  const state = hos.data ?? null;
  const clocks: Clock[] = state
    ? [
        {
          label: t("Drive"),
          remainingMs: state.driveRemainingMs,
          limitMs: state.driveLimitMs || DEFAULT_LIMITS.drive * 60 * MS_PER_MINUTE,
        },
        {
          label: t("Duty window"),
          remainingMs: state.shiftRemainingMs,
          limitMs: state.shiftLimitMs || DEFAULT_LIMITS.shift * 60 * MS_PER_MINUTE,
        },
        {
          label: t("Cycle"),
          remainingMs: state.cycleRemainingMs,
          limitMs: state.cycleLimitMs || DEFAULT_LIMITS.cycle * 60 * MS_PER_MINUTE,
        },
      ]
    : [];
  const current = load.data?.rows[0];
  const home = [data.city, data.state?.abbreviation].filter(Boolean).join(", ");
  const cdl = expiry(data.profile?.licenseExpiry, t);
  const medical = expiry(data.profile?.medicalCardExpiry, t);

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
      {clocks.length > 0 && <HosStrip clocks={clocks} />}
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
      <PreviewFields
        rows={[
          home ? { label: t("Home terminal"), value: home } : null,
          cdl
            ? {
                label: data.profile?.cdlClass
                  ? t("CDL class {0}", data.profile.cdlClass)
                  : t("CDL"),
                value: cdl.value,
                state: cdl.state,
              }
            : null,
          medical ? { label: t("Medical card"), value: medical.value, state: medical.state } : null,
        ]}
      />
    </>
  );
}
