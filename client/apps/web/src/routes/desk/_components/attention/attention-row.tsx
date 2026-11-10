import { DeskIcon, type DeskIconName } from "@/components/desk-chat/desk-icons";
import type { DeskAttentionGroup, DeskLens } from "@/lib/graphql/desk-attention";
import { formatList } from "@trenova/shared/i18n/format";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { formatCountdown, minutesLeft } from "../watchtower/lanes";
import { severityLabel } from "../watchtower/severity";
import { LifeBadge, groupLife } from "./life-badge";
import { groupLenses, groupName, leadItem, lensLabel } from "./lens";

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
  if (record?.readable && record.label !== "") {
    return [record.subtitle, record.status].filter((part) => part !== "");
  }
  if (record && !record.readable) {
    return [t("A record you can't open")];
  }
  return [];
}

/**
 * One row of Needs you: one record and every open signal about it, always
 * two lines so every row is the same height however much is open about the
 * record. The first line says what it is and how long is left; the second
 * names the record, counts its signals by the queue that raised them and
 * says where the work stands. Each signal in full is in the item it opens.
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
  const recordLabel =
    record?.readable && record.label !== ""
      ? record.label
      : record && !record.readable
        ? ""
        : groupName(t, group);
  const context = [recordLabel, ...recordContext(t, group)].filter((part) => part !== "");
  const lenses = groupLenses(group).map((lens) => ({
    lens,
    count: group.items.filter((item) => item.deskLens === lens).length,
  }));
  const signals = formatList(
    lenses.map(({ lens, count }) => t("{0}: {1}", lensLabel(t, lens), count)),
  );

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
      <span className="dk-w-ar-ti">
        <span className="dk-w-ar-tt">{lead.title}</span>
        {group.unseen && <span className="dk-w-newdot" aria-label={t("Not opened yet")} />}
      </span>
      <span className="dk-w-ar-due">
        {countdown !== null ? (
          <span className={cn("dk-w-due", dueTone(minutes))}>{countdown}</span>
        ) : (
          group.dueLabel && <span className="dk-w-due">{group.dueLabel}</span>
        )}
      </span>
      <span className="dk-w-ar-rc">
        <span className="dk-w-ar-ctx">{context.join(" · ")}</span>
        {lenses.length > 0 && (
          <span className="dk-w-ar-sigs" title={signals} aria-label={signals}>
            {lenses.map(({ lens, count }) => (
              <span key={lens} className="dk-w-ar-sig">
                <DeskIcon name={LENS_ICON[lens]} size={11} />
                {count}
              </span>
            ))}
          </span>
        )}
      </span>
      <span className="dk-w-ar-life">
        <LifeBadge life={groupLife(group.items)} />
      </span>
    </button>
  );
}
