import { useAssistantAgent, useDelegateIdentity } from "@/components/agent-identity/agent-context";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  CircleAlertIcon,
  CircleCheckIcon,
  CircleSlashIcon,
  FlaskConicalIcon,
  PauseCircleIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { humanizeToolName, proposedByOther, type ProposalPresentation } from "./proposal-state";
import { WorkingDot } from "./voice/working-dot";

/**
 * Whether this view watched the state change. A card that moves from
 * "waiting" to "done" while someone looks at it says so with motion; a card
 * opened from history is simply in its state, with nothing to announce.
 */
export function useWatchedChange<T>(value: T): boolean {
  const [first] = useState(value);

  return value !== first;
}

/**
 * The mark of a settled decision. Work still running is the breathing dot,
 * the one loop the product allows; an outcome that lands while watched
 * settles with the confirm spring.
 */
export function OutcomeIcon({
  state,
  className,
}: {
  state: ProposalPresentation;
  className?: string;
}) {
  const watched = useWatchedChange(state);
  const glyph = cn("size-3.5 shrink-0", watched && "animate-confirm", className);

  switch (state) {
    case "failed":
      return <CircleAlertIcon key={state} aria-hidden className={cn(glyph, "text-danger")} />;
    case "done":
      return <CircleCheckIcon key={state} aria-hidden className={cn(glyph, "text-success")} />;
    case "running":
      return <WorkingDot working />;
    case "simulated":
      return (
        <FlaskConicalIcon key={state} aria-hidden className={cn(glyph, "text-foreground-muted")} />
      );
    case "held":
      return (
        <PauseCircleIcon key={state} aria-hidden className={cn(glyph, "text-foreground-muted")} />
      );
    default:
      return (
        <CircleSlashIcon key={state} aria-hidden className={cn(glyph, "text-foreground-subtle")} />
      );
  }
}

/**
 * Who proposed a change, when it was not the agent the conversation is with:
 * "Proposed by Report Builder", beside that agent's mark. The conversation's
 * own agent heads the reply the card sits in, so its cards say nothing.
 */
export function ProposedBy({
  agentId,
  agentName,
}: {
  agentId?: string | null;
  agentName?: string | null;
}) {
  const t = useT();
  const conversation = useAssistantAgent();
  const fallback = useMemo(
    () => ({ id: agentId ?? "", name: agentName ?? "" }),
    [agentId, agentName],
  );
  const identity = useDelegateIdentity(fallback);
  const name = identity.name ?? "";

  if (!proposedByOther(conversation?.id, agentId) || name === "") {
    return null;
  }

  return (
    <p className="text-foreground-subtle flex min-w-0 items-center gap-1.5 text-xs">
      <AgentTile agent={identity} size="xs" />
      <span className="min-w-0 truncate">{t("Proposed by {0}", name)}</span>
    </p>
  );
}

/** The verbs a tool name opens with that read better as another word. */
const VERBS: Readonly<Record<string, string>> = {
  transition: "Move",
  reassign: "Reassign",
};

/** Arguments that name a record the way a person would, in the order they are preferred. */
const NAME_KEYS = [
  "number",
  "invoiceNumber",
  "proNumber",
  "code",
  "reference",
  "title",
  "name",
] as const;

const OPAQUE_ID = /^[a-z]{2,8}_[0-9A-Za-z]{6,}$/u;

function words(identifier: string): string[] {
  return identifier
    .replace(/([a-z0-9])([A-Z])/gu, "$1 $2")
    .split(/[_\s]+/u)
    .map((word) => word.toLowerCase())
    .filter((word) => word !== "");
}

function singular(word: string): string {
  if (/ies$/u.test(word)) return word.slice(0, -3) + "y";
  if (/(ses|xes|ches|shes)$/u.test(word)) return word.slice(0, -2);
  if (/s$/u.test(word) && !/ss$/u.test(word)) return word.slice(0, -1);
  return word;
}

