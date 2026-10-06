import { useT } from "@trenova/shared/i18n/use-t";
import { CheckIcon, MessageChatCircleIcon, Send01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { formatClockDurationMs, formatMinutesSpan } from "@trenova/shared/lib/date";
import { cn, formatCurrency, formatPerMile } from "@trenova/shared/lib/utils";
import type { Shipment } from "@trenova/shared/types/shipment";
import type {
  CarrierCoverageSuggestion,
  DriverCoverageSuggestion,
} from "@/lib/graphql/shipment-board";
import { queries } from "@/lib/queries";
import { useShipmentCapabilities } from "@/lib/shipment-board/capabilities";
import { resolveCoverage, type Coverage } from "@/lib/shipment-board/coverage";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useState, type ReactNode } from "react";
import { IdentityAvatar } from "../identity-avatar";
import { useBoardActions } from "../use-board-actions";
import { useDriverHos } from "../use-driver-hos";

const DRIVE_LIMIT_MS = 11 * 3_600_000;
const LOW_HOS_MS = 4 * 3_600_000;
const COMMITTED_STATE_MS = 3_000;

function Heading({ children, tone }: { children: ReactNode; tone?: string }) {
  return <span className={cn("text-muted-foreground text-xs font-medium", tone)}>{children}</span>;
}

function OptionRow({
  avatar,
  title,
  detail,
  trailing,
  actionLabel,
  onCommit,
  pending,
}: {
  avatar: ReactNode;
  title: string;
  detail: string;
  trailing?: ReactNode;
  actionLabel: string;
  onCommit: () => void;
  pending: boolean;
}) {
  return (
    <button
      type="button"
      disabled={pending}
      onClick={onCommit}
      className="group/option ui-focus-ring hover:bg-surface-hover -mx-2 flex items-center gap-2.5 rounded-md px-2 py-1.5 text-left disabled:opacity-60"
    >
      {avatar}
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-sm font-medium">{title}</span>
        <span className="text-muted-foreground truncate font-mono text-xs tabular-nums">{detail}</span>
      </span>
      {trailing}
      <span className="text-brand text-xs font-medium opacity-0 group-hover/option:opacity-100 group-focus-visible/option:opacity-100">
        {actionLabel}
      </span>
    </button>
  );
}

function CommittedState({ coverage }: { coverage: Coverage }) {
  const t = useT();
  if (coverage.kind !== "driver" && coverage.kind !== "carrier") return null;
  const isCarrier = coverage.kind === "carrier";
  return (
    <div className="animate-success flex flex-col gap-2">
      <Heading tone="text-success">{isCarrier ? t("Tendered") : t("Assigned")}</Heading>
      <div className="flex items-center gap-2.5">
        <IdentityAvatar id={coverage.id} initials={coverage.initials} shape={isCarrier ? "square" : "circle"} className="size-7 text-xs" />
        <span className="flex min-w-0 flex-col">
          <span className="truncate text-sm font-medium">{coverage.name}</span>
          <span className="text-muted-foreground truncate text-xs">
            {isCarrier ? t("Rate con sent · awaiting acceptance") : t("{0} · dispatch sheet sent", coverage.detail ?? "")}
          </span>
        </span>
        <CheckIcon className="text-success ml-auto size-4" aria-hidden />
      </div>
    </div>
  );
}

