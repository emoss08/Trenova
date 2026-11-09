import { isRecordEntityType, recordPath } from "@/config/record-links";
import { usePermission } from "@/hooks/use-permission";
import { deskAttention } from "@/lib/queries/desk-attention";
import { useQuery } from "@tanstack/react-query";
import { ArrowUpRightIcon } from "@trenova/shared/components/icons";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { getStartOfDay, resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router";
import { InPlaceRecord, recordCardFor, usePreloadRecordCard } from "../artifacts/cards/registry";
import { casePath } from "../attention/attention-record";
import { lensLabel } from "../attention/lens";
import { QueueRow } from "../queue/queue-list";
import { WatchtowerItemBar } from "../watchtower/item-bar";
import { WatchtowerItemCard } from "../watchtower/item-card";
import { WatchtowerKindGlyph } from "../watchtower/kind-glyph";
import { formatCountdown, isHot, minutesLeft, showsCountdown } from "../watchtower/lanes";
import { severityClass } from "../watchtower/severity";
import { useWatchtowerActions } from "../watchtower/use-watchtower-actions";
import { CaseNova } from "./case-nova";

type Side = "open" | "nova";

/**
 * A record's case: the record itself, drawn by its own card, and beside it
 * everything open about it (its Watchtower conditions, the messages about it,
 * the changes waiting on a person) or Nova, the conversation about it. The
 * card is the record's own; the signals are the Watchtower's items with the
 * Watchtower's card and keys; Nova is the Desk's thread. Nothing here is a
 * copy of a page that exists elsewhere.
 */
export function DeskRecordCase({ kind, id }: { kind: string; id: string }) {
  const t = useT();
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const now = useNow("minute");
  const [since] = useState(() => getStartOfDay(new Date(), timezone));
  const { allowed: canWatch } = usePermission(Resource.Watchtower, Operation.Read);
  const { allowed: canAct } = usePermission(Resource.Watchtower, Operation.Update);
  const entry = recordCardFor(kind);
  usePreloadRecordCard(entry);

  const attention = useQuery({ ...deskAttention.list({ recordId: id }), enabled: canWatch });
  const group = attention.data?.groups[0] ?? null;
  const items = group?.items ?? [];
  const record = group?.record ?? null;
  const label = record?.readable && record.label !== "" ? record.label : id;

  const [side, setSide] = useState<Side | null>(null);
  const shown: Side = side ?? (items.length > 0 ? "open" : "nova");
  const actions = useWatchtowerActions({
    items,
    now,
    timezone,
    since,
    keysEnabled: canAct && shown === "open",
  });
  const { current, select } = actions;

  // ⌘J asks Nova here: the margin turns to the conversation about the record.
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (!(event.metaKey || event.ctrlKey) || event.altKey || event.key.toLowerCase() !== "j") {
        return;
      }
      event.preventDefault();
      setSide("nova");
      requestAnimationFrame(() =>
        document.querySelector<HTMLTextAreaElement>(".dk-rc-side textarea")?.focus(),
      );
    };
    window.addEventListener("keydown", onKey, { capture: true });
    return () => window.removeEventListener("keydown", onKey, { capture: true });
  }, []);

  const pageLink = isRecordEntityType(kind) ? recordPath(kind, id) : null;

  return (
    <div className="dk-rc">
      <section className="dk-rc-main" aria-label={t("The record")}>
        {entry ? (
          <InPlaceRecord
            entry={entry}
            recordId={id}
            order={[id]}
            onSelect={() => undefined}
            onBack={() => void navigate("/desk/needs")}
            backLabel={t("Needs you")}
          />
        ) : (
          <div className="dk-rc-plain">
            <b>{label}</b>
            {pageLink && (
              <Link className="dk-at-link ui-focus-ring" to={pageLink}>
                {t("Open the record")}
                <ArrowUpRightIcon aria-hidden className="size-3" />
              </Link>
            )}
          </div>
        )}
      </section>

      <aside className="dk-rc-side" aria-label={t("Beside the record")}>
        <div className="dk-rc-tabs" role="tablist" aria-label={t("Beside the record")}>
          <button
            type="button"
            role="tab"
            aria-selected={shown === "open"}
            className={cn("dk-at-chip ui-focus-ring", shown === "open" && "dk-on")}
            onClick={() => setSide("open")}
          >
            {t("Open about this")}
            <i>{items.length}</i>
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={shown === "nova"}
            className={cn("dk-at-chip ui-focus-ring", shown === "nova" && "dk-on")}
            onClick={() => setSide("nova")}
            title={t("Press {0}", "⌘J")}
          >
            {t("Ask Nova")}
          </button>
        </div>

        {shown === "nova" ? (
          <CaseNova kind={kind} id={id} label={label} path={casePath(kind, id)} />
        ) : attention.isPending && canWatch ? (
          <div className="dk-td-sk dk-rc-pad" aria-busy="true" aria-label={t("Reading what is open")}>
            <i className="dk-sk" />
            <i className="dk-sk" />
          </div>
        ) : items.length === 0 ? (
          <p className="dk-td-quiet dk-rc-pad">
            {t("Nothing is open about this record. Ask Nova if something looks wrong.")}
          </p>
        ) : (
          <div className="dk-rc-open">
            <div className="dk-rc-list" role="list" aria-label={t("Open about this")}>
              {items.map((item) => {
                const minutes = minutesLeft(item.dueAt, now);
                return (
                  <div role="listitem" key={item.id}>
                    <QueueRow
                      selected={current?.id === item.id}
                      onSelect={() => select(item.id)}
                      leading={
                        <span className={cn("dk-w3-ic", severityClass(item.severity))}>
                          <WatchtowerKindGlyph kind={item.sourceKind} />
                        </span>
                      }
                      title={item.title}
                      subtitle={lensLabel(t, item.deskLens)}
                      trailing={
                        showsCountdown(minutes) ? (
                          <span className={cn("dk-w3-due", isHot(minutes) && "dk-hot")}>
                            {formatCountdown(t, minutes)}
                          </span>
                        ) : undefined
                      }
                    />
                  </div>
                );
              })}
            </div>
            {current && (
              <>
                <WatchtowerItemCard
                  key={current.id}
                  item={current}
                  timezone={timezone}
                  leaving={actions.leaving}
                  onReviewAndApprove={actions.reviewAndApprove}
                />
                <WatchtowerItemBar item={current} actions={actions} canAct={canAct} />
              </>
            )}
          </div>
        )}
      </aside>
    </div>
  );
}
