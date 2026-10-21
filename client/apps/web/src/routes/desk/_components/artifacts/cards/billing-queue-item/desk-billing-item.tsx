import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { recordPath } from "@/config/record-links";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { assignBillingQueueBillerGraphQL } from "@/lib/graphql/billing-queue";
import { fetchGraphQLSelectOptions } from "@/lib/graphql/select-options";
import { billingQueue } from "@/lib/queries/billing-queue";
import { apiService } from "@/services/api";
import { useInfiniteQuery, useQuery, useQueryClient } from "@tanstack/react-query";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type {
  BillingCheck,
  BillingChargeLine,
  BillingIssue,
  BillingQueueEvent,
  BillingQueueHoldReason,
  BillingQueueItem,
  BillingQueuePostResult,
} from "@trenova/shared/types/billing-queue";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useCallback, useState, type ReactNode } from "react";
import { deskPopoverClass } from "@/components/desk-chat/desk-popover";
import {
  HOLD_REASONS,
  barReason,
  checkDetail,
  checkTitle,
  holdReasonLabel,
  initials,
  issueOf,
  itemStage,
  paymentTermLabel,
  type ItemStage,
  stageLabel,
} from "./billing-item-checks";
import {
  CardBar,
  CardBarHint,
  CardCheck,
  CardChecks,
  CardFacts,
  CardHead,
  CardIcon,
  CardLink,
  CardSection,
  CardStatus,
  CardStepper,
  CardTimeline,
  RecordCard,
  type CardTone,
} from "../../record-card/record-card";
import { CardDocuments } from "../../record-card/card-documents";
import { cardDay, cardMoment, money } from "../../record-card/record-card-values";

function daysSince(seconds: number): number {
  return Math.max(0, Math.floor((Date.now() / 1000 - seconds) / 86_400));
}

const STAGE_TONE: Record<ItemStage, CardTone> = {
  review: "progress",
  approved: "success",
  posted: "ink",
  held: "warn",
  other: "progress",
};

type Stop = NonNullable<Shipment["moves"]>[number]["stops"][number];

function placeOf(stop: Stop | undefined): string {
  const location = stop?.location;
  if (!location) return "";
  const state = (location as { state?: { abbreviation?: string } | null }).state?.abbreviation;
  return [location.city, state].filter(Boolean).join(", ") || location.name;
}

/** The lane as the design draws it: from and to, then delivered · driver · miles. */
function laneOf(shipment: BillingQueueItem["shipment"], t: TranslateFn) {
  const moves = shipment?.moves ?? [];
  const stops = moves.flatMap((move) => move.stops);
  if (stops.length === 0) return null;
  const from = stops.find((stop) => stop.type === "Pickup") ?? stops[0];
  const to = [...stops].reverse().find((stop) => stop.type === "Delivery") ?? stops.at(-1);
  const driver = moves.find((move) => move.assignment?.primaryWorker)?.assignment?.primaryWorker;
  const driverName =
    driver?.wholeName || [driver?.firstName, driver?.lastName].filter(Boolean).join(" ");
  const miles = moves
    .filter((move) => move.loaded !== false)
    .reduce((sum, move) => sum + (Number(move.distance) || 0), 0);
  return {
    from: placeOf(from),
    to: placeOf(to),
    detail: [
      to?.actualArrival ? t("Delivered {0}", cardDay(to.actualArrival)) : "",
      driverName,
      miles > 0 ? t("{0} mi", Math.round(miles).toLocaleString()) : "",
    ]
      .filter((part) => part !== "")
      .join(" · "),
  };
}

type Biller = { id: string; name: string };

/** The billers to hand an item to: the person first, marked as them. */
function BillerPicker({
  me,
  busy,
  onPick,
}: {
  me: Biller | null;
  busy: boolean;
  onPick: (biller: Biller) => void;
}) {
  const t = useT();
  const { data } = useQuery({
    queryKey: ["desk-billers"],
    queryFn: ({ signal }) =>
      fetchGraphQLSelectOptions({ resource: "USER", initialLimit: 8 }, { signal }),
    staleTime: 60_000,
  });
  const others = (data?.results ?? [])
    .filter((option) => option.id !== me?.id)
    .map((option) => ({ id: option.id, name: option.label }));
  const billers = me ? [me, ...others] : others;

  return (
    <div className="dk-bi-opts">
      {billers.map((biller) => (
        <Button
          key={biller.id}
          variant="outline"
          size="sm"
          className="disabled:opacity-55"
          disabled={busy}
          onClick={() => onPick(biller)}
        >
          <span className="dk-bi-av">{initials(biller.name)}</span>
          {biller.name}
          {biller.id === me?.id && <em>{t("you")}</em>}
        </Button>
      ))}
    </div>
  );
}

