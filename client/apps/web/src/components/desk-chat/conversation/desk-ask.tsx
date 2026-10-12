import {
  answersInComposer,
  type AskField,
  type ThreadAsk,
  type ThreadAskRequest,
} from "@/components/assistant/ask-requests";
import {
  isKeyForScope,
  isWithinDialog,
  isWithinPopup,
  mayMoveFocusWithin,
  numberKeyIndex,
} from "@/lib/dom";
import type { AssistantEntityRef } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useId, useMemo, useRef } from "react";
import { DeskMentionText } from "../composer/desk-mentions";
import { DeskCard } from "../desk-card";
import { DeskIcon } from "../desk-icons";
import { DeskAskForm } from "./desk-ask-fields";

/** The most choices a number key reaches. */
const MAX_KEYED_OPTIONS = 9;

/** Sends an answer, with the records it names. */
export type DeskAskAnswer = (text: string, mentions?: readonly AssistantEntityRef[]) => void;

/**
 * What the composer invites while a question waits: the shape of the value
 * the agent asked for when the composer is where it is typed, and otherwise
 * that the answer is above, because any message still answers it.
 */
export function deskAskPlaceholder(request: ThreadAskRequest, t: TranslateFn): string {
  if (answersInComposer(request)) {
    return request.otherHint !== "" ? request.otherHint : t("Type your answer…");
  }
  return t("Answer above, or reply…");
}

/**
 * A question the agent needs answered before it can go on, waiting above the
 * composer.
 *
 * A model that lacks a required value has three options: guess it, refuse, or
 * ask. It used to ask in prose — "common choices are 7, 14 or 30" — which put
 * the work back on the reader to retype one of them, and left the model free to
 * pick for them on the next turn. The choices are controls now, so answering is
 * one click (or its number key) and the answer is exact; each choice's line of
 * consequence is in view under it, not behind a hover. A value the agent needs
 * typed in a shape (a number, a day, a record) gets the control for that shape,
 * and a few values that complete one change get a small form.
 *
 * A value in words is typed in the composer, which says what to type: the
 * question draws no second box. Any message answers the question, which is the
 * whole trick: an answer is sent as an ordinary message, so nothing new has to
 * be stored, the thread reads the way it did before, and a conversation that
 * was answered by typing is indistinguishable from one answered by clicking.
 */
