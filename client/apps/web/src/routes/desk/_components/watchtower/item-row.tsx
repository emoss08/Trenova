import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import type { WatchtowerItem } from "@/lib/graphql/watchtower";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { QueueRow, QueueUnseenDot } from "../queue/queue-list";
import { WatchtowerKindGlyph } from "./kind-glyph";
import { formatCountdown, isHot, minutesLeft, showsCountdown } from "./lanes";
import { severityClass } from "./severity";

/**
 * One item in the queue: its kind (or the agent working it) on a tile tinted
 * by severity, the title over what it is about, and the countdown when the
 * deadline is inside a day — else a dot while nobody has opened it.
 */
export function WatchtowerItemRow({
  item,
  now,
  selected,
  onSelect,
}: {
  item: WatchtowerItem;
  now: number;
  selected: boolean;
  onSelect: () => void;
}) {
  const t = useT();
  const minutes = minutesLeft(item.dueAt, now);
  const work = item.agentWork;
  const agentName = work?.agent?.name ?? t("An agent");

  const subtitle = work
    ? work.status === "NeedsYou"
      ? t("{0} needs you", agentName)
      : t("{0} is on it", agentName)
    : item.record?.label || item.kindLabel;

  return (
    <QueueRow
      className="dk-w3-qi"
      selected={selected}
      onSelect={onSelect}
      leading={
        <span className={cn("dk-w3-ic", severityClass(item.severity))}>
          {work ? (
            <DeskAgentTile agent={work.agent} size="xs" />
          ) : (
            <WatchtowerKindGlyph kind={item.sourceKind} />
          )}
        </span>
      }
      title={item.title}
      subtitle={subtitle}
      trailing={
        showsCountdown(minutes) ? (
          <span className={cn("dk-w3-due", isHot(minutes) && "dk-hot")}>
            {formatCountdown(t, minutes)}
          </span>
        ) : (
          !item.seen && <QueueUnseenDot label={t("Not opened yet")} />
        )
      }
    />
  );
}