function CheckRow({
  item,
  check,
  canAct,
  busy,
  me,
  onAssign,
  onResolve,
}: {
  item: BillingQueueItem;
  check: BillingCheck;
  canAct: boolean;
  busy: boolean;
  me: Biller | null;
  onAssign: (biller: Biller) => void;
  onResolve: (issue: BillingIssue, optionKey: string) => void;
}) {
  const t = useT();
  const [picking, setPicking] = useState(false);
  const issue = issueOf(item, check);
  const ok = check.state === "ok";

  return (
    <CardCheck
      state={check.state === "fail" ? "bad" : check.state}
      title={checkTitle(check.key, t)}
      detail={issue && !ok ? issue.summary : checkDetail(check, t)}
    >
      {!ok &&
        check.code === "unassigned" &&
        canAct &&
        (picking ? (
          <BillerPicker
            me={me}
            busy={busy}
            onPick={(biller) => {
              setPicking(false);
              onAssign(biller);
            }}
          />
        ) : (
          <div className="dk-bi-opts">
            {me && (
              <Button
                size="sm"
                className="disabled:opacity-55"
                disabled={busy}
                onClick={() => onAssign(me)}
              >
                {t("Assign to me")}
              </Button>
            )}
            <Button
              variant="outline"
              size="sm"
              className="disabled:opacity-55"
              disabled={busy}
              onClick={() => setPicking(true)}
            >
              {t("Someone else…")}
            </Button>
          </div>
        ))}
      {!ok && issue && (
        <>
          {issue.reasoning && <p className="dk-bi-why">{issue.reasoning}</p>}
          {canAct && issue.options.length > 0 && (
            <div className="dk-bi-opts">
              {issue.options.map((option, index) => {
                const asked = option.effect.kind === "request" && issue.requestedAt;
                return (
                  <Button
                    key={option.key}
                    variant={index === 0 && !asked ? "default" : "outline"}
                    size="sm"
                    className="disabled:opacity-55"
                    disabled={busy || Boolean(asked)}
                    onClick={() => onResolve(issue, option.key)}
                  >
                    {asked ? t("Asked {0}", cardMoment(issue.requestedAt)) : option.label}
                  </Button>
                );
              })}
            </div>
          )}
        </>
      )}
    </CardCheck>
  );
}

function LedgerRow({
  line,
  index,
  canUndo,
  busy,
  onUndo,
}: {
  line: BillingChargeLine;
  index: number;
  canUndo: boolean;
  busy: boolean;
  onUndo: (issueId: string) => void;
}) {
  const t = useT();
  return (
    <div
      className={cn("dk-bi-lr", line.flagged && "dk-flag", line.removed && "dk-off")}
      style={{ animationDelay: `${index * 40}ms` }}
    >
      <span className="dk-bi-ll">
        <b>{line.label}</b>
        {line.basis && <span>{line.basis}</span>}
      </span>
      <span className="dk-bi-n dk-mut">
        {line.expected === null || line.expected === undefined ? "—" : money(line.expected)}
      </span>
      <span className="dk-bi-n">{money(line.billed)}</span>
      {(line.removed || line.adjusted) && canUndo && line.issueId && (
        <Button
          variant="quiet"
          size="icon-xs"
          className="absolute top-1/2 -right-6 -translate-y-1/2 text-dsk-subtle"
          title={t("Undo")}
          aria-label={t("Undo")}
          disabled={busy}
          onClick={() => line.issueId && onUndo(line.issueId)}
        >
          <CardIcon name="undo" size={12} />
        </Button>
      )}
    </div>
  );
}

