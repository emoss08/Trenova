import { fetchGraphQLData } from "@/hooks/data-table/use-data-table-query";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { recordPath } from "@/config/record-links";
import { auditLogTableGraphQLConfig } from "@/lib/graphql/audit-log-table";
import {
  assignBillingQueueBillerGraphQL,
  updateBillingQueueStatusGraphQL,
} from "@/lib/graphql/billing-queue";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { formatDisplayValue, statusPhase } from "@/components/assistant/readable-values";
import { useOutsideDismiss } from "../use-outside-dismiss";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type {
  BillingQueueItem,
  BillingQueueUpdateStatusInput,
} from "@trenova/shared/types/billing-queue";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo, useRef, useState, type ReactNode } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import { approveBlocker, billingChecks, money, type BillingCheck } from "./billing-item-checks";
import { ArtIcon } from "./desk-art-kinds";

const PHASE_PILL: Record<string, string> = {
  failed: "dk-warn",
  attention: "dk-warn",
  awaiting: "dk-blue",
  active: "dk-blue",
  queued: "dk-blue",
  complete: "dk-green",
  closed: "dk-ink",
};

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

/** The route as one line: where it was picked up, where it went, and how it ended. */
function routeOf(shipment: Shipment | undefined, t: TranslateFn) {
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
  const arrived = to?.actualArrival;
  return {
    lane: `${placeOf(from)} → ${placeOf(to)}`,
    detail: [
      arrived ? t("Delivered {0}", moment(arrived)) : "",
      driverName,
      miles > 0 ? t("{0} mi", Math.round(miles).toLocaleString()) : "",
    ]
      .filter((part) => part !== "")
      .join(" · "),
  };
}

function CheckRow({
  check,
  onAssignMe,
  someoneElse,
  busy,
}: {
  check: BillingCheck;
  onAssignMe: (() => void) | null;
  someoneElse: string;
  busy: boolean;
}) {
  const t = useT();
  return (
    <div className={cn("dk-bq-chk", `dk-${check.state}`)}>
      <span className="dk-bq-ico">
        <ArtIcon
          name={check.state === "ok" ? "check" : check.state === "warn" ? "warn" : "x"}
          size={check.state === "warn" ? 13 : 11}
          stroke={2.4}
        />
      </span>
      <div className="dk-bq-chk-b">
        <div className="dk-bq-chk-h">
          <b>{check.title}</b>
          {check.state === "ok" && <span>{check.detail}</span>}
        </div>
        {check.state !== "ok" && <div className="dk-bq-find">{check.detail}</div>}
        {check.note && check.state !== "ok" && <p className="dk-bq-note">{check.note}</p>}
        {check.action === "assign" && (
          <div className="dk-bq-acts">
            {onAssignMe && (
              <button
                type="button"
                className="dk-bq-btn dk-light"
                disabled={busy}
                onClick={onAssignMe}
              >
                {t("Assign to me")}
              </button>
            )}
            <Link className="dk-bq-btn" to={someoneElse}>
              {t("Someone else…")}
            </Link>
          </div>
        )}
      </div>
    </div>
  );
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
    <section className="dk-bq-sec">
      <div className="dk-bq-sec-h">
        <b>{title}</b>
        {aside}
      </div>
      {children}
    </section>
  );
}

/** Put on hold, or taken off it: the one thing the Hold menu does for the item's status. */
function HoldMenu({
  item,
  busy,
  onStatus,
}: {
  item: BillingQueueItem;
  busy: boolean;
  onStatus: (input: BillingQueueUpdateStatusInput) => void;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const close = useCallback(() => setOpen(false), []);
  useOutsideDismiss(rootRef, open, close);
  const held = item.status === "OnHold";
  const canHold = ["ReadyForReview", "InReview"].includes(item.status);
  if (!held && !canHold) return <span />;

  return (
    <div className="dk-bq-hold" ref={rootRef}>
      <button
        type="button"
        className="dk-bq-hold-b"
        disabled={busy}
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
      >
        <ArtIcon name="pause" size={12} stroke={2.2} />
        {held ? t("On hold") : t("Hold")}
        <ArtIcon name="down" size={10} stroke={2.4} />
      </button>
      {open && (
        <div className="dk-bq-menu" role="menu">
          {held ? (
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                close();
                onStatus({ status: "ReadyForReview" });
              }}
            >
              <b>{t("Take off hold")}</b>
              <span>{t("Back to ready for review")}</span>
            </button>
          ) : (
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                close();
                onStatus({ status: "OnHold" });
              }}
            >
              <b>{t("Put on hold")}</b>
              <span>{t("Keep it out of review until you take it off")}</span>
            </button>
          )}
        </div>
      )}
    </div>
  );
}