function CoverageOptions({ shipment, onCommitted }: { shipment: Shipment; onCommitted: () => void }) {
  const t = useT();
  const { ai, runsAssets, runsBrokerage, hos } = useShipmentCapabilities();
  const actions = useBoardActions();
  const shipmentId = shipment.id ?? "";
  const { data, isLoading } = useQuery({
    ...queries.shipmentBoard.coverageSuggestions(shipmentId),
    enabled: !!shipmentId,
  });
  const pending = actions.assign.isPending || actions.tender.isPending;

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2">
        <Skeleton className="h-4 w-28" />
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-9 w-full" />
      </div>
    );
  }

  const drivers = runsAssets ? (data?.drivers ?? []) : [];
  const carriers = runsBrokerage ? (data?.carriers ?? []) : [];
  if (drivers.length === 0 && carriers.length === 0) {
    return (
      <div className="flex flex-col gap-1">
        <Heading>{t("Coverage")}</Heading>
        <p className="text-muted-foreground text-sm">
          {t("Nobody is free for this load yet. It stays in Needs coverage until someone is.")}
        </p>
      </div>
    );
  }

  const assignDriver = (driver: DriverCoverageSuggestion) =>
    actions.assign.mutate(
      [{ moveId: driver.moveId, workerId: driver.workerId, tractorId: driver.tractorId }],
      { onSuccess: onCommitted },
    );
  const tenderCarrier = (carrier: CarrierCoverageSuggestion) =>
    actions.tender.mutate([{ shipmentId, carrierId: carrier.carrierId }], {
      onSuccess: (result) => result.tendered.length > 0 && onCommitted(),
    });

  return (
    <div className="flex flex-col gap-1">
      {drivers.length > 0 ? (
        <>
          <Heading>{ai ? t("Suggested drivers") : t("Nearest drivers")}</Heading>
          {drivers.map((driver) => (
            <OptionRow
              key={driver.workerId}
              avatar={<IdentityAvatar id={driver.workerId} initials={driver.initials} shape="circle" className="size-7 text-xs" />}
              title={driver.name}
              detail={[
                driver.distanceMiles != null ? t("{0} mi out", Math.round(driver.distanceMiles)) : null,
                hos && driver.driveRemainingMs != null
                  ? t("{0} HOS", formatClockDurationMs(driver.driveRemainingMs))
                  : null,
              ]
                .filter(Boolean)
                .join(" · ")}
              trailing={
                ai && driver.fitPercent != null ? (
                  <span className="text-success font-mono text-xs tabular-nums">{Math.round(driver.fitPercent)}%</span>
                ) : null
              }
              actionLabel={t("Assign")}
              onCommit={() => assignDriver(driver)}
              pending={pending}
            />
          ))}
        </>
      ) : null}
      {carriers.length > 0 ? (
        <>
          <Heading tone={drivers.length > 0 ? "mt-2" : undefined}>
            {drivers.length > 0
              ? t("Or tender to a carrier")
              : ai
                ? t("Suggested carriers")
                : t("Carriers on this lane")}
          </Heading>
          {carriers.map((carrier) => (
            <OptionRow
              key={carrier.carrierId}
              avatar={<IdentityAvatar id={carrier.carrierId} initials={carrier.initials} shape="square" className="size-7 text-xs" />}
              title={carrier.name}
              detail={[
                formatCurrency(Number(carrier.quote)),
                formatPerMile(Number(carrier.ratePerMile)),
                carrier.acceptancePercent != null ? t("{0}% accept", Math.round(carrier.acceptancePercent)) : null,
              ]
                .filter(Boolean)
                .join(" · ")}
              actionLabel={t("Tender")}
              onCommit={() => tenderCarrier(carrier)}
              pending={pending}
            />
          ))}
        </>
      ) : null}
    </div>
  );
}

function DelayNotice({ shipment }: { shipment: Shipment }) {
  const t = useT();
  const { ai } = useShipmentCapabilities();
  const actions = useBoardActions();
  const slack = shipment.eta?.slackMinutes;
  const behind = slack != null && slack < 0 ? formatMinutesSpan(-slack) : null;
  const reason = shipment.eta?.reason ?? t("a delay on the road");
  const customerName = shipment.customer?.name ?? t("the customer");
  const draft = t(
    "{0} has held up {1}. The new ETA is {2}; we will confirm a new delivery window within the hour.",
    reason,
    shipment.proNumber ?? "",
    behind ? t("{0} behind the appointment", behind) : t("being confirmed"),
  );
  const [message, setMessage] = useState(draft);
  const [editing, setEditing] = useState(false);
  const [sent, setSent] = useState(false);

  if (sent) {
    return (
      <div className="animate-success flex flex-col gap-1">
        <Heading tone="text-success">{t("Update sent")}</Heading>
        <p className="text-muted-foreground text-sm">{t("{0} has the new ETA.", customerName)}</p>
      </div>
    );
  }

  const send = () =>
    actions.notifyDelay.mutate(
      { shipmentId: shipment.id ?? "", message },
      { onSuccess: () => setSent(true) },
    );

  return (
    <div className="flex flex-col gap-2">
      <Heading tone="text-danger">{ai ? t("Drafted update") : t("Running late")}</Heading>
      {ai ? (
        editing ? (
          <Textarea value={message} onChange={(event) => setMessage(event.target.value)} rows={4} className="text-sm" />
        ) : (
          <p className="text-sm leading-relaxed">“{message}”</p>
        )
      ) : (
        <p className="text-sm">
          {[behind ? t("{0} behind", behind) : null, shipment.eta?.reason].filter(Boolean).join(" · ")}
          {". "}
          {t("The appointment will be missed.")}
        </p>
      )}
      <div className="flex items-center gap-1.5">
        <Button size="sm" onClick={send} isLoading={actions.notifyDelay.isPending} loadingText={t("Sending")}>
          <Send01Icon className="size-3.5" />
          {ai ? t("Send to {0}", customerName.split(" ")[0]) : t("Notify customer")}
        </Button>
        {ai ? (
          <Button size="sm" variant="ghost" onClick={() => setEditing((value) => !value)}>
            {editing ? t("Done") : t("Edit")}
          </Button>
        ) : null}
      </div>
    </div>
  );
}

