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
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { newIdempotencyKey } from "@trenova/shared/lib/idempotency";
import { useRef, useState } from "react";
import { CaseFooter } from "./case-footer";
import { casePlayAgainLabel, casePlayLabel } from "./case-play";
import { CasePlayDialog } from "./case-play-dialog";

type Candidate = HandOffResult["candidates"][number];

const NO_ACTIONS: readonly { label: string; onClick: () => void }[] = [];

/**
 * A case nobody has worked: the Watchtower raised something about the record
 * and no agent has looked, so there is a reading and nothing to decide. No
 * agent is put on it by opening it: an agent run costs the organization, and
 * most of what the Watchtower raises a person settles faster than a model
 * reads the record.
 *
 * What the person sees first is what they can do themselves: the condition's
 * first move (`handOff.play`), which they read and send under their own name
 * from `CasePlayDialog` with no agent set up, and anything else the page
 * offers for the kind (`actions`). Once a move has been made the case says
 * who was asked what, and when (`handOff.request`), so the next person does
 * not ask twice without knowing.
 *
 * An agent is something they ask for: the ones that take this kind of thing
 * on their own are sent it, and when none does the person picks who works
 * it. What the agent proposes lands on this page to approve. Marking the
 * risk accepted is the other way out, for when nothing is needed.
 *
 * An item an agent raised because it needs a person (`agentRaised`) is not
 * handed to another agent: the question is the person's to answer.
 */
export function CaseUnworked({
  handOff,
  canHandOff,
  locked = "",
  writing = NOTHING_WRITTEN,
  agentRaised = false,
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
  /** An agent raised this to ask for a person, so no agent is offered for it. */
  agentRaised?: boolean;
  /** Told once an agent has it or a move was made, so the page reads the case again. */
  onHanded: () => void;
  /** Opens the reason for accepting the risk; null when the person may not. */
  onAcceptRisk: (() => void) | null;
  /** What a person can do about this kind of condition themselves, first to last. */
  actions?: readonly { label: string; onClick: () => void }[];
}) {
  const t = useT();
  const now = useNow("minute");
  const sending = useRef(false);
  const [pending, setPending] = useState<string | null>(null);
  const [candidates, setCandidates] = useState<Candidate[] | null>(null);
  // Who the hand-off just went to, shown until the case is read again and
  // says for itself that the agent has it, or what it said when it finished.
  const [handedTo, setHandedTo] = useState<string | null>(null);
  const [playing, setPlaying] = useState(false);
  const answered = handOff.agent !== null;
  const play = handOff.play ?? null;
  const request = handOff.request ?? null;

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

  if (handOff.working || (handedTo !== null && !answered)) {
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
        : agentRaised
          ? t(
              "An agent raised this because it needs you. Do it yourself, ask about it below, or accept the risk.",
            )
          : request !== null
            ? t("Waiting on an answer. You can ask again, do it yourself, or accept the risk.")
            : handOff.note !== ""
              ? t("Do it yourself, hand it to another agent, or accept the risk.")
              : play !== null
                ? t(
                    "The Watchtower raised it and nobody has looked. Start with the first step: you read it and send it yourself.",
                  )
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
        <div className="dk-cz-plan-k">{request !== null ? t("Asked") : t("No plan")}</div>
        <div className="dk-cz-plan-row">
          <h3>
            {request !== null
              ? withoutRecordIds(request.summary)
              : handOff.note !== ""
                ? t(
                    "{0} looked at this and proposed nothing.",
                    handOff.agent?.name ?? t("An agent"),
                  )
                : t("No agent has worked this yet.")}
          </h3>
        </div>
        {request !== null && (
          <div className="text-dsk-sm text-dsk-subtle">
            {request.by !== ""
              ? t("{0}, {1}", request.by, formatRelativeTime(request.at - now))
              : formatRelativeTime(request.at - now)}
          </div>
        )}
      </div>
      {handOff.note !== "" && (
        <div className="dk-cz-said">
          <AiMarkdown content={withoutRecordIds(handOff.note)} className="text-xs" deskSubset />
        </div>
      )}
      <CaseFooter note={note}>
        {canHandOff && play !== null && (
          <Button disabled={pending !== null} onClick={() => setPlaying(true)}>
            {request !== null ? casePlayAgainLabel(play, t) : casePlayLabel(play, t)}
          </Button>
        )}
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
        {canHandOff && !agentRaised && candidates === null && (
          <Button
            variant={play !== null ? "secondary" : "default"}
            isLoading={pending === "any"}
            onClick={() => void send()}
          >
            {handOff.note !== "" ? t("Hand it to another agent") : t("Hand it to an agent")}
          </Button>
        )}
        {onAcceptRisk && (
          <Button variant="outline" disabled={pending !== null} onClick={onAcceptRisk}>
            {t("Accept the risk…")}
          </Button>
        )}
      </CaseFooter>
      {playing && play !== null && (
        <CasePlayDialog
          itemId={handOff.itemId}
          play={play}
          onOpenChange={setPlaying}
          onSent={onHanded}
        />
      )}
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