export function DeskAskDock({
  request,
  disabled,
  disabledNote,
  compact = false,
  onAnswer,
}: {
  request: ThreadAskRequest;
  /** No answer can go now: a reply is being written, the conversation is offline, or it waits. */
  disabled: boolean;
  /** Why no answer can go now, when it is not that a reply is being written. */
  disabledNote?: string;
  compact?: boolean;
  onAnswer: DeskAskAnswer;
}) {
  const t = useT();
  const id = useId();
  const ref = useRef<HTMLDivElement>(null);
  const focusedFor = useRef<string | null>(null);
  const { options, fields } = request;
  const typed = !answersInComposer(request) && request.allowOther && fields.length === 0;
  const keyed = !disabled && options.length > 0;
  const own = useMemo<AskField[]>(
    () => [
      {
        ...request.input,
        key: "value",
        label: request.otherHint !== "" ? request.otherHint : t("Your own answer"),
        hint: "",
        optional: false,
      },
    ],
    [request.input, request.otherHint, t],
  );

  // A choice is a number key away while focus is not in a field, a menu or a
  // dialog, and is somewhere in this conversation or nowhere at all.
  useEffect(() => {
    if (!keyed) {
      return;
    }
    const onKey = (event: KeyboardEvent) => {
      const target = event.target;
      if (event.repeat || isWithinDialog(target) || isWithinPopup(target)) {
        return;
      }
      if (!isKeyForScope(target, ref.current?.closest(".dk-room") ?? null)) {
        return;
      }
      const index = numberKeyIndex(event, Math.min(options.length, MAX_KEYED_OPTIONS));
      if (index !== -1) {
        event.preventDefault();
        onAnswer(options[index].value);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [keyed, onAnswer, options]);

  // Typed in the composer, the answer starts where the person already is.
  // Otherwise the question takes focus once, when it can first be answered,
  // unless the person is in the middle of typing something.
  useEffect(() => {
    const card = ref.current;
    if (
      disabled ||
      card === null ||
      answersInComposer(request) ||
      focusedFor.current === request.callId
    ) {
      return;
    }
    focusedFor.current = request.callId;
    if (!mayMoveFocusWithin(card.closest(".dk-room"))) {
      return;
    }
    const first =
      card.querySelector<HTMLElement>("input:not(:disabled)") ??
      card.querySelector<HTMLElement>("button:not(:disabled)");
    first?.focus();
  }, [disabled, request]);

  const sub = disabled
    ? (disabledNote ?? t("You can answer once the reply finishes."))
    : fields.length > 0
      ? undefined
      : answersInComposer(request)
        ? options.length > 0
          ? t("Pick one, or type your own answer below.")
          : t("Type your answer below.")
        : undefined;

  return (
    <div ref={ref} className="dk-ec-dockcard">
      <DeskCard
        tone="ask"
        icon="chat"
        compact={compact}
        label={t("Question from the agent")}
        title={request.question}
        titleId={`${id}-q`}
        sub={sub}
      >
        {options.length > 0 && (
          <div className="dk-ask-opts" role="group" aria-labelledby={`${id}-q`}>
            {options.map((option, index) => {
              const detailId = option.detail !== "" ? `${id}-d${index}` : undefined;
              const key = index < MAX_KEYED_OPTIONS ? String(index + 1) : undefined;
              return (
                <Button
                  key={option.value}
                  variant="bare"
                  size="bare"
                  className={cn("dk-ask-opt items-start", detailId && "dk-d")}
                  disabled={disabled}
                  aria-label={option.label}
                  aria-describedby={detailId}
                  aria-keyshortcuts={keyed ? key : undefined}
                  onClick={() => onAnswer(option.value)}
                >
                  {keyed && key && (
                    <span className="dk-kbd" aria-hidden>
                      {key}
                    </span>
                  )}
                  <span className="dk-ask-optt">
                    <b>{option.label}</b>
                    {detailId && <em id={detailId}>{option.detail}</em>}
                  </span>
                </Button>
              );
            })}
          </div>
        )}
        {fields.length > 0 && (
          <DeskAskForm fields={fields} disabled={disabled} onAnswer={onAnswer} />
        )}
        {typed && <DeskAskForm fields={own} single disabled={disabled} onAnswer={onAnswer} />}
      </DeskCard>
    </div>
  );
}

/** What the person answered, as they would know it: the choice's label when they picked one. */
function answerText(ask: ThreadAsk): string {
  const content = ask.answer?.content.trim() ?? "";
  return ask.request.options.find((option) => option.value === content)?.label ?? content;
}

/**
 * Where a question was asked, in the conversation. While it waits, a line
 * says so and points at the question above the composer, where it is
 * answered; nothing on it can be pressed twice over. Once answered, the
 * question and what the person said to it, so reading back shows what was
 * settled and how.
 */
export function DeskAskRecord({
  ask,
  docked,
}: {
  ask: ThreadAsk;
  /** The question waits above the composer. */
  docked: boolean;
}) {
  const t = useT();
  const { request, answer } = ask;

  return (
    <div className="dk-ask-rec">
      <span className="dk-ask-rec-ic">
        <DeskIcon name="chat" size={13} stroke={2} />
      </span>
      <div className="dk-ask-rec-b">
        <span className="dk-ask-rec-q">
          <b>{t("Asked")}</b> {request.question}
        </span>
        {answer !== null ? (
          <span className="dk-ask-rec-a">
            <DeskIcon name="enter" size={12} stroke={2} />
            <span>
              <span className="sr-only">{t("You answered")} </span>
              {answer.mentions && answer.mentions.length > 0 ? (
                <DeskMentionText text={answer.content} mentions={answer.mentions} />
              ) : (
                answerText(ask)
              )}
            </span>
          </span>
        ) : (
          <span className="dk-ask-rec-s">
            {docked ? t("Waiting for your answer below") : t("Not answered yet")}
          </span>
        )}
      </div>
    </div>
  );
}