function CoveredBy({ shipment, coverage }: { shipment: Shipment; coverage: Coverage }) {
  const t = useT();
  const driveRemainingMs = useDriverHos(coverage.kind === "driver" ? coverage.id : null);
  if (coverage.kind !== "driver" && coverage.kind !== "carrier") return null;
  const isCarrier = coverage.kind === "carrier";
  return (
    <div className="flex flex-col gap-2">
      <Heading>{t("Coverage")}</Heading>
      <div className="flex items-center gap-2.5">
        <IdentityAvatar id={coverage.id} initials={coverage.initials} shape={isCarrier ? "square" : "circle"} className="size-7 text-xs" />
        <span className="flex min-w-0 flex-1 flex-col">
          <span className="truncate text-sm font-medium">{coverage.name}</span>
          <span className="text-muted-foreground truncate text-xs">{coverage.detail ?? shipment.proNumber}</span>
        </span>
        <Button variant="ghost" size="icon-sm" aria-label={isCarrier ? t("Message carrier") : t("Message driver")}>
          <MessageChatCircleIcon className="size-4" />
        </Button>
      </div>
      {driveRemainingMs != null ? (
        <div className="flex items-center gap-2">
          <span className="bg-muted h-1.5 flex-1 overflow-hidden rounded-full">
            <span
              className={cn("block h-full rounded-full", driveRemainingMs < LOW_HOS_MS ? "bg-warning" : "bg-success")}
              style={{ width: `${Math.min(100, (driveRemainingMs / DRIVE_LIMIT_MS) * 100)}%` }}
            />
          </span>
          <span className="text-muted-foreground font-mono text-xs tabular-nums">
            {t("{0} of 11:00 drive", formatClockDurationMs(driveRemainingMs))}
          </span>
        </div>
      ) : null}
    </div>
  );
}

/**
 * The one thing the open row asks of the dispatcher, chosen by the load's
 * state: cover it, tell the customer it is late, or see who has it.
 */
export function NextStep({ shipment }: { shipment: Shipment }) {
  const coverage = resolveCoverage(shipment);
  const [justCommitted, setJustCommitted] = useState(false);

  useEffect(() => {
    if (!justCommitted) return;
    const timer = window.setTimeout(() => setJustCommitted(false), COMMITTED_STATE_MS);
    return () => window.clearTimeout(timer);
  }, [justCommitted]);

  if (justCommitted && (coverage.kind === "driver" || coverage.kind === "carrier")) {
    return <CommittedState coverage={coverage} />;
  }
  if (coverage.kind === "uncovered") {
    return <CoverageOptions shipment={shipment} onCommitted={() => setJustCommitted(true)} />;
  }
  if (shipment.stage === "Late") {
    return <DelayNotice shipment={shipment} />;
  }
  if (coverage.kind === "tendered") {
    return <CoverageOptions shipment={shipment} onCommitted={() => setJustCommitted(true)} />;
  }
  return <CoveredBy shipment={shipment} coverage={coverage} />;
}
