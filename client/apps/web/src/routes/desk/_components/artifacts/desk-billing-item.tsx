import { recordPath } from "@/config/record-links";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { assignBillingQueueBillerGraphQL } from "@/lib/graphql/billing-queue";
import { fetchGraphQLSelectOptions } from "@/lib/graphql/select-options";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import {
  keepPreviousData,
  useInfiniteQuery,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
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
import { useCallback, useRef, useState, type ReactNode } from "react";
import { Link } from "react-router";
import { useOutsideDismiss } from "../use-outside-dismiss";
import {
  HOLD_REASONS,
  barReason,
  checkDetail,
  checkTitle,
  holdReasonLabel,
  initials,
  issueOf,
  itemStage,
  money,
  paymentTermLabel,
  stageLabel,
} from "./billing-item-checks";

const BI_PATHS = {
  check: <path d="M5 12.5l4.5 4.5L19 7" />,
  warn: (
    <>
      <path d="M12 4l9 15.5H3z" />
      <path d="M12 10v4M12 17v.01" />
    </>
  ),
  x: <path d="M7 7l10 10M17 7L7 17" />,
  file: (
    <>
      <path d="M6 3.5h8l4 4v13H6z" />
      <path d="M14 3.5v4h4" />
    </>
  ),
  pause: <path d="M9 6v12M15 6v12" />,
  truck: (
    <>
      <path d="M3.5 6.5h10v9h-10zM13.5 9.5h4l3 3v3h-7" />
      <circle cx="7" cy="17" r="1.6" />
      <circle cx="17" cy="17" r="1.6" />
    </>
  ),
  ext: (
    <path d="M14 4.5h5.5V10M19.5 4.5L11 13M17 14v4.5a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V8a1 1 0 0 1 1-1h4.5" />
  ),
  down: <path d="M6 9l6 6 6-6" />,
  up: <path d="M6 15l6-6 6 6" />,
  undo: <path d="M9 7L4.5 11.5 9 16M5 11.5h9a5 5 0 0 1 0 10h-2" />,
  back: <path d="M15 6l-6 6 6 6" />,
} as const;

/** The billing item's own icons, drawn as the design draws them. */
export function BiIcon({
  name,
  size = 13,
  stroke = 1.9,
}: {
  name: keyof typeof BI_PATHS;
  size?: number;
  stroke?: number;
}) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={stroke}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {BI_PATHS[name]}
    </svg>
  );
}

function shortDate(seconds: number | null | undefined): string {
  if (!seconds) return "";
  return new Date(seconds * 1000).toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

function moment(seconds: number | null | undefined): string {
  if (!seconds) return "";
  return new Date(seconds * 1000).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

function daysSince(seconds: number): number {
  return Math.max(0, Math.floor((Date.now() / 1000 - seconds) / 86_400));
}

type Stop = NonNullable<Shipment["moves"]>[number]["stops"][number];

function placeOf(stop: Stop | undefined): string {
  const location = stop?.location;
  if (!location) return "";
  const state = (location as { state?: { abbreviation?: string } | null }).state?.abbreviation;
  return [location.city, state].filter(Boolean).join(", ") || location.name;
}

/** The lane as the design draws it: from and to, then delivered · driver · miles. */
function laneOf(shipment: Shipment | undefined, t: TranslateFn) {
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
      to?.actualArrival ? t("Delivered {0}", shortDate(to.actualArrival)) : "",
      driverName,
      miles > 0 ? t("{0} mi", Math.round(miles).toLocaleString()) : "",
    ]
      .filter((part) => part !== "")
      .join(" · "),
  };
}

