import { humanizeCode } from "@/components/assistant/readable-values";
import { DeskIcon, type DeskIconName } from "@/components/desk-chat/desk-icons";
import type { DeskAttentionGroup, DeskLens } from "@/lib/graphql/desk-attention";
import { formatList } from "@trenova/shared/i18n/format";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { formatCountdown, minutesLeft } from "../watchtower/lanes";
import { severityLabel, severityMarkClass } from "../watchtower/severity";
import { caseWorkLabel, caseWorkOf, caseWorkShort } from "./case-work";
import { LifeBadge, groupLife } from "./life-badge";
import { deskListRow } from "../list/desk-list";
import { groupLenses, groupName, leadItem, lensLabel } from "./lens";

const LENS_ICON: Record<DeskLens, DeskIconName> = {
  Everything: "stack",
  Watchtower: "radar",
  Triage: "mail",
  Decisions: "inbox",
};

// i18n-ignore: class names, never shown to a person
const DUE_CLASS: Record<string, string> = {
  "dk-crit": "text-danger-subtle-foreground",
  "dk-warn": "text-dsk-warn-fg",
  "": "text-dsk-subtle",
};

// i18n-ignore: class names, never shown to a person
const DUE_TEXT = "text-dsk-sm font-medium whitespace-nowrap tabular-nums";

/**
 * What a decision's row leads with in the Decisions lens: the change itself,
 * and under it who proposes it. Every row there waits on an approval, so the
 * row spends its words on what differs between them. The server's item says
 * both as facts; a row without them reads as it does everywhere else.
 */
export function decisionLine(group: DeskAttentionGroup): { change: string; agent: string } | null {
  const decision = group.items.find((item) => item.deskLens === "Decisions");
  if (!decision) {
    return null;
  }
  const fact = (label: string) =>
    decision.facts.find((entry) => entry.label === label)?.value?.trim() ?? "";
  const change = fact("Change");
  if (change === "") {
    return null;
  }

  return { change: change.charAt(0).toUpperCase() + change.slice(1), agent: fact("Agent") };
}

// i18n-ignore: class names, never shown to a person
const SIGNALS_CLASS = "inline-flex flex-none items-center gap-2 text-dsk-faint";

// i18n-ignore: class names, never shown to a person
const SIGNAL_CLASS = "inline-flex items-center gap-[3px] text-xs leading-[inherit] tabular-nums";

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
    // A status arrives as the server names it ("ReadyToInvoice"); a person reads words.
    return [record.subtitle, humanizeCode(record.status)].filter((part) => part !== "");
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
 * says where the work stands, and whose case it is once someone has picked
 * it up. Each signal in full is in the item it opens.
 *
 * In the Decisions lens every row waits on an approval, so saying so on each
 * is noise: the row leads with the change and names who proposes it instead,
 * and keeps a badge only for what is not the same on every row.
 */
export function AttentionRow({
  group,
  now,
  selected,
  onOpen,
  id,
  position,
  lens,
}: {
  group: DeskAttentionGroup;
  now: number;
  /** Set in a list a person moves through; left out on Today, where a row only opens. */
  selected?: boolean;
  onOpen: () => void;
  id?: string;
  /** Where the row sits in a virtualized list, so assistive tech counts rows that are not mounted. */
  position?: { index: number; size: number };
  /** The lens the list is read through; the Decisions lens draws its rows as decisions. */
  lens?: DeskLens;
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
  const decision = lens === "Decisions" ? decisionLine(group) : null;
  const work = caseWorkOf(group);
  const context = (
    decision
      ? [decision.agent, record ? recordLabel : "", ...recordContext(t, group)]
      : [recordLabel, ...recordContext(t, group)]
  ).filter((part) => part !== "");
  const lenses = groupLenses(group).map((entry) => ({
    lens: entry,
    count: group.items.filter((item) => item.deskLens === entry).length,
  }));
  const signals = formatList(
    lenses.map((entry) => t("{0}: {1}", lensLabel(t, entry.lens), entry.count)),
  );
  // "Needs your approval" is every row of the Decisions lens; anything else
  // a row's work can say (an agent on it, a write that failed) still shows.
  const shownLife = groupLife(group.items, lens === "Decisions");

  const inList = selected !== undefined;

  return (
    <button
      type="button"
      id={id}
      role={selected === undefined ? undefined : "option"}
      aria-selected={selected}
      aria-posinset={position ? position.index + 1 : undefined}
      aria-setsize={position?.size}
      // In a list the pane's glide is the hover; on Today the row lights itself.
      data-glide-target={inList ? true : undefined}
      className={cn(
        // dk-w-ar is the row's entrance (desk-room.css), which the list turns off once settled.
        "dk-w-ar grid h-14 w-full grid-cols-[20px_minmax(0,1fr)_auto] grid-rows-[18px_18px] items-center gap-x-3 gap-y-0.5 ui-focus-ring px-3 py-[9px]",
        inList
          ? deskListRow(selected === true).className
          : "relative rounded-[8px] text-left hover:bg-dsk-hover",
      )}
      onClick={onOpen}
    >
      <span
        className={cn(
          "row-start-1 size-2 justify-self-center rounded-[2px]",
          severityMarkClass(group.severity),
        )}
        role="img"
        aria-label={severityLabel(t, group.severity)}
      />
      <span className="flex min-w-0 items-center text-base/[18px] font-medium text-dsk-fg">
        <span className="min-w-0 truncate">{decision ? decision.change : lead.title}</span>
        {group.unseen && (
          <span
            className="ml-1.5 inline-block size-1.5 flex-none rounded-full bg-dsk-blue align-middle"
            aria-label={t("Not opened yet")}
          />
        )}
      </span>
      <span className="justify-self-end leading-[18px]">
        {countdown !== null ? (
          <span className={cn(DUE_TEXT, DUE_CLASS[dueTone(minutes)])}>{countdown}</span>
        ) : (
          group.dueLabel && (
            <span className={cn(DUE_TEXT, "text-dsk-subtle")}>{group.dueLabel}</span>
          )
        )}
      </span>
      <span className="col-start-2 flex min-w-0 items-center gap-2.5 text-sm/[18px] text-dsk-subtle">
        <span className="min-w-0 truncate">{context.join(" · ")}</span>
        {work && (
          <span
            className={SIGNALS_CLASS}
            title={caseWorkLabel(t, work)}
            aria-label={caseWorkLabel(t, work)}
          >
            <span className={SIGNAL_CLASS}>
              <DeskIcon name="file" size={11} />
              {caseWorkShort(t, work)}
            </span>
          </span>
        )}
        {lenses.length > 0 && (
          <span className={SIGNALS_CLASS} title={signals} aria-label={signals}>
            {lenses.map((entry) => (
              <span key={entry.lens} className={SIGNAL_CLASS}>
                <DeskIcon name={LENS_ICON[entry.lens]} size={11} />
                {entry.count}
              </span>
            ))}
          </span>
        )}
      </span>
      {shownLife !== null && (
        <span className="col-start-3 flex items-center justify-self-end">
          <LifeBadge life={shownLife} />
        </span>
      )}
    </button>
  );
}