function ActivityLog({ itemId, meId }: { itemId: string; meId: string | undefined }) {
  const t = useT();
  const query = useInfiniteQuery({
    queryKey: billingQueue.activity(itemId).queryKey,
    queryFn: ({ pageParam }) =>
      apiService.billingQueueService.getActivity(itemId, pageParam ?? undefined),
    initialPageParam: null as { at: number; id: string } | null,
    getNextPageParam: (page) => {
      const last = page.items.at(-1);
      return page.hasMore && last ? { at: last.at, id: last.id } : null;
    },
  });
  // Oldest first, the way the timeline reads down.
  const entries = (query.data?.pages ?? []).flatMap((page) => page.items).reverse();
  if (entries.length === 0) return null;

  const who = (entry: BillingQueueEvent) => {
    if (entry.actorType === "User") {
      return entry.actorId === meId ? t("You") : entry.actorName || t("Someone");
    }
    return entry.actorType === "Agent" ? entry.actorName || t("Agent") : t("System");
  };

  return (
    <CardSection title={t("Activity")}>
      {query.hasNextPage && (
        <Button
          variant="bare"
          size="bare"
          className="-mt-0.5 mb-2.5 ml-6 text-xs text-dsk-subtle hover:text-dsk-fg"
          disabled={query.isFetchingNextPage}
          onClick={() => void query.fetchNextPage()}
        >
          {t("Show earlier")}
        </Button>
      )}
      <CardTimeline
        entries={entries.map((entry) => ({
          id: entry.id,
          text: entry.text,
          meta: `${who(entry)} · ${cardMoment(entry.at)}`,
          mine: entry.actorType === "User" && entry.actorId === meId,
        }))}
      />
    </CardSection>
  );
}