/**
 * One billing item as a biller reviews it: what it bills and to whom, what
 * stands between it and approval, the charges, the shipment and its paperwork,
 * and what has happened to it. It is read live with the person's own access,
 * and its buttons are the billing queue's own actions, taken by the person,
 * not by the agent. Until the item loads, or if the person cannot read it,
 * `fallback` (the card the agent's lookup produced) shows instead.
 */
export function DeskBillingItem({ itemId, fallback }: { itemId: string; fallback: ReactNode }) {
  const t = useT();
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const { allowed: canUpdate } = usePermission(Resource.BillingQueue, Operation.Update);
  const { allowed: canAssign } = usePermission(Resource.BillingQueue, Operation.Assign);
  const { allowed: canReadAudit } = usePermission(Resource.AuditLog, Operation.Read);

  const itemQuery = useQuery({ ...queries.billingQueue.get(itemId), retry: false });
  const item = itemQuery.data;
  const shipmentId = item?.shipmentId ?? "";
  const { data: readiness } = useQuery({
    ...queries.shipment.billingReadiness(shipmentId),
    enabled: shipmentId !== "",
  });
  const { data: documents = [] } = useQuery({
    queryKey: ["documents", "shipment", shipmentId, "includeDocumentType"],
    queryFn: () =>
      apiService.documentService.getByResource("shipment", shipmentId, undefined, {
        includeDocumentType: "true",
      }),
    enabled: shipmentId !== "",
  });
  const { data: activity } = useQuery({
    queryKey: [...queries.audit.history(itemId).queryKey, "desk"],
    queryFn: ({ signal }) =>
      fetchGraphQLData(
        8,
        auditLogTableGraphQLConfig,
        {
          fieldFilters: [{ field: "resourceId", operator: "eq", value: itemId }],
          sort: [{ field: "timestamp", direction: "desc" }],
        },
        { signal },
      ),
    enabled: canReadAudit,
  });

  const refresh = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
    void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
    void queryClient.invalidateQueries({ queryKey: ["billing-readiness", shipmentId] });
    void queryClient.invalidateQueries({ queryKey: queries.audit.history(itemId).queryKey });
  }, [itemId, queryClient, shipmentId]);

  const status = useApiMutation({
    mutationFn: (input: BillingQueueUpdateStatusInput) =>
      updateBillingQueueStatusGraphQL(itemId, input),
    resourceName: "BillingQueueItem",
    onSuccess: (_, input) => {
      refresh();
      toast.success(
        input.status === "Approved"
          ? t("Approved")
          : input.status === "OnHold"
            ? t("Put on hold")
            : t("Taken off hold"),
      );
    },
  });
  const assign = useApiMutation({
    mutationFn: (billerId: string) => assignBillingQueueBillerGraphQL(itemId, { billerId }),
    resourceName: "BillingQueueItem",
    onSuccess: () => {
      refresh();
      toast.success(t("Assigned to you"));
    },
  });
  const busy = status.isPending || assign.isPending;

  const checks = useMemo(
    () => (item ? billingChecks(item, readiness, t) : []),
    [item, readiness, t],
  );

  if (!item) {
    return <>{fallback}</>;
  }

  const open = checks.filter((check) => check.state !== "ok").length;
  const blocker = approveBlocker(item, checks, readiness, t);
  const phase = statusPhase(item.status);
  const share = item.payerShare;
  const route = routeOf(item.shipment, t);
  const queuePath = recordPath("billing_queue_item", item.id);
  const shipmentPath = recordPath("shipment", item.shipmentId);
  const current = documents.filter((document) => document.isCurrentVersion !== false);
  const entries = [...(activity?.results ?? [])].reverse();
  const ids = [item.number, item.shipment?.proNumber, item.shipment?.bol]
    .filter((part): part is string => Boolean(part))
    .join(" · ");
  const settled = ["Approved", "Posted", "Canceled"].includes(item.status);

  return (
    <div className="dk-bq">
      <div className="dk-bq-scroll">
        <div className="dk-bq-head">
          {ids !== "" && <div className="dk-bq-ids">{ids}</div>}
          <div className="dk-bq-top">
            <div className="min-w-0">
              <div className="dk-bq-name">{item.billToCustomer?.name ?? t("No bill-to")}</div>
              <div className="dk-bq-sub">
                {[item.billToCustomer?.code, formatDisplayValue("status", item.billType, t)]
                  .filter(Boolean)
                  .join(" · ")}
              </div>
            </div>
            <div className="dk-bq-amt">
              <b>{money(item.allocatedTotalAmount ?? share?.totalAmount)}</b>
              {share?.isSplit && <span>{t("Their share of {0}", money(share.shipmentTotal))}</span>}
            </div>
          </div>
          <div className="dk-bq-meta">
            <span className={cn("dk-ax-pill dk-lg", phase ? PHASE_PILL[phase] : "")}>
              <i />
              {formatDisplayValue("status", item.status, t)}
            </span>
            <span>
              {t("Queued")} <em>{moment(item.createdAt)}</em>
            </span>
            {!settled && daysSince(item.createdAt) > 0 && (
              <span>
                ·{" "}
                {t(
                  "{0, plural, one {# day in queue} other {# days in queue}}",
                  daysSince(item.createdAt),
                )}
              </span>
            )}
          </div>
        </div>

        {!settled && checks.length > 0 && (
          <Section
            title={t("Before it can be approved")}
            aside={
              open > 0 ? (
                <span className="dk-bq-need">{t("{0} of {1} need you", open, checks.length)}</span>
              ) : (
                <span className="dk-bq-clear">{t("All clear")}</span>
              )
            }
          >
            <div className="dk-bq-checks">
              {checks.map((check) => (
                <CheckRow
                  key={check.key}
                  check={check}
                  busy={busy}
                  someoneElse={queuePath}
                  onAssignMe={canAssign && user?.id ? () => assign.mutate(user.id as string) : null}
                />
              ))}
            </div>
          </Section>
        )}

        {share && share.lines.length > 0 && (
          <Section
            title={t("Charges")}
            aside={
              share.isSplit ? (
                <span>{t("of {0} on the shipment", money(share.shipmentTotal))}</span>
              ) : undefined
            }
          >
            <table className="dk-bq-charges">
              <thead>
                <tr>
                  <th>{t("Charge")}</th>
                  <th className="dk-ar">{t("Billed")}</th>
                </tr>
              </thead>
              <tbody>
                {share.lines.map((line, index) => (
                  <tr key={line.additionalChargeId ?? `${line.kind}-${index}`}>
                    <td>
                      <b>{line.description || formatDisplayValue("status", line.kind, t)}</b>
                      <span>
                        {[
                          line.description &&
                          line.description.toLowerCase() !== line.kind.toLowerCase()
                            ? formatDisplayValue("status", line.kind, t)
                            : "",
                          line.partial ? t("{0}% share", Number(line.percent)) : "",
                        ]
                          .filter(Boolean)
                          .join(" · ")}
                      </span>
                    </td>
                    <td className="dk-ar">
                      <span className="dk-ax-num">{money(line.amount)}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
              <tfoot>
                <tr>
                  <td>{t("Total")}</td>
                  <td className="dk-ar">
                    <b className="dk-ax-num">{money(share.totalAmount)}</b>
                  </td>
                </tr>
              </tfoot>
            </table>
          </Section>
        )}

        {(route || current.length > 0) && (
          <Section
            title={t("Shipment")}
            aside={
              <Link className="dk-bq-link" to={shipmentPath}>
                {item.shipment?.proNumber || t("Open")}
                <ArtIcon name="ext" size={11} />
              </Link>
            }
          >
            {route && (
              <div className="dk-bq-route">
                <span className="dk-bq-route-i">
                  <ArtIcon name="truck" size={15} />
                </span>
                <div>
                  <b>{route.lane}</b>
                  {route.detail !== "" && <span>{route.detail}</span>}
                </div>
              </div>
            )}
            {current.length > 0 && (
              <div className="dk-bq-docs">
                {current.slice(0, 8).map((document) => (
                  <div key={document.id} className="dk-bq-doc">
                    <ArtIcon name="doc" size={13} />
                    <div>
                      <b>{document.documentType?.name || document.originalName}</b>
                      <span>{document.description || document.originalName}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </Section>
        )}

        {entries.length > 0 && (
          <Section title={t("Activity")}>
            <ol className="dk-bq-log">
              {entries.map((entry) => (
                <li key={String(entry.id)}>
                  <i />
                  <div>
                    <b>{String(entry.comment || entry.operation || "")}</b>
                    <span>
                      {[
                        (entry.user as { name?: string } | null)?.name ?? t("System"),
                        moment(Number(entry.timestamp)),
                      ]
                        .filter(Boolean)
                        .join(" · ")}
                    </span>
                  </div>
                </li>
              ))}
            </ol>
          </Section>
        )}
      </div>

      {!settled && canUpdate && (
        <div className="dk-bq-foot">
          <HoldMenu item={item} busy={busy} onStatus={(input) => status.mutate(input)} />
          <span className="flex-1" />
          {blocker !== "" && <span className="dk-bq-why">{blocker}</span>}
          <button
            type="button"
            className="dk-bq-approve"
            disabled={busy || blocker !== ""}
            onClick={() => status.mutate({ status: "Approved" })}
          >
            {t("Approve")}
          </button>
        </div>
      )}
    </div>
  );
}
