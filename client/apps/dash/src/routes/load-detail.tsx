import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { fetchMyLoadPayEstimate } from "@trenova/shared/lib/graphql/driver-portal";
import { useQuery } from "@tanstack/react-query";
import { m } from "motion/react";
import { ArrowLeftIcon, CheckIcon, CopyIcon, NavigationIcon } from "lucide-react";
import { useState } from "react";
import { Link, useParams } from "react-router";
import { AssignmentResponseCard } from "../_components/assignment-response-card";
import { useDashFeatures } from "../_components/use-dash-features";
import { LoadPayChip, StopTimeline } from "../_components/load-card";
import { LoadChat } from "../_components/load-chat";
import { LoadDocuments } from "../_components/load-documents";
import { LoadStatusBadge } from "../_components/portal-badges";
import {
  destinationStop,
  directionsUrl,
  formatMiles,
  formatPieces,
  formatWeight,
  originStop,
  stopPlace,
  useLoad,
} from "../_components/use-loads";

function CopyChip({ label, value }: { label: string; value: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <button
      type="button"
      onClick={() => void handleCopy()}
      className="flex items-center gap-1.5 rounded-full border border-border px-2.5 py-1 text-xs font-medium transition-colors hover:bg-accent"
    >
      {copied ? (
        <CheckIcon className="size-3 text-green-600" />
      ) : (
        <CopyIcon className="size-3 text-muted-foreground" />
      )}
      {label} <span className="font-mono">{value}</span>
    </button>
  );
}

function DetailStat({ label, value }: { label: string; value: string | null }) {
  if (!value) return null;
  return (
    <div className="rounded-xl border border-border bg-card px-3 py-2">
      <p className="text-2xs font-medium text-muted-foreground uppercase">{label}</p>
      <p className="mt-0.5 truncate text-sm font-semibold">{value}</p>
    </div>
  );
}

function PayEstimateCard({ shipmentId, moveId }: { shipmentId: string; moveId: string }) {
  const estimate = useQuery({
    queryKey: ["dash-pay-estimate", shipmentId, moveId],
    queryFn: async () => {
      try {
        return await fetchMyLoadPayEstimate(shipmentId, moveId);
      } catch {
        return null;
      }
    },
    staleTime: 5 * 60 * 1000,
  });

  if (estimate.isPending || !estimate.data || estimate.data.grossMinor <= 0) {
    return null;
  }

  return (
    <div className="rounded-2xl border border-border bg-card p-4">
      <p className="text-2xs font-medium text-muted-foreground uppercase">
        Estimated pay for this load
      </p>
      <p className="mt-1 text-2xl font-semibold tracking-tight">
        <AmountDisplay value={estimate.data.grossMinor} currency={estimate.data.currencyCode} />
      </p>
      <p className="mt-1 text-xs text-muted-foreground">
        Based on your current pay plan — final pay locks in when the load completes.
      </p>
    </div>
  );
}

export function DashLoadDetailPage() {
  const { assignmentId = "" } = useParams();
  const { load, isPending } = useLoad(assignmentId);
  const features = useDashFeatures();

  if (isPending) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-6 w-28" />
        <Skeleton className="h-40 w-full rounded-2xl" />
        <Skeleton className="h-64 w-full rounded-2xl" />
      </div>
    );
  }

  if (!load) {
    return (
      <div className="flex flex-col items-start gap-4">
        <Link to="/dash/loads" className="flex items-center gap-1 text-sm text-muted-foreground">
          <ArrowLeftIcon className="size-4" /> Loads
        </Link>
        <div className="w-full rounded-2xl border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
          We couldn&apos;t find that load. It may have been reassigned.
        </div>
      </div>
    );
  }

  const origin = originStop(load);
  const destination = destinationStop(load);
  const nextStop = load.stops.find((stop) => !stop.actualDeparture);
  const isActive = load.status === "Assigned" || load.status === "InTransit";

  return (
    <m.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.22, ease: "easeOut" }}
      className="flex flex-col gap-4"
    >
      <Link to="/dash/loads" className="flex items-center gap-1 text-sm text-muted-foreground">
        <ArrowLeftIcon className="size-4" /> Loads
      </Link>

      <div className="rounded-2xl border border-border bg-card p-4">
        <div className="flex items-center justify-between gap-2">
          <p className="font-mono text-sm font-semibold">{load.proNumber || "Pending pro #"}</p>
          <LoadStatusBadge status={load.status} />
        </div>
        <h1 className="mt-2 text-xl font-semibold tracking-tight">
          {stopPlace(origin)} <span className="text-muted-foreground">→</span>{" "}
          {stopPlace(destination)}
        </h1>
        <div className="mt-3 flex flex-wrap items-center gap-1.5">
          {load.proNumber ? <CopyChip label="PRO" value={load.proNumber} /> : null}
          {load.bol ? <CopyChip label="BOL" value={load.bol} /> : null}
          {isActive && nextStop && (nextStop.addressLine || nextStop.locationName) ? (
            <a
              href={directionsUrl(nextStop)}
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-1.5 rounded-full bg-primary px-2.5 py-1 text-xs font-semibold text-primary-foreground"
            >
              <NavigationIcon className="size-3" />
              Next stop
            </a>
          ) : null}
        </div>
      </div>

      <AssignmentResponseCard load={load} />

      {load.payGrossMinor != null ? (
        <div className="rounded-2xl border border-border bg-card p-4">
          <div className="flex items-center justify-between gap-2">
            <div>
              <p className="text-2xs font-medium text-muted-foreground uppercase">
                Your pay for this load
              </p>
              <p className="mt-1 text-2xl font-semibold tracking-tight">
                <AmountDisplay value={load.payGrossMinor} />
              </p>
            </div>
            <LoadPayChip load={load} />
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            {load.payOnHold
              ? "This pay is on hold — check with your fleet manager."
              : load.payStatus === "Settled"
                ? "Paid out on a settlement — see the Pay tab."
                : "Earned — lands on your next settlement."}
          </p>
        </div>
      ) : isActive && features.showPayEstimates ? (
        <PayEstimateCard shipmentId={load.shipmentId} moveId={load.moveId} />
      ) : null}

      <LoadDocuments shipmentId={load.shipmentId} />

      <LoadChat shipmentId={load.shipmentId} />

      <div className="grid grid-cols-2 gap-2">
        <DetailStat label="Distance" value={formatMiles(load.distanceMiles)} />
        <DetailStat label="Weight" value={formatWeight(load.weight)} />
        <DetailStat label="Pieces" value={formatPieces(load.pieces)} />
        <DetailStat label="Truck" value={load.tractorCode || null} />
        <DetailStat label="Trailer" value={load.trailerCode || null} />
        <DetailStat label="Role" value={load.isPrimary ? "Primary driver" : "Co-driver"} />
      </div>

      <div className="rounded-2xl border border-border bg-card p-4">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold">Route</h2>
          <Badge variant="secondary">
            {load.stops.length} stop{load.stops.length === 1 ? "" : "s"}
          </Badge>
        </div>
        <StopTimeline
          stops={load.stops}
          showDirections={isActive}
          moveId={isActive && features.allowStopActions ? load.moveId : undefined}
        />
      </div>
    </m.div>
  );
}
