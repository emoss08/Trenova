import { withoutRecordIds } from "@/lib/record-ids";
import { NOTHING_WRITTEN, type CaseWriting as Writing } from "@/lib/desk-case-stream";
import { CaseWritingCard } from "./case-writing";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { AiMarkdown } from "@/components/elements/ai-markdown";
import type { DeskCaseHandOff } from "@/lib/graphql/desk-case";
import { handOffWatchtowerItem, type HandOffResult } from "@/lib/graphql/watchtower";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { Button } from "@trenova/shared/components/ui/button";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useT } from "@trenova/shared/i18n/use-t";
import { newIdempotencyKey } from "@trenova/shared/lib/idempotency";
import { useEffect, useRef, useState } from "react";
import { CaseFooter } from "./case-footer";

type Candidate = HandOffResult["candidates"][number];

const NO_ACTIONS: readonly { label: string; onClick: () => void }[] = [];

/**
 * A case nobody has worked: the Watchtower raised something about the record
 * and no agent has looked, so there is a reading and nothing to decide. Here
 * the person puts an agent on it. The agents that take this kind of thing on
 * their own are sent it at once; when none does, the person picks who works
 * it. What the agent offers and drafts then lands on this page to approve.
 * Marking the risk accepted is the other way out, for when nothing is needed.
 *
 * A person who opened the case from the Watchtower came to work it, so the
 * case is handed over as it opens (`autoStart`) and they wait for the plan
 * instead of asking for it. That only ever starts a case no agent has looked
 * at: one an agent already answered, or is on, is left as it is.
 */
export function CaseUnworked({
  handOff,
  canHandOff,
  locked = "",
  writing = NOTHING_WRITTEN,
  autoStart = false,
  onHanded,
  onAcceptRisk,
  actions = NO_ACTIONS,
}: {
  handOff: DeskCaseHandOff;
  /** The person owns the case, or may work it, and may work the Watchtower. */
  canHandOff: boolean;
  /** Why the reader cannot act on the case; said in place of the actions. */
  locked?: string;
  /** What the agent on the case is writing, while one is. */
  writing?: Writing;
  /** Hand the case over as it opens, when no agent has looked at it yet. */
  autoStart?: boolean;
  /** Told once an agent has it, so the page watches for what it drafts. */
  onHanded: () => void;
  /** Opens the reason for accepting the risk; null when the person may not. */
  onAcceptRisk: (() => void) | null;
  /** What a person can do about this kind of condition themselves, first to last. */
  actions?: readonly { label: string; onClick: () => void }[];
}) {
  const t = useT();
  const sending = useRef(false);
  const [pending, setPending] = useState<string | null>(null);
  const [candidates, setCandidates] = useState<Candidate[] | null>(null);
  // Who the hand-off just went to, shown until the case is read again and
  // says for itself that the agent has it, or what it said when it finished.
  const [handedTo, setHandedTo] = useState<string | null>(null);
  const answered = handOff.agent !== null;

  const send = async (agent?: Candidate) => {
    if (sending.current) {
      return;
    }
    sending.current = true;
    setPending(agent?.id ?? "any");
    try {
      const result = await handOffWatchtowerItem(handOff.itemId, agent?.id, {
        idempotencyKey: newIdempotencyKey(),
      });
      if (agent) {
        setHandedTo(agent.name);
        onHanded();
      } else if (result.runId !== null) {
        setHandedTo(result.subscribers[0]?.name ?? "");
        onHanded();
      } else {
        setCandidates(result.candidates);
      }
    } catch (error) {
      handleMutationError({ error, resourceName: "Case" });
    } finally {
      sending.current = false;
      setPending(null);
    }
  };

  // Handing over as the case opens happens once, and reads as an agent
  // already on it rather than as a button the person pressed.
  const started = useRef(false);
  const [opening, setOpening] = useState(false);
  const start = useLatestCallback(async () => {
    setOpening(true);
    await send();
    setOpening(false);
  });
  const startNow = autoStart && canHandOff && !answered && !handOff.working && handOff.note === "";
  useEffect(() => {
    if (!startNow || started.current) {
      return;
    }
    started.current = true;
    void start();
  }, [start, startNow]);

  if (handOff.working || opening || (handedTo !== null && !answered)) {
    const name = handOff.agent?.name ?? handedTo ?? "";

    return (
      <CaseWritingCard
        writing={writing}
        lead={
          name !== ""
            ? t("{0} is working this case. What it proposes shows here for you to approve.", name)
            : t("An agent is working this case. What it proposes shows here for you to approve.")
        }
      />
    );
  }

  const note =
    !canHandOff && onAcceptRisk === null && locked !== ""
      ? lockedNote(locked)
      : candidates !== null
        ? candidates.length > 0
          ? t("No agent takes this kind of thing on its own. Choose who works it.")
          : t("No agent is set up to work this. Ask about it below, or turn one on in AI control.")
        : handOff.note !== ""
          ? t("Do it yourself, hand it to another agent, or accept the risk.")
          : actions.length > 0
            ? t(
                "The Watchtower raised it and nobody has looked. Do it yourself, or hand it to an agent to lay out what could happen and draft a plan for you to approve.",
              )
            : t(
                "The Watchtower raised it and nobody has looked. Hand it to an agent and it reads the record, lays out what could happen and drafts a plan for you to approve.",
              );

  return (
    <section className="dk-cz-plan animate-rise" aria-label={t("Plan")}>
      <div className="dk-cz-plan-h">
        <div className="dk-cz-plan-k">{t("No plan")}</div>
        <div className="dk-cz-plan-row">
          <h3>
            {handOff.note !== ""
              ? t("{0} looked at this and proposed nothing.", handOff.agent?.name ?? t("An agent"))
              : t("No agent has worked this yet.")}
          </h3>
        </div>
      </div>
      {handOff.note !== "" && (
        <div className="dk-cz-said">
          <AiMarkdown content={withoutRecordIds(handOff.note)} className="text-xs" deskSubset />
        </div>
      )}
      <CaseFooter note={note}>
        {canHandOff &&
          actions.map((action) => (
            <Button
              key={action.label}
              variant="outline"
              disabled={pending !== null}
              onClick={action.onClick}
            >
              {action.label}
            </Button>
          ))}
        {canHandOff && candidates === null && (
          <Button isLoading={pending === "any"} onClick={() => void send()}>
            {handOff.note !== "" ? t("Hand it to another agent") : t("Hand it to an agent")}
          </Button>
        )}
        {onAcceptRisk && (
          <Button variant="outline" disabled={pending !== null} onClick={onAcceptRisk}>
            {t("Accept the risk…")}
          </Button>
        )}
      </CaseFooter>
      {canHandOff && candidates !== null && candidates.length > 0 && (
        <div className="dk-cz-pick" role="group" aria-label={t("Choose who works it")}>
          {candidates.map((agent) => (
            <Button
              key={agent.id}
              variant="outline"
              isLoading={pending === agent.id}
              disabled={pending !== null && pending !== agent.id}
              onClick={() => void send(agent)}
            >
              <DeskAgentTile agent={agent} size="xs" />
              {agent.name}
            </Button>
          ))}
        </div>
      )}
    </section>
  );
}

/** Why nothing can be done here by this reader, where the actions would be. */
function lockedNote(reason: string) {
  return (
    <span className="inline-flex items-start gap-2">
      <DeskIcon name="lock" size={13} className="mt-0.5 shrink-0" />
      {reason}
    </span>
  );
}
