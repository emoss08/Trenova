import { useT } from "@trenova/shared/i18n/use-t";
import { CapabilityGate } from "@/components/capability-gate";
import { RateConfirmationActions } from "@/components/carrier-assignment/rate-confirmation-actions";
import { VerifyEquipmentDialog } from "@/components/carrier-assignment/verify-equipment-dialog";
import { PermissionGate } from "@/components/permission-gate";
import { recordPath } from "@/config/record-links";
import { usePermission } from "@/hooks/use-permission";
import { isTypingTarget, isWithinDialog, isWithinPopup } from "@/lib/dom";
import type {
  DispatchBoardDriver,
  DispatchBoardMove,
  DispatchCandidate,
} from "@/lib/graphql/dispatch-console";
import { dispatchConsoleQueries } from "@/lib/queries/dispatch-console";
import type { TimeRange } from "@/lib/timeline/time-scale";
import { useDispatchConsoleStore } from "@/stores/dispatch-console-store";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import {
  Building07Icon,
  CalendarIcon,
  CheckCircleIcon,
  FlameIcon,
  Lock01Icon,
  ScanIcon,
  Send01Icon,
  Snowflake01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import { formatUnixDateMedium, formatUnixTime } from "@trenova/shared/lib/date";
import { cn, formatCompactCurrency, formatCurrency } from "@trenova/shared/lib/utils";
import {
  hasOrganizationCapability,
  OrganizationCapability,
} from "@trenova/shared/types/organization-capability";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { Link } from "react-router";
import { previousAssignmentOf } from "./assignment-input";
import { CandidateList, isPickable, MAX_HOTKEY_RANK } from "./candidate-list";
import { InspectorSkeleton } from "./console-skeletons";
import { DriverInspector } from "./driver-inspector";
import { InspectorScroll } from "./inspector-scroll";
import { formatMiles, pickupState } from "./dispatch-vocabulary";
import { formatWhen } from "./format-when";
import { FailedApproval } from "./failed-approval";
import { NovaProposalPanel, useProposalApproval, type DeclineDraft } from "./nova-proposal-panel";
import { PlanPricePanel } from "./plan-price-panel";
import { TenderHistory } from "./tender-history";
import { TenderLivePanel } from "./tender-live-panel";
import type { DispatchActions } from "./use-dispatch-actions";

const CANDIDATE_STALE_MS = 30_000;
const CLOSED_DECLINE: DeclineDraft = { open: false, note: "" };

/** A key the inspector may answer: not typed into a field, a dialog or someone else's list. */
function isInspectorKey(event: KeyboardEvent, scopes: readonly (Element | null)[]): boolean {
  if (event.metaKey || event.ctrlKey || event.altKey) return false;
  if (isTypingTarget(event.target) || isWithinDialog(event.target)) return false;
  if (isWithinPopup(event.target)) {
    return scopes.some(
      (scope) => scope !== null && event.target instanceof Node && scope.contains(event.target),
    );
  }
  return true;
}

function CarrierCoverageCard({ move }: { move: DispatchBoardMove }) {
  const t = useT();

  const openCarrierAssign = useDispatchConsoleStore((s) => s.openCarrierAssign);
  const openCarrierCancel = useDispatchConsoleStore((s) => s.openCarrierCancel);
  const [verifyOpen, setVerifyOpen] = useState(false);

  return (
    <div className="bg-card border-border flex flex-col gap-1.5 rounded-md border px-3 py-2.5">
      <div className="flex items-center gap-1.5">
        <Building07Icon className="text-muted-foreground size-3.5 shrink-0" aria-hidden />
        <span className="truncate text-xs font-medium">{move.assignedCarrierName}</span>
        <Badge variant="success" className="h-4 shrink-0 rounded-md px-1 text-3xs">
          {t("Carrier")}
        </Badge>
      </div>
      {move.carrierTotalCost != null && (
        <span className="text-muted-foreground text-2xs tabular-nums">
          {t("Total cost {0}", formatCurrency(move.carrierTotalCost))}
        </span>
      )}
      {move.carrierAssignmentId && (
        <RateConfirmationActions
          moveId={move.moveId}
          carrierAssignmentId={move.carrierAssignmentId}
        />
      )}
      <div className="flex flex-wrap items-center gap-1.5">
        {/* Re-brokering is a brokerage action; canceling is the way an
            asset-only org unwinds coverage it inherited, so it always stays. */}
        <CapabilityGate capability={OrganizationCapability.Brokerage}>
          <Button
            size="sm"
            variant="outline"
            className="h-6 px-2 text-2xs"
            title={t(
              "Broker this move to a different carrier — the current assignment is replaced",
            )}
            onClick={() => openCarrierAssign(move)}
          >
            {t("Replace carrier")}
          </Button>
        </CapabilityGate>
        {move.carrierAssignmentId && (
          <PermissionGate resource={Resource.EquipmentVerification} operation={Operation.Create}>
            <Button
              size="sm"
              variant="outline"
              className="h-6 px-2 text-2xs"
              onClick={() => setVerifyOpen(true)}
            >
              <ScanIcon className="size-3" aria-hidden />
              {t("Verify equipment")}
            </Button>
          </PermissionGate>
        )}
        <Button
          size="sm"
          variant="outline"
          className="h-6 px-2 text-2xs"
          onClick={() => openCarrierCancel(move)}
        >
          {t("Cancel carrier assignment")}
        </Button>
      </div>
      {verifyOpen && move.carrierAssignmentId ? (
        <VerifyEquipmentDialog
          carrierAssignmentId={move.carrierAssignmentId}
          carrierName={move.assignedCarrierName ?? null}
          open
          onOpenChange={setVerifyOpen}
        />
      ) : null}
    </div>
  );
}

function TenderCoverageCard({
  move,
  actions,
}: {
  move: DispatchBoardMove;
  actions: DispatchActions;
}) {
  const openCarrierAssign = useDispatchConsoleStore((s) => s.openCarrierAssign);

  const { data: liveTender, isLoading } = useQuery({
    ...dispatchConsoleQueries.liveTender(move.moveId),
  });

  // The board summary can lag the live query: a tender that just resolved still
  // flags the move while liveTender comes back null. Rendering the frame anyway
  // would leave an empty bordered strip in the inspector.
  if (!isLoading && !liveTender) {
    return null;
  }

  return (
    <div className="bg-card border-border flex flex-col gap-1.5 rounded-md border px-3 py-2.5">
      {isLoading ? (
        <Skeleton className="h-20 rounded-md" />
      ) : liveTender ? (
        <TenderLivePanel
          tender={liveTender}
          isTendering={actions.isTendering}
          onCancelTender={actions.cancelTender}
          onRecordResponse={actions.recordOfferResponse}
          onBounceTender={actions.bounceTender}
          onAssignManually={() => openCarrierAssign(move)}
        />
      ) : null}
    </div>
  );
}

function MoveFlags({ move }: { move: DispatchBoardMove }) {
  const t = useT();

  const flags = [
    move.temperatureMin != null ? (
      <Badge key="reefer" variant="info" className="h-5 rounded-md px-1.5 text-2xs">
        <Snowflake01Icon className="mr-0.5 size-3" aria-hidden />
        {t("Reefer {0}–{1}°F", move.temperatureMin, move.temperatureMax)}
      </Badge>
    ) : null,
    move.hasHazmat ? (
      <Badge key="hazmat" variant="danger" className="h-5 rounded-md px-1.5 text-2xs">
        <FlameIcon className="mr-0.5 size-3" aria-hidden />
        {t("Hazmat")}
      </Badge>
    ) : null,
    move.hasActiveHold ? (
      <Badge key="hold" variant="warning" className="h-5 rounded-md px-1.5 text-2xs">
        <Lock01Icon className="mr-0.5 size-3" aria-hidden />
        {t("On hold")}
      </Badge>
    ) : null,
  ].filter(Boolean);

  return flags.length > 0 ? (
    <div className="flex flex-wrap items-center gap-1.5">{flags}</div>
  ) : null;
}

function MoveHeader({
  move,
  nowSeconds,
  onClose,
}: {
  move: DispatchBoardMove;
  nowSeconds: number;
  onClose: () => void;
}) {
  const t = useT();

  const when = (seconds: number) => formatWhen(seconds, nowSeconds);
  const pickup = pickupState(move, nowSeconds, when);
  const sameDay = (a: number, b: number) =>
    new Date(a * 1000).toDateString() === new Date(b * 1000).toDateString();
  const pickupWindow =
    move.originWindowStart > 0
      ? move.originWindowEnd && move.originWindowEnd > move.originWindowStart
        ? t(
            "{0}–{1}",
            when(move.originWindowStart),
            sameDay(move.originWindowStart, move.originWindowEnd)
              ? formatUnixTime(move.originWindowEnd)
              : when(move.originWindowEnd),
          )
        : when(move.originWindowStart)
      : t("No appointment");
  const deliveryWindow =
    move.destinationWindowStart > 0 ? when(move.destinationWindowStart) : t("No appointment");

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center gap-2">
        <span className="font-mono text-xs font-semibold">{move.proNumber}</span>
        {move.moveCount > 1 ? (
          <Badge variant="neutral" appearance="outline" className="h-4 rounded-md px-1 text-3xs">
            {t("Leg {0} of {1}", move.sequence + 1, move.moveCount)}
          </Badge>
        ) : null}
        <Button
          variant="link"
          size="xxs"
          className="ml-auto"
          nativeButton={false}
          render={<Link to={recordPath("shipment", move.shipmentId)} />}
        >
          {t("Open shipment")}
        </Button>
        <Button
          variant="ghost"
          size="icon-xs"
          aria-label={t("Close inspector")}
          title={t("Close (Esc)")}
          onClick={onClose}
        >
          <XCloseIcon className="size-3.5" />
        </Button>
      </div>
      <h3 className="text-base leading-6 font-semibold">
        {t(
          "{0} → {1}",
          [move.originCity || move.originName, move.originState].filter(Boolean).join(", "),
          [move.destinationCity || move.destinationName, move.destinationState]
            .filter(Boolean)
            .join(", "),
        )}
      </h3>
      <div
        className={cn(
          "text-xs font-medium",
          pickup.tone === "danger"
            ? "text-danger-foreground"
            : pickup.tone === "warning"
              ? "text-warning-foreground"
              : "text-muted-foreground",
        )}
      >
        {pickup.headline}
      </div>
      <MoveFlags move={move} />
      <DescriptionList layout="inline" className="mt-1.5 text-xs">
        <DescriptionItem label={t("Customer")}>{move.customerName || "—"}</DescriptionItem>
        <DescriptionItem label={t("Pickup")}>
          {move.originName ? t("{0} · {1}", move.originName, pickupWindow) : pickupWindow}
        </DescriptionItem>
        <DescriptionItem label={t("Delivery")}>
          {move.destinationName
            ? t("{0} · {1}", move.destinationName, deliveryWindow)
            : deliveryWindow}
        </DescriptionItem>
        <DescriptionItem label={t("Distance")}>
          <span className="tabular-nums">
            {move.distance != null && move.revenue != null
              ? t("{0} · {1}", formatMiles(move.distance), formatCompactCurrency(move.revenue))
              : move.distance != null
                ? formatMiles(move.distance)
                : move.revenue != null
                  ? t("Distance not set · {0}", formatCompactCurrency(move.revenue))
                  : t("Not set")}
          </span>
        </DescriptionItem>
      </DescriptionList>
    </div>
  );
}

