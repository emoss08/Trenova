import type { InboundMailbox, TriageView } from "@/lib/graphql/inbox";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import {
  ArchiveIcon,
  CheckIcon,
  ClockIcon,
  Inbox01Icon,
  LayersThree01Icon,
  Send01Icon,
  User01Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { classificationLabel } from "./classification";
import type { TriageFolder } from "./folders";
import type { LaneKey } from "./lanes";

export const LANE_ICON: Record<LaneKey, IconComponent> = {
  waiting: Inbox01Icon,
  them: Send01Icon,
  mine: User01Icon,
  snoozed: ClockIcon,
  handled: CheckIcon,
  ignored: ArchiveIcon,
  all: LayersThree01Icon,
};

export function laneLabel(t: TranslateFn, lane: LaneKey): string {
  switch (lane) {
    case "waiting":
      return t("Waiting on you");
    case "them":
      return t("Waiting on them");
    case "mine":
      return t("Assigned to me");
    case "snoozed":
      return t("Snoozed");
    case "handled":
      return t("Handled");
    case "ignored":
      return t("Ignored");
    case "all":
      return t("Everything");
  }
}

export function mailboxName(mailbox: Pick<InboundMailbox, "name" | "address">): string {
  return mailbox.name.trim() === "" ? mailbox.address : mailbox.name;
}

/** What a folder is called in the rail, the search box and its empty state. */
export function folderTitle(
  t: TranslateFn,
  folder: TriageFolder,
  mailboxes: readonly InboundMailbox[],
  views: readonly TriageView[],
): string {
  switch (folder.kind) {
    case "briefing":
      return t("Briefing");
    case "rules":
      return t("Rules");
    case "lane":
      return laneLabel(t, folder.lane);
    case "classification":
      return classificationLabel(t, folder.classification);
    case "mailbox": {
      const mailbox = mailboxes.find((row) => row.id === folder.mailboxId);
      return mailbox === undefined ? t("Source") : mailboxName(mailbox);
    }
    case "view":
      return views.find((row) => row.id === folder.viewId)?.name ?? t("View");
  }
}
