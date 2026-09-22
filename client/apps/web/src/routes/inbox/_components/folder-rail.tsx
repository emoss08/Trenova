import type { InboundMailbox, InboundMessageCounts } from "@/lib/graphql/inbox";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  ArchiveIcon,
  CheckCheckIcon,
  InboxIcon,
  LayersIcon,
  MailIcon,
  type LucideIcon,
} from "lucide-react";
import {
  CLASSIFICATION_ICON,
  INBOUND_CLASSIFICATIONS,
  classificationLabel,
} from "./classification";
import { isSameFolder, type InboxFolder } from "./folders";
import type { LaneKey } from "./lanes";

const LANE_ICON: Record<LaneKey, LucideIcon> = {
  waiting: InboxIcon,
  handled: CheckCheckIcon,
  ignored: ArchiveIcon,
  all: LayersIcon,
};

export function laneLabel(t: (value: string) => string, lane: LaneKey): string {
  switch (lane) {
    case "waiting":
      return t("Waiting on you");
    case "handled":
      return t("Handled");
    case "ignored":
      return t("Ignored");
    case "all":
      return t("Everything");
  }
}

function laneCount(counts: InboundMessageCounts | undefined, lane: LaneKey): number | undefined {
  if (counts === undefined) {
    return undefined;
  }
  switch (lane) {
    case "waiting":
      return counts.waiting;
    case "handled":
      return counts.handled;
    case "ignored":
      return counts.ignored;
    case "all":
      return counts.total;
  }
}

/**
 * The inbox's folders: lanes by state, kinds by what the mail is, and the
 * addresses it came in on.
 *
 * A folder shows its waiting count when anything in it is waiting, and its
 * total, quietly, when nothing is — the number a person acts on is the loud
 * one, and a folder with nothing to do still says how much it holds.
 */
export function FolderRail({
  folder,
  counts,
  mailboxes,
  onSelect,
}: {
  folder: InboxFolder;
  counts?: InboundMessageCounts;
  mailboxes: InboundMailbox[];
  onSelect: (folder: InboxFolder) => void;
}) {
  const t = useT();
  const mailboxCounts = new Map(counts?.byMailbox.map((row) => [row.mailboxId, row]) ?? []);
  const kindCounts = new Map(
    counts?.byClassification.map((row) => [row.classification, row]) ?? [],
  );

  return (
    <nav aria-label={t("Inbox folders")} className="flex min-h-0 flex-col">
      <ScrollArea className="min-h-0 flex-1">
        <div className="flex flex-col gap-5 p-3">
          <RailSection>
            {(["waiting", "handled", "ignored", "all"] as const).map((lane) => {
              const target: InboxFolder = { kind: "lane", lane };
              const count = laneCount(counts, lane);
              return (
                <RailItem
                  key={lane}
                  icon={LANE_ICON[lane]}
                  label={laneLabel(t, lane)}
                  active={isSameFolder(folder, target)}
                  count={count}
                  loud={lane === "waiting" && (count ?? 0) > 0}
                  onClick={() => onSelect(target)}
                />
              );
            })}
          </RailSection>

          <RailSection title={t("By kind")}>
            {INBOUND_CLASSIFICATIONS.map((classification) => {
              const target: InboxFolder = { kind: "classification", classification };
              const row = kindCounts.get(classification);
              const waiting = row?.waiting ?? 0;
              return (
                <RailItem
                  key={classification}
                  icon={CLASSIFICATION_ICON[classification]}
                  label={classificationLabel(t, classification)}
                  active={isSameFolder(folder, target)}
                  count={waiting > 0 ? waiting : row?.total}
                  loud={waiting > 0}
                  onClick={() => onSelect(target)}
                />
              );
            })}
          </RailSection>

          {mailboxes.length > 0 && (
            <RailSection title={t("Mailboxes")}>
              {mailboxes.map((mailbox) => {
                const target: InboxFolder = { kind: "mailbox", mailboxId: mailbox.id };
                const row = mailboxCounts.get(mailbox.id);
                const waiting = row?.waiting ?? 0;
                return (
                  <RailItem
                    key={mailbox.id}
                    icon={MailIcon}
                    label={mailbox.name === "" ? mailbox.address : mailbox.name}
                    hint={mailbox.name === "" ? undefined : mailbox.address}
                    muted={mailbox.status !== "Active"}
                    active={isSameFolder(folder, target)}
                    count={waiting > 0 ? waiting : (row?.total ?? 0)}
                    loud={waiting > 0}
                    onClick={() => onSelect(target)}
                  />
                );
              })}
            </RailSection>
          )}
        </div>
      </ScrollArea>

      <KeyboardLegend />
    </nav>
  );
}

function RailSection({ title, children }: { title?: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5">
      {title !== undefined && (
        <h3 className="text-foreground-subtle px-2 pb-1 text-xs font-medium">{title}</h3>
      )}
      {children}
    </div>
  );
}

function RailItem({
  icon: Icon,
  label,
  hint,
  count,
  loud = false,
  muted = false,
  active,
  onClick,
}: {
  icon: LucideIcon;
  label: string;
  hint?: string;
  count?: number;
  loud?: boolean;
  muted?: boolean;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-current={active ? "page" : undefined}
      title={hint}
      className={cn(
        "ui-focus-ring flex h-8 w-full items-center gap-2.5 rounded-md px-2 text-left text-sm transition-colors",
        active
          ? "bg-nav-active text-nav-active-foreground"
          : "text-foreground-muted hover:bg-surface-hover hover:text-foreground",
        muted && !active && "text-foreground-subtle",
      )}
    >
      <Icon className="size-4 shrink-0" aria-hidden />
      <span className={cn("min-w-0 flex-1 truncate", loud && "text-foreground font-medium")}>
        {label}
      </span>
      {count !== undefined && count > 0 && (
        <span
          className={cn(
            "shrink-0 text-xs tabular-nums",
            loud ? "text-foreground font-medium" : "text-foreground-subtle",
          )}
        >
          {count}
        </span>
      )}
    </button>
  );
}

/** The keys, where a person looking for them will find them. */
function KeyboardLegend() {
  const t = useT();
  const rows: [string, string][] = [
    ["j k", t("Next, previous")],
    ["e", t("Mark handled")],
    ["#", t("Ignore")],
    ["l", t("Link by hand")],
    ["a", t("Ask the desk")],
    ["/", t("Search")],
  ];

  return (
    <div className="border-border hidden flex-col gap-1.5 border-t p-3 lg:flex">
      {rows.map(([keys, label]) => (
        <div
          key={keys}
          className="text-foreground-subtle flex items-center justify-between text-xs"
        >
          <span>{label}</span>
          <span className="flex gap-1">
            {keys.split(" ").map((key) => (
              <Kbd key={key}>{key}</Kbd>
            ))}
          </span>
        </div>
      ))}
    </div>
  );
}
