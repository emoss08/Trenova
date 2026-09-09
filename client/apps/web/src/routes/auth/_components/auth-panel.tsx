import logoRainbow from "@/assets/logo.webp";
import { updateService } from "@/services/update";
import type { NetworkPulse, NetworkPulseLane } from "@/types/update";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useReducedMotion } from "motion/react";
import type { ReactNode } from "react";
import { AUTH_PITCH, laneStatusLabel } from "./auth-ambient";
import { Tally } from "./auth-primitives";

export type ReceiptRow = {
  key: string;
  value?: string;
};

export type CredentialReceipt = {
  rows: ReceiptRow[];
  issued: boolean;
};

/**
 * The ambient half of the sign-in screen. It is decoration with one exception: the
 * credential receipt, which fills in row by row as the real flow resolves identity,
 * workspace, roles and session — so the user can see what the session is being scoped
 * to before it is issued.
 *
 * Hidden below 900px, where the card takes the whole viewport.
 */
export function AuthPanel({ receipt }: { receipt: CredentialReceipt }) {
  const prefersReducedMotion = useReducedMotion();
  // One fetch feeds both the metrics and the lane band. The endpoint is opt-in
  // (system.networkPulse.enabled) and 404s when it is off, so a failure means "render
  // nothing" rather than an error state on a screen the user cannot act on.
  const pulseQuery = useQuery({
    queryKey: ["network-pulse"],
    queryFn: updateService.getNetworkPulse,
    retry: false,
    staleTime: PULSE_REFETCH_MS,
    refetchInterval: PULSE_REFETCH_MS,
  });
  const pulse = pulseQuery.data;

  return (
    <aside className="border-border-2 bg-panel relative hidden min-w-0 flex-col justify-between overflow-hidden border-r p-10 min-[900px]:flex">
      <div className="auth-weave" />
      <div className="auth-aura" />
      {!prefersReducedMotion && <div className="auth-scan" />}

      <div className="relative flex items-center gap-2.5">
        <img src={logoRainbow} alt="" className="size-6 object-contain" />
        <span className="text-[14px] font-semibold tracking-[-0.02em]">Trenova</span>
        <span className="border-border text-subtle-foreground font-table ml-0.5 border-l pl-2.5 text-[10.5px]">
          Enterprise
        </span>
      </div>

      <div className="relative flex flex-col gap-6">
        <h2 className="m-0 max-w-[19ch] text-[31px] leading-[1.14] font-medium tracking-[-0.038em] text-balance">
          {AUTH_PITCH}
        </h2>
        <NetworkPulseMetrics pulse={pulse} />
        <NetworkPulseLanes lanes={pulse?.lanes ?? []} />
        <CredentialReceiptCard receipt={receipt} />
      </div>

      <PanelFooter />
    </aside>
  );
}

const PULSE_REFETCH_MS = 60_000;

/**
 * The instance's real figures, across every organization and business unit. The on-time
 * figure is dropped on its own when the window scored no deliveries, because a
 * percentage over an empty sample is not 0%, it is nothing.
 */
function NetworkPulseMetrics({ pulse }: { pulse?: NetworkPulse }) {
  if (!pulse) {
    return null;
  }

  const hasOnTime = pulse.sampleSize > 0;
  const windowLabel = pulse.windowDays === 7 ? "this week" : `last ${pulse.windowDays} days`;

  return (
    <div className="relative hidden gap-10 whitespace-nowrap [@media(min-height:620px)]:flex">
      <Metric label="loads in motion">
        <Tally value={pulse.loadsInMotion} />
      </Metric>
      {hasOnTime && (
        <Metric label={`on-time ${windowLabel}`}>{pulse.onTimePercent.toFixed(1)}%</Metric>
      )}
    </div>
  );
}

function Metric({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <b className="block text-[24px] leading-[1.1] font-medium tracking-[-0.035em] tabular-nums">
        {children}
      </b>
      <span className="text-subtle-foreground mt-0.5 block text-[11.5px]">{label}</span>
    </div>
  );
}

// The band drifts continuously, so it needs enough chips to cover the track before the
// -50% translate wraps. A real instance running three lanes would otherwise show three
// chips and a long gap.
const MIN_LANE_CHIPS = 8;

/**
 * The lanes the instance is actually running, grouped to state pairs and load counts —
 * no customer, city or shipment identity. Absent entirely when there are none, rather
 * than falling back to invented routes.
 */