function MoveInspector({
  move,
  outsideWindow,
  onJumpTo,
  hotkeysEnabled,
  queueScope,
  onClose,
  onNextMove,
  onCovered,
  actions,
}: {
  move: DispatchBoardMove;
  outsideWindow: boolean;
  onJumpTo: (move: DispatchBoardMove) => void;
  hotkeysEnabled: boolean;
  queueScope: Element | null;
  onClose: () => void;
  onNextMove: () => void;
  onCovered: (moveId: string) => void;
  actions: DispatchActions;
}) {
  const t = useT();
  const nowSeconds = useNow("minute");
  const scopeRef = useRef<HTMLDivElement>(null);

  const [includeBlocked, setIncludeBlocked] = useState(false);
  const [armedWorkerId, setArmedWorkerId] = useState<string | null>(null);
  const [decline, setDecline] = useState<DeclineDraft>(CLOSED_DECLINE);

  const openCarrierAssign = useDispatchConsoleStore((s) => s.openCarrierAssign);
  const openTender = useDispatchConsoleStore((s) => s.openTender);
  const candidatesOpen = useDispatchConsoleStore((s) => s.candidatesOpen);
  const setCandidatesOpen = useDispatchConsoleStore((s) => s.setCandidatesOpen);
  const capabilities = useOrgCapabilities();
  // Ranking drivers is the whole point of this pane, and an organization that employs
  // none has nothing to rank: the ranking is withheld and its query never issued.
  const canRankDrivers = hasOrganizationCapability(
    capabilities,
    OrganizationCapability.AssetOperations,
  );
  const { allowed: mayAssign } = usePermission(Resource.ShipmentMove, Operation.Assign);
  const { allowed: canTender } = usePermission(Resource.Tender, Operation.Create);
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Update);
  const isCarrierCovered = move.coverageType === "carrier";
  // A carrier-covered move cannot take a driver on top; the carrier assignment is
  // canceled first, so driver assignment is held off rather than allowed to fail.
  const canAssign = mayAssign && !isCarrierCovered && !move.hasActiveHold;
  const showsCarrierPaths =
    hasOrganizationCapability(capabilities, OrganizationCapability.Brokerage) &&
    !move.isCovered &&
    (mayAssign || canTender);

  const candidatesQuery = useQuery({
    ...dispatchConsoleQueries.moveCandidates({ moveId: move.moveId, includeBlocked }),
    staleTime: CANDIDATE_STALE_MS,
    enabled: canRankDrivers,
  });
  const candidates = candidatesQuery.data ?? [];

  const proposal = move.isCovered ? null : (move.pendingProposal ?? null);
  const proposedCandidate = proposal
    ? (candidates.find((candidate) => candidate.workerId === proposal.workerId) ?? null)
    : null;
  const approval = useProposalApproval(proposal, canDecide);

  const proposalDecision = proposal
    ? {
        proposalId: proposal.id,
        moveId: move.moveId,
        proNumber: move.proNumber,
        driverName: proposal.workerName,
      }
    : null;

  const approve = async (previewDigest: string | undefined) => {
    if (!proposalDecision) return;
    const approved = await actions.approveProposal({ ...proposalDecision, previewDigest });
    if (approved) onCovered(move.moveId);
  };

  const declineWith = async (note: string) => {
    if (!proposalDecision) return;
    const declined = await actions.declineProposal(proposalDecision, note);
    if (declined) setDecline(CLOSED_DECLINE);
  };

  const confirmAssign = async (
    candidate: DispatchCandidate,
    params: { tractorId: string; trailerId?: string },
  ) => {
    const assigned = await actions.assign([
      {
        input: {
          moveId: move.moveId,
          primaryWorkerId: candidate.workerId,
          tractorId: params.tractorId,
          trailerId: params.trailerId,
          reassign: move.isCovered,
        },
        proNumber: move.proNumber,
        driverName: candidate.workerName,
        previous: previousAssignmentOf(move),
      },
    ]);
    if (assigned) {
      setArmedWorkerId(null);
      onCovered(move.moveId);
    }
  };

  // The inspector's own keys: A and D decide Nova's proposal, a digit opens that driver's
  // pre-flight, and Esc backs out of whatever is open here before it closes the pane.
  // Capture runs this before the page's keys, so Esc never closes the pane under an open
  // pre-flight.
  useEffect(() => {
    if (!hotkeysEnabled) return;

    function onKeyDown(event: KeyboardEvent) {
      if (event.defaultPrevented) return;
      if (event.key === "Escape" && (armedWorkerId || decline.open)) {
        if (isWithinDialog(event.target)) return;
        event.preventDefault();
        setArmedWorkerId(null);
        setDecline(CLOSED_DECLINE);
        return;
      }
      if (!isInspectorKey(event, [scopeRef.current, queueScope])) return;

      const key = event.key.toLowerCase();
      if (key === "a" && approval.approvable && !actions.isDeciding && !decline.open) {
        event.preventDefault();
        void approve(approval.digest);
        return;
      }
      if (key === "d" && proposal && canDecide && !decline.open) {
        event.preventDefault();
        setArmedWorkerId(null);
        setDecline({ open: true, note: "" });
        return;
      }
      const rank = Number.parseInt(event.key, 10);
      if (Number.isNaN(rank) || rank < 1 || rank > MAX_HOTKEY_RANK || !canAssign) return;
      const candidate = candidates[rank - 1];
      if (!candidate || !isPickable(candidate)) return;
      event.preventDefault();
      setDecline(CLOSED_DECLINE);
      // A digit reaches a driver even with the list folded away, and opens it so the
      // pre-flight it starts is in view.
      setCandidatesOpen(true);
      setArmedWorkerId(candidate.workerId);
    }

    window.addEventListener("keydown", onKeyDown, { capture: true });
    return () => window.removeEventListener("keydown", onKeyDown, { capture: true });
  });

  const tenderHistory = (
    <CapabilityGate capability={OrganizationCapability.Brokerage}>
      <TenderHistory
        shipmentId={move.shipmentId}
        isTendering={actions.isTendering}
        onBounceTender={actions.bounceTender}
      />
    </CapabilityGate>
  );

  return (
    <div ref={scopeRef} className="animate-panel-in flex min-h-0 flex-1 flex-col">
      <InspectorScroll className="flex flex-col gap-3 p-3.5">
        <MoveHeader move={move} nowSeconds={nowSeconds} onClose={onClose} />

        {outsideWindow ? (
          <div className="bg-brand-subtle border-brand/30 flex items-start gap-2 rounded-md border px-3 py-2.5 text-xs">
            <CalendarIcon className="mt-px size-3.5 shrink-0" aria-hidden />
            <div>
              <span className="font-medium">{t("Outside this window.")}</span>{" "}
              {move.originWindowStart > 0
                ? t(
                    "{0} picks up {1}.",
                    move.proNumber,
                    formatUnixDateMedium(move.originWindowStart),
                  )
                : t("{0} has no pickup appointment.", move.proNumber)}{" "}
              {move.originWindowStart > 0 ? (
                <Button variant="link" size="xxs" onClick={() => onJumpTo(move)}>
                  {t("Jump to {0}", formatUnixDateMedium(move.originWindowStart))}
                </Button>
              ) : null}
            </div>
          </div>
        ) : null}

        {move.isCovered && !isCarrierCovered ? (
          <div className="bg-success-subtle border-success/30 flex items-start gap-2 rounded-md border px-3 py-2.5 text-xs">
            <CheckCircleIcon
              className="text-success-foreground mt-px size-3.5 shrink-0"
              aria-hidden
            />
            <div className="min-w-0 flex-1">
              <span className="font-medium">
                {move.assignedTractorCode
                  ? t("Covered by {0} · {1}", move.assignedWorkerName, move.assignedTractorCode)
                  : t("Covered by {0}", move.assignedWorkerName)}
              </span>
              {canAssign ? (
                <span className="text-muted-foreground block">
                  {t("Pick another driver below to reassign it.")}
                </span>
              ) : null}
            </div>
            <Button
              variant="link"
              size="xxs"
              className="shrink-0"
              onClick={onNextMove}
              shortcut="J"
            >
              {t("Next move")}
            </Button>
          </div>
        ) : null}

        {isCarrierCovered && <CarrierCoverageCard move={move} />}

        {move.liveTender && (
          <CapabilityGate capability={OrganizationCapability.Brokerage}>
            <TenderCoverageCard move={move} actions={actions} />
          </CapabilityGate>
        )}

        {!proposal && !move.isCovered && move.failedProposal ? (
          <FailedApproval move={move} failed={move.failedProposal} />
        ) : null}

        {proposal ? (
          <NovaProposalPanel
            move={move}
            proposal={proposal}
            candidate={proposedCandidate}
            canDecide={canDecide}
            approval={approval}
            isDeciding={actions.isDeciding}
            decline={decline}
            onDeclineChange={setDecline}
            onApprove={approve}
            onDecline={declineWith}
          />
        ) : null}

        {!move.isCovered && move.planPrice ? (
          <PlanPricePanel price={move.planPrice} nowSeconds={nowSeconds} />
        ) : null}

        {showsCarrierPaths && (
          <div className="flex items-center gap-1.5">
            {mayAssign && (
              <Button
                size="sm"
                variant="outline"
                className="flex-1"
                disabled={actions.isAssigning}
                onClick={() => openCarrierAssign(move)}
              >
                <Building07Icon className="size-3" aria-hidden />
                {t("Assign to carrier")}
              </Button>
            )}
            {canTender && (
              <Button
                size="sm"
                variant="outline"
                className="flex-1"
                disabled={actions.isAssigning}
                onClick={() => openTender(move)}
              >
                <Send01Icon className="size-3" aria-hidden />
                {t("Tender to carriers")}
              </Button>
            )}
          </div>
        )}

        {canRankDrivers && !isCarrierCovered ? (
          <CandidateList
            move={move}
            read={{
              candidates,
              isPending: candidatesQuery.isPending,
              isError: candidatesQuery.isError,
              error: candidatesQuery.error,
              refetch: () => void candidatesQuery.refetch(),
            }}
            includeBlocked={includeBlocked}
            onToggleBlocked={() => setIncludeBlocked((value) => !value)}
            armedWorkerId={armedWorkerId}
            proposedWorkerId={proposal?.workerId ?? null}
            canAssign={canAssign}
            isAssigning={actions.isAssigning}
            onArm={(candidate) => {
              setDecline(CLOSED_DECLINE);
              setArmedWorkerId(candidate.workerId);
            }}
            onDisarm={() => setArmedWorkerId(null)}
            onConfirm={confirmAssign}
            onTender={showsCarrierPaths && canTender ? () => openTender(move) : null}
            open={candidatesOpen}
            onOpenChange={setCandidatesOpen}
          />
        ) : null}

        {tenderHistory}
      </InspectorScroll>
      <footer className="border-border text-muted-foreground flex flex-wrap gap-x-3.5 gap-y-1 border-t px-3.5 py-2 text-2xs">
        {candidatesQuery.dataUpdatedAt > 0 ? (
          <span>
            {t("Ranked {0}", formatUnixTime(Math.floor(candidatesQuery.dataUpdatedAt / 1000)))}
          </span>
        ) : null}
        <span>{t("Assign opens a pre-flight; nothing is written until you confirm it")}</span>
      </footer>
    </div>
  );
}