function Section({
  title,
  aside,
  children,
}: {
  title: string;
  aside?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="dk-bi-sec">
      <div className="dk-bi-sh">
        <h3>{title}</h3>
        {aside}
      </div>
      {children}
    </section>
  );
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
        <button key={biller.id} type="button" disabled={busy} onClick={() => onPick(biller)}>
          <span className="dk-bi-av">{initials(biller.name)}</span>
          {biller.name}
          {biller.id === me?.id && <em>{t("you")}</em>}
        </button>
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
    <div className={cn("dk-bi-ck", `dk-${check.state === "fail" ? "bad" : check.state}`)}>
      <span className="dk-bi-ci">
        <BiIcon
          name={ok ? "check" : check.state === "warn" ? "warn" : "x"}
          size={check.state === "warn" ? 12 : 11}
          stroke={check.state === "warn" ? 2 : 2.8}
        />
      </span>
      <div className="dk-bi-cb">
        <b>{checkTitle(check.key, t)}</b>
        <span>{issue && !ok ? issue.summary : checkDetail(check, t)}</span>
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
                <button
                  type="button"
                  className="dk-pri"
                  disabled={busy}
                  onClick={() => onAssign(me)}
                >
                  {t("Assign to me")}
                </button>
              )}
              <button type="button" disabled={busy} onClick={() => setPicking(true)}>
                {t("Someone else…")}
              </button>
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
                    <button
                      key={option.key}
                      type="button"
                      className={cn(index === 0 && !asked && "dk-pri")}
                      disabled={busy || Boolean(asked)}
                      onClick={() => onResolve(issue, option.key)}
                    >
                      {asked ? t("Asked {0}", moment(issue.requestedAt)) : option.label}
                    </button>
                  );
                })}
              </div>
            )}
          </>
        )}
      </div>
    </div>
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
        <button
          type="button"
          className="dk-bi-undo"
          title={t("Undo")}
          aria-label={t("Undo")}
          disabled={busy}
          onClick={() => line.issueId && onUndo(line.issueId)}
        >
          <BiIcon name="undo" size={12} />
        </button>
      )}
    </div>
  );
}