function NetworkPulseLanes({ lanes }: { lanes: NetworkPulseLane[] }) {
  if (lanes.length === 0) {
    return null;
  }

  return (
    <div className="auth-lanes relative hidden flex-col gap-2 [@media(min-height:620px)]:flex">
      <LaneTrack lanes={lanes} offset={0} reverse={false} />
      {lanes.length > 1 && <LaneTrack lanes={lanes} offset={Math.ceil(lanes.length / 2)} reverse />}
    </div>
  );
}

function LaneTrack({
  lanes,
  offset,
  reverse,
}: {
  lanes: NetworkPulseLane[];
  offset: number;
  reverse: boolean;
}) {
  const rotation = offset % lanes.length;
  const rotated = [...lanes.slice(rotation), ...lanes.slice(0, rotation)];

  const filled: NetworkPulseLane[] = [];
  while (filled.length < MIN_LANE_CHIPS) {
    filled.push(...rotated);
  }
  // Duplicated once more so the -50% translate loops seamlessly.
  const chips = [...filled, ...filled];

  return (
    <div className="auth-lane-track flex gap-2" data-reverse={reverse} aria-hidden="true">
      {chips.map((lane, index) => (
        <span
          key={`${lane.from}-${lane.to}-${lane.status}-${index}`}
          className="border-border-2 text-subtle-foreground font-table inline-flex items-center gap-2 rounded-full border bg-[color-mix(in_oklch,var(--card)_40%,transparent)] px-2.5 py-[5px] text-[10.5px] whitespace-nowrap"
        >
          <span className="text-muted-foreground">{lane.from}</span>→
          <span className="text-muted-foreground">{lane.to}</span>
          <i className="bg-muted-foreground size-1 shrink-0 rounded-full" />
          {lane.count} {laneStatusLabel(lane.status)}
        </span>
      ))}
    </div>
  );
}

function CredentialReceiptCard({ receipt }: { receipt: CredentialReceipt }) {
  return (
    <div className="auth-receipt border-border-2 relative w-full max-w-[392px] rounded-xl border">
      <div className="border-border-2 flex items-center justify-between border-b border-dashed px-3.5 py-[11px]">
        <span className="text-subtle-foreground font-table text-[10.5px] whitespace-nowrap">
          Credential
        </span>
        <span className="text-subtle-foreground font-table text-[10.5px] whitespace-nowrap">
          {receipt.issued ? "Issued" : "Assembling"}
        </span>
      </div>
      {receipt.rows.map((row, index) => (
        <div
          key={row.key}
          className={cn(
            "border-border-2 grid grid-cols-[78px_1fr] items-baseline gap-3 border-b border-dashed px-3.5 py-2.5 last:border-b-0",
            // The stamp is pinned to the bottom-right corner and overlays the last row.
            receipt.issued && index === receipt.rows.length - 1 && "pr-[104px]",
          )}
        >
          <span className="text-subtle-foreground font-table text-[10.5px]">{row.key}</span>
          {row.value ? (
            <span
              key={row.value}
              className="auth-receipt-fill min-w-0 truncate text-[12.5px]"
              title={row.value}
            >
              {row.value}
            </span>
          ) : (
            <span className="auth-receipt-pending" aria-hidden="true" />
          )}
        </div>
      ))}
      {receipt.issued && (
        <span className="auth-stamp border-foreground font-table absolute right-3.5 bottom-3 rounded border px-[7px] py-[3px] text-[10.5px] tracking-[0.04em] uppercase">
          Authorized
        </span>
      )}
    </div>
  );
}

function PanelFooter() {
  // Public endpoint — it is the same call the update banner uses, and its success is
  // also the honest answer to whether the API is reachable from this browser.
  const versionQuery = useQuery({
    queryKey: ["system-version"],
    queryFn: updateService.getVersion,
    retry: false,
    staleTime: Number.POSITIVE_INFINITY,
  });
  const reachable = versionQuery.isSuccess;

  return (
    <div className="text-subtle-foreground font-table relative flex flex-wrap items-center gap-4 text-[10.5px] whitespace-nowrap">
      <span className="inline-flex items-center gap-1.5">
        <i
          className={cn(
            "size-[5px] rounded-full",
            reachable ? "bg-foreground auth-pulse" : "bg-auth-danger",
          )}
        />
        {versionQuery.isPending
          ? "Contacting network"
          : reachable
            ? "Network operational"
            : "Network unreachable"}
      </span>
      {versionQuery.data?.environment && <span>{versionQuery.data.environment}</span>}
      {versionQuery.data?.version && <span>v{versionQuery.data.version}</span>}
    </div>
  );
}
