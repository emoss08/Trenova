import { Tally } from "@/routes/auth/_components/auth-primitives";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQuery } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { laneStatusLabel, MIN_LANE_CHIPS, NETWORK_PULSE_REFETCH_MS } from "../lib/network-pulse";
import { networkPulseService } from "../services/network-pulse";
import type { NetworkPulse as NetworkPulseData, NetworkPulseLane } from "../types/network-pulse";

/**
 * The hosted instance's real figures on the sign-in panel: loads in motion, the on-time
 * rate and the lanes it is running, aggregated across every organization. The endpoint
 * is opt-in (system.networkPulse.enabled) and 404s when it is off, so a failure renders
 * nothing rather than an error on a screen the user cannot act on.
 */
export function NetworkPulse() {
  const pulseQuery = useQuery({
    queryKey: ["network-pulse"],
    queryFn: networkPulseService.get,
    retry: false,
    staleTime: NETWORK_PULSE_REFETCH_MS,
    refetchInterval: NETWORK_PULSE_REFETCH_MS,
  });
  const pulse = pulseQuery.data;

  if (!pulse) {
    return null;
  }

  return (
    <>
      <NetworkPulseMetrics pulse={pulse} />
      <NetworkPulseLanes lanes={pulse.lanes} />
    </>
  );
}

/**
 * The on-time figure is dropped on its own when the window scored no deliveries,
 * because a percentage over an empty sample is not 0%, it is nothing.
 */
function NetworkPulseMetrics({ pulse }: { pulse: NetworkPulseData }) {
  const t = useT();

  const hasOnTime = pulse.sampleSize > 0;
  const windowLabel = pulse.windowDays === 7 ? "this week" : `last ${pulse.windowDays} days`;

  return (
    <div className="relative hidden gap-10 whitespace-nowrap [@media(min-height:620px)]:flex">
      <Metric label={t("loads in motion")}>
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
      <b className="block text-3xl leading-[1.1] font-medium tracking-[-0.035em] tabular-nums">
        {children}
      </b>
      <span className="text-subtle-foreground mt-0.5 block text-xs">{label}</span>
    </div>
  );
}

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
          className="border-border-2 text-subtle-foreground font-table inline-flex items-center gap-2 rounded-full border bg-[color-mix(in_oklch,var(--card)_40%,transparent)] px-2.5 py-[5px] text-2xs whitespace-nowrap"
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