/**
 * The inspector matches in both directions, because dispatchers work in both: covering a
 * load, and finding work for an idle truck. It opens when something is selected and
 * closes by width when nothing is, so the queue takes the room back.
 */
export function Inspector({
  move,
  moveRead,
  outsideWindow,
  driver,
  range,
  hotkeysEnabled,
  queueScope,
  onClose,
  onSelectMove,
  onNextMove,
  onCovered,
  onJumpTo,
  actions,
}: {
  move: DispatchBoardMove | null;
  /** The read of a selected move that is not on the board; null when it is. */
  moveRead: { isPending: boolean; isError: boolean; error: unknown; refetch: () => void } | null;
  outsideWindow: boolean;
  driver: DispatchBoardDriver | null;
  range: TimeRange;
  hotkeysEnabled: boolean;
  queueScope: Element | null;
  onClose: () => void;
  onSelectMove: (moveId: string) => void;
  onNextMove: () => void;
  onCovered: (moveId: string) => void;
  onJumpTo: (move: DispatchBoardMove) => void;
  actions: DispatchActions;
}) {
  const t = useT();

  if (move) {
    return (
      <MoveInspector
        key={move.moveId}
        move={move}
        outsideWindow={outsideWindow}
        onJumpTo={onJumpTo}
        hotkeysEnabled={hotkeysEnabled}
        queueScope={queueScope}
        onClose={onClose}
        onNextMove={onNextMove}
        onCovered={onCovered}
        actions={actions}
      />
    );
  }

  if (moveRead?.isPending) {
    return (
      <InspectorScroll>
        <InspectorSkeleton />
      </InspectorScroll>
    );
  }

  if (moveRead?.isError) {
    return (
      <InspectorScroll className="p-3.5">
        <ErrorState
          error={moveRead.error}
          layout="compact"
          title={t("Couldn't open this move")}
          description={t("It isn't on the board in this window, and reading it on its own failed.")}
          onRetry={moveRead.refetch}
          actions={
            <Button size="sm" variant="outline" onClick={onClose}>
              {t("Close")}
            </Button>
          }
        />
      </InspectorScroll>
    );
  }

  if (driver) {
    return (
      <DriverInspector
        key={driver.workerId}
        driver={driver}
        range={range}
        onSelectMove={onSelectMove}
        onClose={onClose}
      />
    );
  }

  return null;
}