/** Hold, with the three reasons a biller gives. */
function HoldMenu({
  busy,
  onHold,
}: {
  busy: boolean;
  onHold: (reason: BillingQueueHoldReason) => void;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        disabled={busy}
        render={
          <Button
            variant="quiet"
            className={cn("disabled:opacity-35", open && "bg-dsk-hover text-dsk-fg")}
          />
        }
      >
        <CardIcon name="pause" size={12} stroke={2.4} />
        {t("Hold")}
        <CardIcon name="down" size={10} stroke={2.4} />
      </PopoverTrigger>
      <PopoverContent
        side="top"
        align="start"
        sideOffset={6}
        className={cn(deskPopoverClass, "w-50 p-1")}
      >
        <div role="menu">
          {HOLD_REASONS.map((reason) => (
            <Button
              key={reason}
              variant="bare"
              size="bare"
              className="block w-full rounded-lg px-2.25 py-1.75 text-left text-sm text-dsk-fg2 hover:bg-dsk-hover hover:text-dsk-fg"
              role="menuitem"
              onClick={() => {
                setOpen(false);
                onHold(reason);
              }}
            >
              {holdReasonLabel(reason, t)}
            </Button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
}

/**
 * One billing queue item as a biller reviews it, after the design's billing
 * item: what it bills and to whom, the five checks that stand between it and
 * approval with a way to settle each, the charges against the rate con, the
 * shipment and its paperwork, and what has happened to it. The checks are the
 * server's; the card only words them. Its buttons are the queue's own
 * actions, taken by the person, never by the agent.
 *
 * The card shows whichever item is picked, and steps through the queue in the
 * list's own order; `onSelect` is how a step picks the next one. Until the
 * item loads, or if the person cannot read it, `fallback` shows instead; a
 * step shows it too, never the item it left, whose buttons would act on the
 * item now asked for.
 */
export function DeskBillingItem({
  itemId,
  fallback,
  onSelect,
  onBack,
  backLabel,
}: {
  itemId: string;
  fallback: ReactNode;
  onSelect?: (itemId: string) => void;
  onBack?: () => void;
  /** What the way back says; the queue, unless the item was opened from another card. */
  backLabel?: string;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const me: Biller | null = user?.id ? { id: user.id, name: user.name ?? "" } : null;
  const { allowed: canUpdate } = usePermission(Resource.BillingQueue, Operation.Update);
  const { allowed: canAssign } = usePermission(Resource.BillingQueue, Operation.Assign);
  const { allowed: canPost } = usePermission(Resource.Invoice, Operation.Post);
  const [posted, setPosted] = useState<BillingQueuePostResult | null>(null);

  const itemQuery = useQuery({ ...billingQueue.get(itemId), retry: false });
  const item = itemQuery.data;
  const { data: neighbors } = useQuery({
    ...billingQueue.neighbors(itemId),
    enabled: onSelect !== undefined,
  });

  // Every change lands on the item and on the queue's rows and counts, so
  // the table, the item and anything else reading the queue stay in step.
  const settle = useCallback(
    (next?: BillingQueueItem) => {
      if (next?.review) {
        queryClient.setQueryData(billingQueue.get(itemId).queryKey, next);
      }
      void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
      void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
    },
    [itemId, queryClient],
  );

  const assign = useApiMutation({
    mutationFn: (biller: Biller) =>
      assignBillingQueueBillerGraphQL(itemId, { billerId: biller.id }),
    resourceName: "BillingQueueItem",
    onSuccess: () => settle(),
  });
  const status = useApiMutation({
    mutationFn: (input: {
      status: "Approved" | "OnHold";
      holdReasonCode?: BillingQueueHoldReason;
    }) => apiService.billingQueueService.updateStatus(itemId, input),
    resourceName: "BillingQueueItem",
    onSuccess: () => settle(),
  });
  const release = useApiMutation({
    mutationFn: () => apiService.billingQueueService.release(itemId),
    resourceName: "BillingQueueItem",
    onSuccess: () => settle(),
  });
  const resolve = useApiMutation({
    mutationFn: ({ issueId, optionKey }: { issueId: string; optionKey: string }) =>
      apiService.billingQueueService.resolveIssue(itemId, issueId, optionKey),
    resourceName: "BillingQueueItem",
    onSuccess: (next) => settle(next),
  });
  const undo = useApiMutation({
    mutationFn: (issueId: string) => apiService.billingQueueService.undoIssue(itemId, issueId),
    resourceName: "BillingQueueItem",
    onSuccess: (next) => settle(next),
  });
  const post = useApiMutation({
    mutationFn: () => apiService.billingQueueService.post(itemId),
    resourceName: "BillingQueueItem",
    onSuccess: (result) => {
      setPosted(result);
      settle();
    },
  });
  const busy =
    assign.isPending ||
    status.isPending ||
    release.isPending ||
    resolve.isPending ||
    undo.isPending ||
    post.isPending;

  if (!item) {
    return <>{fallback}</>;
  }

  const review = item.review;
  const stage = itemStage(item.status);
  const charges = review?.charges;
  const total = charges?.billedTotal ?? item.allocatedTotalAmount ?? 0;
  const invoice = review?.invoice;
  const terms = review?.terms;
  const recipient = terms?.recipients[0] ?? "";
  const invoiceNumber = invoice ? (invoice.posted ? invoice.number : invoice.draftNumber) : "";
  const lane = laneOf(item.shipment, t);
  const shipmentPath = recordPath("shipment", item.shipmentId);
  const reason = barReason(item, t);
  const needs = review?.needsCount ?? 0;
  const postedNumber = posted?.invoiceNumber || (invoice?.posted ? invoice.number : "");
  const sentTo = posted?.sentTo ?? "";
  const step = (id: string) => {
    if (onSelect) {
      setPosted(null);
      onSelect(id);
    }
  };

  return (
    <RecordCard className={cn(stage === "posted" && "dk-posted", stage === "held" && "dk-held")}>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the queue")}
        kicker={[
          item.number,
          ...(invoiceNumber !== "" && invoiceNumber !== item.number ? [invoiceNumber] : []),
          ...(item.poNumber ? [t("PO {0}", item.poNumber)] : []),
        ]}
        stepper={
          onSelect &&
          neighbors && (
            <CardStepper
              neighbors={neighbors}
              onStep={step}
              previousLabel={t("Previous item")}
              nextLabel={t("Next item")}
            />
          )
        }
        title={item.billToCustomer?.name ?? t("No bill-to")}
        subtitle={recipient ? t("AP · {0}", recipient) : (item.billToCustomer?.code ?? "")}
        figure={
          <>
            <b key={String(total)}>{money(total)}</b>
            {terms?.paymentTerm && (
              <span>
                {terms.dueDate
                  ? t(
                      "{0} · due {1}",
                      paymentTermLabel(terms.paymentTerm, t),
                      cardDay(terms.dueDate),
                    )
                  : paymentTermLabel(terms.paymentTerm, t)}
              </span>
            )}
          </>
        }
        meta={
          <>
            <CardStatus
              key={`${item.status}:${item.holdReasonCode ?? ""}`}
              tone={STAGE_TONE[stage]}
              mark={
                stage === "held"
                  ? "pause"
                  : stage === "approved" || stage === "posted"
                    ? "check"
                    : "dot"
              }
            >
              {stageLabel(item.status, item.holdReasonCode, t)}
            </CardStatus>
            <CardFacts>
              {[
                t("Queued {0}", cardMoment(item.createdAt)),
                t(
                  "{0, plural, one {# day in queue} other {# days in queue}}",
                  daysSince(item.createdAt),
                ),
              ]}
            </CardFacts>
            {stage === "held" && canUpdate && (
              <Button
                variant="outline"
                size="xs"
                className="ml-auto"
                disabled={busy}
                onClick={() => release.mutate(undefined)}
              >
                {t("Release")}
              </Button>
            )}
          </>
        }
      />

      {stage === "review" && review && (
        <CardSection
          title={t("Before it can be approved")}
          aside={
            needs > 0 ? (
              <span className="dk-warn">{t("{0} of 5 need you", needs)}</span>
            ) : (
              <span className="dk-ok">{t("All clear")}</span>
            )
          }
        >
          <CardChecks>
            {review.checks.map((check) => (
              <CheckRow
                key={check.key}
                item={item}
                check={check}
                canAct={check.key === "biller" ? canAssign : canUpdate}
                busy={busy}
                me={me}
                onAssign={(biller) => assign.mutate(biller)}
                onResolve={(issue, optionKey) => resolve.mutate({ issueId: issue.id, optionKey })}
              />
            ))}
          </CardChecks>
        </CardSection>
      )}

      {charges && charges.lines.length > 0 && (
        <CardSection
          title={t("Charges")}
          aside={
            charges.hasRateCon ? (
              <span>{t("vs rate con {0}", money(charges.expectedTotal))}</span>
            ) : undefined
          }
        >
          <div className="dk-bi-led">
            <div className="dk-bi-lh">
              <span>{t("Charge")}</span>
              <span>{t("Rate con")}</span>
              <span>{t("Billed")}</span>
            </div>
            {charges.lines.map((line, index) => (
              <LedgerRow
                key={line.key}
                line={line}
                index={index}
                canUndo={stage === "review" && canUpdate}
                busy={busy}
                onUndo={(issueId) => undo.mutate(issueId)}
              />
            ))}
            <div className="dk-bi-lt">
              <span>{t("Total")}</span>
              <span className="dk-bi-n dk-mut">
                {charges.hasRateCon ? money(charges.expectedTotal) : "—"}
              </span>
              <span className="dk-bi-n">{money(charges.billedTotal)}</span>
            </div>
            {charges.hasRateCon && Math.abs(Number(charges.difference)) >= 0.005 && (
              <div className="dk-bi-diff">
                {Number(charges.difference) > 0
                  ? t("{0} over the rate con", money(Math.abs(Number(charges.difference))))
                  : t("{0} under the rate con", money(Math.abs(Number(charges.difference))))}
              </div>
            )}
          </div>
        </CardSection>
      )}

      {lane && (
        <CardSection
          title={t("Shipment")}
          aside={<CardLink to={shipmentPath}>{item.shipment?.proNumber || t("Open")}</CardLink>}
        >
          <div className="dk-bi-lane">
            <span className="dk-bi-lt-i">
              <CardIcon name="truck" size={14} />
            </span>
            <div>
              <b>
                {lane.from} <i>→</i> {lane.to}
              </b>
              {lane.detail !== "" && <span>{lane.detail}</span>}
            </div>
          </div>
        </CardSection>
      )}

      <ActivityLog itemId={itemId} meId={user?.id} />

      {/* The documents a biller bills from are the shipment's, where they are filed. */}
      {item.shipmentId !== "" && (
        <CardDocuments
          resourceType="shipment"
          resourceId={item.shipmentId}
          manageTo={shipmentPath}
        />
      )}

      {canUpdate &&
        (stage === "review" || stage === "approved" || stage === "posted" || stage === "held") && (
          <CardBar>
            {stage === "posted" ? (
              <div className="dk-bi-done">
                <CardIcon name="check" size={13} stroke={2.6} />
                {sentTo
                  ? t("Posted as {0} · sent to {1}", postedNumber, sentTo)
                  : t("Posted as {0}", postedNumber || item.number)}
              </div>
            ) : stage === "held" ? (
              <CardBarHint>{reason}</CardBarHint>
            ) : (
              <>
                {stage === "review" && (
                  <HoldMenu
                    busy={busy}
                    onHold={(holdReasonCode) => status.mutate({ status: "OnHold", holdReasonCode })}
                  />
                )}
                <CardBarHint>{reason}</CardBarHint>
                {stage === "review" ? (
                  <Button
                    className="disabled:opacity-35"
                    disabled={busy || !review?.ready}
                    onClick={() => status.mutate({ status: "Approved" })}
                  >
                    {t("Approve")}
                  </Button>
                ) : canPost ? (
                  <Button
                    className="disabled:opacity-35"
                    disabled={busy}
                    onClick={() => post.mutate(undefined)}
                  >
                    {t("Post {0}", money(total))}
                  </Button>
                ) : (
                  <CardBarHint>{t("Someone who can post invoices posts this one.")}</CardBarHint>
                )}
              </>
            )}
          </CardBar>
        )}
    </RecordCard>
  );
}
