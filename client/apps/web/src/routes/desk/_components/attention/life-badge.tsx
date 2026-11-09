import type { DeskActivityOutcome, DeskAttentionItem } from "@/lib/graphql/desk-attention";
import { DeskIcon, type DeskIconName } from "@/components/desk-chat/desk-icons";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

/**
 * Where a piece of work stands, in the one vocabulary the Desk uses for it
 * everywhere: an agent looking into it, a change waiting on a person, and how
 * a write ended.
 */
export type DeskLife =
  | "looking"
  | "drafted"
  | "failed"
  | "auto"
  | "done"
  | "simulated"
  | "declined";

const TONE: Record<DeskLife, string> = {
  looking: "",
  drafted: "dk-warn",
  failed: "dk-crit",
  auto: "dk-ok",
  done: "dk-ok",
  simulated: "dk-dash",
  declined: "",
};

const ICON: Partial<Record<DeskLife, DeskIconName>> = {
  failed: "alert",
  auto: "check",
  done: "check",
  simulated: "eye",
  declined: "x",
};

function lifeLabel(t: TranslateFn, life: DeskLife): string {
  switch (life) {
    case "looking":
      return t("Looking into it");
    case "drafted":
      return t("Needs your approval");
    case "failed":
      return t("Didn't go through");
    case "auto":
      return t("Done on its own");
    case "done":
      return t("Done");
    case "simulated":
      return t("Previewed only");
    default:
      return t("Declined");
  }
}

/** What an open item says about the work on it: an agent on it, or a change waiting. */
export function itemLife(item: DeskAttentionItem): DeskLife | null {
  if (item.agentWork?.failed) {
    return "failed";
  }
  if (item.agentWork?.status === "Working") {
    return "looking";
  }
  if (item.deskLens === "Decisions") {
    return "drafted";
  }
  return null;
}

/** A row's state: a change waiting on the person first, then an agent at work. */
export function groupLife(items: readonly DeskAttentionItem[]): DeskLife | null {
  const lives = items.map(itemLife);
  for (const wanted of ["drafted", "failed", "looking"] as const) {
    if (lives.includes(wanted)) {
      return wanted;
    }
  }
  return null;
}

export function outcomeLife(outcome: DeskActivityOutcome): DeskLife {
  switch (outcome) {
    case "OnItsOwn":
      return "auto";
    case "Approved":
      return "done";
    case "Previewed":
      return "simulated";
    default:
      return "failed";
  }
}

/** The label that says where a piece of work stands; never a pill, a 4px corner. */
export function LifeBadge({ life, extra }: { life: DeskLife | null; extra?: string }) {
  const t = useT();
  if (life === null) {
    return null;
  }
  const icon = ICON[life];

  return (
    <span className={cn("dk-w-bdg", TONE[life])}>
      {life === "looking" ? (
        <NovaBadge size="xs" spin="always" />
      ) : icon ? (
        <DeskIcon name={icon} size={11} />
      ) : null}
      {lifeLabel(t, life)}
      {extra ? ` ${extra}` : null}
    </span>
  );
}