function plural(word: string): string {
  if (/y$/u.test(word) && !/[aeiou]y$/u.test(word)) return word.slice(0, -1) + "ies";
  if (/(s|x|ch|sh)$/u.test(word)) return word + "es";
  return word + "s";
}

function startsWith(list: readonly string[], prefix: readonly string[]): boolean {
  return prefix.length > 0 && prefix.every((word, index) => list[index] === word);
}

/** The records a call is about, read from its `…Ids` and `…Id` arguments. */
function recordsOf(
  args: Readonly<Record<string, unknown>>,
): { noun: string[]; count: number; listed: boolean } | null {
  for (const [key, value] of Object.entries(args)) {
    if (/Ids$/u.test(key) && Array.isArray(value)) {
      return { noun: words(key.slice(0, -3)), count: value.length, listed: true };
    }
  }
  for (const key of Object.keys(args)) {
    if (/Id$/u.test(key) && key !== "Id") {
      return { noun: words(key.slice(0, -2)), count: 1, listed: false };
    }
  }

  return null;
}

function nameOf(args: Readonly<Record<string, unknown>>): string {
  for (const key of NAME_KEYS) {
    const value = args[key];
    if (typeof value === "string" && value.trim() !== "" && !OPAQUE_ID.test(value.trim())) {
      return value.trim();
    }
  }

  return "";
}

function counted(count: number, noun: readonly string[]): string {
  const head = noun.slice(0, -1).join(" ");
  const last = noun.at(-1) ?? "";
  const word = count === 1 ? singular(last) : plural(singular(last));

  return `${count} ${head === "" ? word : `${head} ${word}`}`;
}

/**
 * What a proposal asks to do, in the words of the tool and the records it is
 * about: "Assign biller · 11 billing queue items", "Approve 11 billing queue
 * items", "Send invoice INV2610000016", "Move 3 items into review". The tool
 * name gives the verb and its object; the arguments give the records, counted
 * from an `…Ids` list and named from a number or a name, never from an id.
 */
export function proposalLabel(
  toolName: string,
  args: Readonly<Record<string, unknown>> | null | undefined,
  t: TranslateFn,
): string {
  const parts = words(toolName);
  if (parts.length === 0) {
    return humanizeToolName(toolName);
  }
  const given = args ?? {};
  const [verbWord, ...rest] = parts;
  const verb = VERBS[verbWord] ?? verbWord.charAt(0).toUpperCase() + verbWord.slice(1);
  const records = recordsOf(given);
  const name = nameOf(given);

  const to = rest.indexOf("to");
  if (verbWord === "transition" && to > 0) {
    const target = rest.slice(to + 1).join(" ").replace(/^in /u, "");
    const object = rest.slice(0, to);
    const what =
      records && records.count > 1
        ? counted(records.count, object)
        : object.map((word) => singular(word)).join(" ");

    return t("{0} {1} into {2}", verb, what, target);
  }

  if (records) {
    const object = rest.map((word) => singular(word)).join(" ");
    const noun = records.noun.map((word) => singular(word)).join(" ");
    if (!records.listed && (object === noun || rest.length === 0)) {
      const what = object === "" ? noun : object;
      return name === "" ? `${verb} ${what}` : `${verb} ${what} ${name}`;
    }
    if (object === noun || rest.length === 0) {
      return `${verb} ${counted(records.count, records.noun)}`;
    }
    if (startsWith(rest, records.noun.slice(0, -1)) && rest.length > records.noun.length - 1) {
      const own = rest.slice(records.noun.length - 1).map((word) => singular(word)).join(" ");
      return `${verb} ${own} · ${counted(records.count, records.noun)}`;
    }
    if (!records.listed) {
      return name === "" ? `${verb} ${object}` : `${verb} ${object} ${name}`;
    }

    return `${verb} ${object} · ${counted(records.count, records.noun)}`;
  }

  const object = rest.join(" ");
  if (name !== "") {
    return object === "" ? `${verb} ${name}` : `${verb} ${object} ${name}`;
  }

  return humanizeToolName(toolName);
}
