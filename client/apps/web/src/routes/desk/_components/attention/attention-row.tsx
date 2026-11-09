import { DeskIcon, type DeskIconName } from "@/components/desk-chat/desk-icons";
import type { DeskAttentionGroup, DeskLens } from "@/lib/graphql/desk-attention";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { formatCountdown, minutesLeft } from "../watchtower/lanes";
import { severityLabel } from "../watchtower/severity";
import { LifeBadge, groupLife } from "./life-badge";
import { groupName, leadItem, lensLabel } from "./lens";

const LENS_ICON: Record<DeskLens, DeskIconName> = {
  Everything: "stack",
  Watchtower: "radar",
  Triage: "mail",
  Decisions: "inbox",
};

const SEVERITY_CLASS: Record<string, string> = {
  Critical: "dk-crit",
  Warning: "dk-warn",
  Info: "dk-info",
};

/** How the time left reads: past due or inside four hours is critical, inside a day a warning. */
export function dueTone(minutes: number | null): string {
  if (minutes === null) {
    return "";
  }
  if (minutes < 240) {
    return "dk-crit";
  }
  return minutes < 1440 ? "dk-warn" : "";
}

/** The record line under a row's title: what the record is, then what the server says of it. */
export function recordContext(t: TranslateFn, group: DeskAttentionGroup): string[] {
  const record = group.record;
  if (record?.readable) {
    return [record.subtitle, record.status].filter((part) => part !== "");
  }
  if (record) {
    return [t("A record you can't open")];
  }
  return [];
}

/**
 * One row of Needs you: one record and every open signal about it, each from
 * the queue that raised it. A shipment's late warning, its customer's email
 * and the plan to recover it are one row, soonest deadline first. Its right
 * side says how long is left and where the work stands.
 */
export function AttentionRow({
  group,
  now,
  selected,
  onOpen,
  id,
}: {
  group: DeskAttentionGroup;
  now: number;
  /** Set in a list a person moves through; left out on Today, where a row only opens. */
  selected?: boolean;
  onOpen: () => void;
  id?: string;
}) {
  const t = useT();
  const lead = leadItem(group);
  const minutes = minutesLeft(group.dueAt, now);
  const countdown = formatCountdown(t, minutes);
  const record = group.record;
  const recordLabel = record?.readable && record.label !== "" ? record.label : groupName(t, group);

  return (
    <button
      type="button"
      id={id}
      role={selected === undefined ? undefined : "option"}
      aria-selected={selected}
      className={cn("dk-w-ar ui-focus-ring", selected && "dk-sel")}
      onClick={onOpen}
    >
      <span
        className={cn("dk-w-ar-sv", SEVERITY_CLASS[group.severity])}
        role="img"
        aria-label={severityLabel(t, group.severity)}
      />
      <span className="dk-w-ar-main">
        <span className="dk-w-ar-ti">
          {lead.title}
          {group.unseen && <span className="dk-w-newdot" aria-label={t("Not opened yet")} />}
        </span>
        <span className="dk-w-ar-rc">
          <span className="dk-w-mono">{recordLabel}</span>
          {recordContext(t, group).map((part) => (
            <span key={part}>· {part}</span>
          ))}
        </span>
      </span>
      <span className="dk-w-ar-rt">
        {countdown !== null ? (
          <span className={cn("dk-w-due", dueTone(minutes))}>{countdown}</span>
        ) : (
          group.dueLabel && <span className="dk-w-due">{group.dueLabel}</span>
        )}
        <LifeBadge life={groupLife(group.items)} />
      </span>
      <span className="dk-w-ar-sigs">
        {group.items.map((item) => (
          <span key={item.id} className="dk-w-sig">
            <DeskIcon name={LENS_ICON[item.deskLens]} size={12} />
            <span className="dk-w-sig-src">{lensLabel(t, item.deskLens)}</span>
            <span className="dk-w-sig-tx">
              {item.id === lead.id ? item.summary || item.kindLabel : item.title}
            </span>
          </span>
        ))}
      </span>
    </button>
  );
}
