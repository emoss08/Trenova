import type { DeskAttentionGroup } from "@/lib/graphql/desk-attention";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { hasCase } from "./lens";

/**
 * Who has the case a row's record is in, once someone does. The item stays in
 * the queue, because the item is what the case is made of and what closes it
 * when the problem goes away; what changes is that it is no longer cleared,
 * snoozed or handed off from here. It is worked in the case, where a plan is
 * approved or the risk accepted with a reason.
 */
export type CaseWork = {
  /** The person who took the case or was given it; null while only an agent has it. */
  owner: { id: string; name: string } | null;
  /** True when the reader works the case: it is theirs, unowned, or they cover for its owner. */
  mine: boolean;
};

/**
 * The case a row is being worked in, or null while nobody has picked it up.
 * A case is being worked when a person took it or was given it, while an
 * agent it was handed to is still on it, or when a change about the record
 * waits to be approved. An owner the record supplied on its own (its
 * dispatcher, its biller) has not picked anything up yet. An agent that
 * finished and proposed nothing, or whose run failed, has handed the item
 * back: nobody is working it, so it is the queue's again to clear, snooze or
 * hand to someone else, with what the agent said beside it.
 */
export function caseWorkOf(group: DeskAttentionGroup): CaseWork | null {
  if (!hasCase(group)) {
    return null;
  }
  const ownership = group.ownership;
  const chosen = ownership?.owner && !ownership.auto ? ownership.owner : null;
  const handed = group.items.some(
    (item) => item.agentWork !== null && !item.agentWork.failed && item.agentWork.status !== "Done",
  );
  const waiting = group.items.some((item) => item.deskLens === "Decisions");
  if (chosen === null && !handed && !waiting) {
    return null;
  }

  return { owner: chosen, mine: ownership?.canWork ?? true };
}

/** How a row says it: whose case it is, or only that it is in one. */
export function caseWorkLabel(t: TranslateFn, work: CaseWork): string {
  if (work.owner === null) {
    return t("In a case");
  }
  return work.mine ? t("Your case") : t("{0}'s case", work.owner.name);
}

/** The few characters a narrow row has room for: who has it, when a person does. */
export function caseWorkShort(t: TranslateFn, work: CaseWork): string {
  if (work.owner === null) {
    return "";
  }
  return work.mine ? t("You") : work.owner.name;
}

/**
 * The ids of the items the queue's keys leave alone, because they are closed
 * somewhere else: an item worked in a case is closed there, and a message is
 * answered, filed or set aside in Triage, which clears it here.
 */
export function itemsWorkedElsewhere(groups: readonly DeskAttentionGroup[]): Set<string> {
  const ids = new Set<string>();
  for (const group of groups) {
    const inCase = caseWorkOf(group) !== null;
    for (const item of group.items) {
      if (inCase || item.deskLens === "Triage") {
        ids.add(item.id);
      }
    }
  }
  return ids;
}
