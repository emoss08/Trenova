import { toneVar } from "@/components/kpi/tone";
import type { WatchtowerItem } from "@/lib/graphql/watchtower";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowRightIcon, XIcon } from "lucide-react";
import { Link } from "react-router";
import { SEVERITY_TONE } from "./severity";
import { FeedbackControl } from "@/components/ai-feedback/feedback-control";
import { isRatableWatchtowerKind } from "@/components/ai-feedback/feedback-targets";

const ROW_FEEDBACK_REVEAL = "opacity-0 group-hover:opacity-100";

/**
 * One thing worth a person's attention.
 *
 * The row is the item, not a card about it: a dot for how loudly it is
 * asking, the title, what it is, when it happened, and the two things a
 * person does with it — open the record, or hand it to an agent. Everything
 * else is on the record's own page, which is one click away, so the feed
 * stays a list a person can run their eye down.
 */
export function WatchtowerItemRow({
  item,
  unseen,
  now,
  onAsk,
  onHandOff,
  onDismiss,
  busy = false,
}: {
  item: WatchtowerItem;
  unseen: boolean;
  now: number;
  onAsk?: (item: WatchtowerItem) => void;
  onHandOff: (item: WatchtowerItem) => void;
  onDismiss: (item: WatchtowerItem) => void;
  busy?: boolean;
}) {
  const t = useT();
  const canHandOff = item.subjectType !== null && item.subjectId !== null;

  return (
    <li
      className={cn(
        "group border-desk-hairline flex items-start gap-3 border-b px-4 py-3",
        "hover:bg-surface-hover transition-colors",
      )}
    >
      <span
        aria-hidden
        className="mt-1.5 size-1.5 shrink-0 rounded-full"
        style={{ backgroundColor: toneVar(SEVERITY_TONE[item.severity]) }}
      />

      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <div className="flex min-w-0 items-baseline gap-2">
          <span className={cn("min-w-0 truncate text-sm", unseen && "font-medium")}>
            {item.title}
          </span>
          {unseen && <span className="sr-only">{t("New since you last looked")}</span>}
        </div>
        {item.summary !== "" && (
          <p className="text-muted-foreground line-clamp-2 text-xs leading-relaxed">
            {item.summary}
          </p>
        )}
        <div className="text-muted-foreground flex items-center gap-1.5 pt-0.5 text-xs">
          <span>{item.kindLabel}</span>
          <span aria-hidden>·</span>
          <span className="tabular-nums">{formatSecondsAgo(now - item.occurredAt)}</span>
        </div>
      </div>

      {isRatableWatchtowerKind(item.sourceKind) && (
        <FeedbackControl
          target={{ targetType: "WatchtowerItem", targetId: item.id }}
          revealClassName={ROW_FEEDBACK_REVEAL}
          className="self-center"
        />
      )}

      <div className="flex shrink-0 items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100">
        {onAsk && (
          <Button size="sm" variant="ghost" onClick={() => onAsk(item)} disabled={busy}>
            {t("Ask about this")}
          </Button>
        )}
        {canHandOff && (
          <Button size="sm" variant="outline" onClick={() => onHandOff(item)} disabled={busy}>
            {t("Hand off")}
          </Button>
        )}
        {item.path !== "" && (
          <Button
            size="icon-sm"
            variant="ghost"
            nativeButton={false}
            aria-label={t("Open the record")}
            render={<Link to={item.path} />}
          >
            <ArrowRightIcon className="size-4" />
          </Button>
        )}
        <Button
          size="icon-sm"
          variant="ghost"
          aria-label={t("Dismiss")}
          className="text-muted-foreground hover:text-foreground"
          onClick={() => onDismiss(item)}
          disabled={busy}
        >
          <XIcon className="size-4" />
        </Button>
      </div>
    </li>
  );
}
