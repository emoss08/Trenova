import type { DeskBoardBlock } from "@/lib/graphql/desk-board-card";
import type { DeskLens } from "@/lib/graphql/desk-attention";
import { useDeskStore } from "@/stores/desk-store";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Link } from "react-router";
import { casePath } from "../../../attention/attention-record";
import { lensLabel } from "../../../attention/lens";
import { formatCountdown, isHot, showsCountdown } from "../../../watchtower/lanes";

type WorkRow = DeskBoardBlock["work"][number];
type BoardAction = DeskBoardBlock["actions"][number];

function minutesUntil(dueAt: number | null, now: number): number | null {
  return dueAt === null ? null : Math.round((dueAt - now) / 60);
}

/**
 * A work block: each record the board names with what is open about it, as
 * Needs you holds it, soonest deadline first. A row opens the record's case,
 * where the work is done; a record the reader may not open is counted, never
 * named.
 */
export function BoardWork({ rows, now }: { rows: readonly WorkRow[]; now: number }) {
  const t = useT();

  return (
    <ul className="dk-bd-work">
      {rows.map((entry) => {
        const minutes = minutesUntil(entry.dueAt ?? null, now);
        const name =
          entry.row.readable && entry.row.label !== ""
            ? entry.row.label
            : t("A record you can't open");
        const body = (
          <>
            <span
              className={cn("dk-bd-work-tone", entry.tone !== "" && `dk-${entry.tone}`)}
              aria-hidden
            />
            <span className="dk-bd-work-main">
              <b>{name}</b>
              {entry.signals.length === 0 ? (
                <span className="dk-bd-work-quiet">{t("Nothing open")}</span>
              ) : (
                <span className="dk-bd-work-sigs">
                  {entry.signals.map((signal, index) => (
                    <span key={`${signal.title}-${index}`}>
                      <em>{lensLabel(t, signal.lens as DeskLens)}</em>
                      {signal.title}
                    </span>
                  ))}
                </span>
              )}
            </span>
            {showsCountdown(minutes) && (
              <span className={cn("dk-w3-due", isHot(minutes) && "dk-hot")}>
                {formatCountdown(t, minutes)}
              </span>
            )}
          </>
        );
        return (
          <li key={entry.row.id}>
            {entry.row.readable ? (
              <Link
                className="dk-bd-work-row ui-focus-ring"
                to={casePath(entry.row.resource, entry.row.id)}
              >
                {body}
              </Link>
            ) : (
              <div className="dk-bd-work-row">{body}</div>
            )}
          </li>
        );
      })}
    </ul>
  );
}

/**
 * An actions block: up to three buttons the agent offers. Each sends its
 * request as the person's own message into the board's conversation and
 * changes nothing itself; whatever the agent proposes in reply waits for
 * approval like any other change.
 */
export function BoardActions({
  actions,
  threadId,
  half,
}: {
  actions: readonly BoardAction[];
  threadId: string;
  half: boolean;
}) {
  const t = useT();
  const askAgent = useDeskStore((state) => state.askAgent);

  return (
    <div
      className={cn("dk-bd-actions", half && "dk-bd-half")}
      role="group"
      aria-label={t("Ask the agent")}
    >
      {actions.map((action) => {
        const named = action.records
          .filter((record) => record.readable && record.label !== "")
          .map((record) => record.label)
          .join(", ");
        const ask = named === "" ? action.ask : t("{0} ({1})", action.ask, named);
        return (
          <Button
            key={action.label}
            variant="outline"
            size="sm"
            title={ask}
            onClick={() => askAgent(threadId, ask)}
          >
            {action.label}
          </Button>
        );
      })}
      <span className="dk-bd-actions-note">
        {t("Sends your request to the agent; nothing changes until you approve it.")}
      </span>
    </div>
  );
}