function ActivityLog({ itemId, meId }: { itemId: string; meId: string | undefined }) {
  const t = useT();
  const query = useInfiniteQuery({
    queryKey: queries.billingQueue.activity(itemId).queryKey,
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
    <Section title={t("Activity")}>
      {query.hasNextPage && (
        <button
          type="button"
          className="dk-bi-more"
          disabled={query.isFetchingNextPage}
          onClick={() => void query.fetchNextPage()}
        >
          {t("Show earlier")}
        </button>
      )}
      <ol className="dk-bi-log">
        {entries.map((entry) => (
          <li
            key={entry.id}
            className={cn(entry.actorType === "User" && entry.actorId === meId && "dk-you")}
          >
            <i />
            <span>{entry.text}</span>
            <em>
              {who(entry)} · {moment(entry.at)}
            </em>
          </li>
        ))}
      </ol>
    </Section>
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
  const rootRef = useRef<HTMLDivElement>(null);
  const close = useCallback(() => setOpen(false), []);
  useOutsideDismiss(rootRef, open, close);

  return (
    <div className="dk-bi-m" ref={rootRef}>
      <button
        type="button"
        className={cn("dk-ax-btn dk-ghost", open && "dk-on")}
        disabled={busy}
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
      >
        <BiIcon name="pause" size={12} stroke={2.4} />
        {t("Hold")}
        <BiIcon name="down" size={10} stroke={2.4} />
      </button>
      {open && (
        <div className="dk-bi-mp" role="menu">
          {HOLD_REASONS.map((reason) => (
            <button
              key={reason}
              type="button"
              role="menuitem"
              onClick={() => {
                close();
                onHold(reason);
              }}
            >
              {holdReasonLabel(reason, t)}
            </button>
          ))}
        </div>
      )}
    </div>
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
 * item loads, or if the person cannot read it, `fallback` shows instead.
 */
export function DeskBillingItem({
  itemId,
  fallback,
  onSelect,
  onBack,
}: {
  itemId: string;
  fallback: ReactNode;
  onSelect?: (itemId: string) => void;
  onBack?: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const me: Biller | null = user?.id ? { id: user.id, name: user.name ?? "" } : null;
  const { allowed: canUpdate } = usePermission(Resource.BillingQueue, Operation.Update);
  const { allowed: canAssign } = usePermission(Resource.BillingQueue, Operation.Assign);
  const [posted, setPosted] = useState<BillingQueuePostResult | null>(null);

  const itemQuery = useQuery({
    ...queries.billingQueue.get(itemId),
    retry: false,
    placeholderData: keepPreviousData,
  });
  const item = itemQuery.data;
  const { data: neighbors } = useQuery({
    ...queries.billingQueue.neighbors(itemId),
    enabled: onSelect !== undefined,
  });

  // Every change lands on the item and on the queue's rows and counts, so
  // the table, the item and anything else reading the queue stay in step.
  const settle = useCallback(
    (next?: BillingQueueItem) => {
      if (next?.review) {
        queryClient.setQueryData(queries.billingQueue.get(itemId).queryKey, next);
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
  const step = (id: string | null | undefined) => {
    if (id && onSelect) {
      setPosted(null);
      onSelect(id);
    }
  };

  return (
    <div className={cn("dk-bi", stage === "posted" && "dk-posted", stage === "held" && "dk-held")}>
      <div className="dk-bi-head">
        <div className="dk-bi-kick">
          {onBack && (
            <button
              type="button"
              className="dk-bi-back"
              onClick={onBack}
              title={t("Back to the queue")}
            >
              <BiIcon name="back" size={12} stroke={2.2} />
            </button>
          )}
          <span>{item.number}</span>
          {invoiceNumber !== "" && invoiceNumber !== item.number && (
            <>
              <i />
              <span>{invoiceNumber}</span>
            </>
          )}
          {item.poNumber && (
            <>
              <i />
              <span>{t("PO {0}", item.poNumber)}</span>
            </>
          )}
          {onSelect && neighbors && (
            <span className="dk-bi-nav">
              <button
                type="button"
                title={t("Previous item")}
                aria-label={t("Previous item")}
                disabled={!neighbors.prevId}
                onClick={() => step(neighbors.prevId)}
              >
                <BiIcon name="up" size={12} stroke={2.2} />
              </button>
              {neighbors.position > 0 && (
                <em>{t("{0} of {1}", neighbors.position, neighbors.total)}</em>
              )}
              <button
                type="button"
                title={t("Next item")}
                aria-label={t("Next item")}
                disabled={!neighbors.nextId}
                onClick={() => step(neighbors.nextId)}
              >
                <BiIcon name="down" size={12} stroke={2.2} />
              </button>
            </span>
          )}
        </div>
        <div className="dk-bi-hrow">
          <div className="dk-bi-who">
            <h2>{item.billToCustomer?.name ?? t("No bill-to")}</h2>
            <span>{recipient ? t("AP · {0}", recipient) : (item.billToCustomer?.code ?? "")}</span>
          </div>
          <div className="dk-bi-amt">
            <b key={String(total)}>{money(total)}</b>
            {terms?.paymentTerm && (
              <span>
                {terms.dueDate
                  ? t(
                      "{0} · due {1}",
                      paymentTermLabel(terms.paymentTerm, t),
                      shortDate(terms.dueDate),
                    )
                  : paymentTermLabel(terms.paymentTerm, t)}
              </span>
            )}
          </div>
        </div>
        <div className="dk-bi-meta">
          <span
            key={`${item.status}:${item.holdReasonCode ?? ""}`}
            className={cn(
              "dk-bi-st",
              stage === "approved" && "dk-s1",
              stage === "posted" && "dk-s2",
              stage === "held" && "dk-hold",
            )}
          >
            {stage === "held" ? (
              <BiIcon name="pause" size={11} stroke={2.6} />
            ) : stage === "approved" || stage === "posted" ? (
              <BiIcon name="check" size={11} stroke={2.8} />
            ) : (
              <i />
            )}
            {stageLabel(item.status, item.holdReasonCode, t)}
          </span>
          <span className="dk-bi-q">
            <span>{t("Queued {0}", moment(item.createdAt))}</span>
            <b aria-hidden="true">·</b>
            <span>
              {t(
                "{0, plural, one {# day in queue} other {# days in queue}}",
                daysSince(item.createdAt),
              )}
            </span>
          </span>
          {stage === "held" && canUpdate && (
            <button
              type="button"
              className="dk-bi-rel"
              disabled={busy}
              onClick={() => release.mutate(undefined)}
            >
              {t("Release")}
            </button>
          )}
        </div>
      </div>

      {stage === "review" && review && (
        <Section
          title={t("Before it can be approved")}
          aside={
            needs > 0 ? (
              <span className="dk-warn">{t("{0} of 5 need you", needs)}</span>
            ) : (
              <span className="dk-ok">{t("All clear")}</span>
            )
          }
        >
          <div className="dk-bi-checks">
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
          </div>
        </Section>
      )}

      {charges && charges.lines.length > 0 && (
        <Section
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
        </Section>
      )}

      {(lane || (review?.documents.length ?? 0) > 0) && (
        <Section
          title={t("Shipment")}
          aside={
            <Link className="dk-bi-lnk" to={shipmentPath}>
              {item.shipment?.proNumber || t("Open")}
              <BiIcon name="ext" size={11} />
            </Link>
          }
        >
          {lane && (
            <div className="dk-bi-lane">
              <span className="dk-bi-lt-i">
                <BiIcon name="truck" size={14} />
              </span>
              <div>
                <b>
                  {lane.from} <i>→</i> {lane.to}
                </b>
                {lane.detail !== "" && <span>{lane.detail}</span>}
              </div>
            </div>
          )}
          {review && review.documents.length > 0 && (
            <div className="dk-bi-docs">
              {review.documents.map((doc) => (
                <Link
                  key={doc.documentId ?? `${doc.code}-${doc.name}`}
                  to={shipmentPath}
                  className={cn(doc.state !== "ok" && "dk-miss")}
                >
                  <BiIcon name="file" size={13} />
                  <span>
                    <b>{doc.name || doc.code}</b>
                    <em>
                      {doc.state === "missing"
                        ? t("Missing")
                        : doc.state === "unsigned"
                          ? t("Unsigned")
                          : doc.signed
                            ? t("Signed")
                            : doc.fileName}
                    </em>
                  </span>
                </Link>
              ))}
            </div>
          )}
        </Section>
      )}

      <ActivityLog itemId={itemId} meId={user?.id} />

      {canUpdate &&
        (stage === "review" || stage === "approved" || stage === "posted" || stage === "held") && (
          <div className="dk-bi-bar">
            {stage === "posted" ? (
              <div className="dk-bi-done">
                <BiIcon name="check" size={13} stroke={2.6} />
                {sentTo
                  ? t("Posted as {0} · sent to {1}", postedNumber, sentTo)
                  : t("Posted as {0}", postedNumber || item.number)}
              </div>
            ) : stage === "held" ? (
              <span className="dk-bi-bh">{reason}</span>
            ) : (
              <>
                {stage === "review" && (
                  <HoldMenu
                    busy={busy}
                    onHold={(holdReasonCode) => status.mutate({ status: "OnHold", holdReasonCode })}
                  />
                )}
                <span className="dk-bi-bh">{reason}</span>
                {stage === "review" ? (
                  <button
                    type="button"
                    className="dk-ax-btn dk-ink"
                    disabled={busy || !review?.ready}
                    onClick={() => status.mutate({ status: "Approved" })}
                  >
                    {t("Approve")}
                  </button>
                ) : (
                  <button
                    type="button"
                    className="dk-ax-btn dk-ink"
                    disabled={busy}
                    onClick={() => post.mutate(undefined)}
                  >
                    {t("Post {0}", money(total))}
                  </button>
                )}
              </>
            )}
          </div>
        )}
    </div>
  );
}
