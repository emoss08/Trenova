import type { DeskActivityOutcome, DeskAttentionItem } from "@/lib/graphql/desk-attention";
import { DeskIcon, type DeskIconName } from "@/components/desk-chat/desk-icons";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { BadgeTone } from "@trenova/shared/types/badge";

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

const TONE: Record<DeskLife, BadgeTone> = {
  looking: "neutral",
  drafted: "warning",
  failed: "danger",
  auto: "success",
  done: "success",
  simulated: "neutral",
  declined: "neutral",
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

/**
 * A row's state: a change waiting on the person first, then an agent at work.
 * `besidesWaiting` leaves the waiting change out, for a list where every row
 * waits on one and the row should say what else is true of it.
 */
export function groupLife(
  items: readonly DeskAttentionItem[],
  besidesWaiting = false,
): DeskLife | null {
  const lives = items.map(itemLife);
  const order = besidesWaiting
    ? (["failed", "looking"] as const)
    : (["drafted", "failed", "looking"] as const);
  for (const wanted of order) {
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

/**
 * The label that says where a piece of work stands: the standard badge, led
 * by its icon or Nova's mark where the state has one and by a dot otherwise.
 * A preview that was never made is outlined, since nothing happened.
 */
export function LifeBadge({ life, extra }: { life: DeskLife | null; extra?: string }) {
  const t = useT();
  if (life === null) {
    return null;
  }
  const icon = ICON[life];

  return (
    <Badge
      variant={TONE[life]}
      appearance={life === "simulated" ? "outline" : "subtle"}
      dot={life !== "looking" && !icon}
    >
      {life === "looking" ? (
        <NovaBadge size="xs" spin="always" />
      ) : icon ? (
        <DeskIcon name={icon} size={11} />
      ) : null}
      {lifeLabel(t, life)}
      {extra ? ` ${extra}` : null}
    </Badge>
  );
}
