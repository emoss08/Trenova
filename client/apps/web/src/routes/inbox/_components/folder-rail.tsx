import { RailItem, RailSection } from "@/components/navigation/queue-rail";
import { usePermission } from "@/hooks/use-permission";
import type { InboundMailbox, InboundMessageCounts } from "@/lib/graphql/inbox";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  ArchiveIcon,
  CheckCheckIcon,
  InboxIcon,
  LayersIcon,
  MailIcon,
  type LucideIcon,
} from "lucide-react";
import { Link } from "react-router";
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
  const { allowed: canManageMailboxes } = usePermission(Resource.InboundMailbox, Operation.Read);
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

          {(mailboxes.length > 0 || canManageMailboxes) && (
            <RailSection
              title={t("Mailboxes")}
              action={
                canManageMailboxes ? (
                  <Link
                    to="/admin/inbound-mailboxes"
                    className="ui-focus-ring text-foreground-subtle hover:text-foreground text-xs"
                  >
                    {t("Manage")}
                  </Link>
                ) : undefined
              }
            >
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
