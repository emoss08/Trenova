import type { AskField, AskQuestionItem } from "@/components/assistant/ask-requests";
import { askQuestionsAnswer, type AskPick, type AskValue } from "@/lib/ask-answer";
import type { AssistantEntityRef } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useId, useState, type FormEvent } from "react";
import { DeskCardButton } from "../desk-card";
import { DeskIcon } from "../desk-icons";
import { DeskAskFieldRow } from "./desk-ask-fields";

/**
 * Several questions the agent asked at once, answered together.
 *
 * An agent that needed three rates, or which batches to plan and whether to
 * include delayed loads, used to ask one question a turn: answer, wait for
 * the reply, answer again. Each question here takes its own answer, a choice
 * of its options or a value of the person's own, and Send waits until every
 * one has one. The answers leave as one ordinary message, a line per
 * question, so the agent reads which answer is which.
 */
export function DeskAskQuestions({
  items,
  disabled,
  onAnswer,
}: {
  items: readonly AskQuestionItem[];
  disabled: boolean;
  onAnswer: (text: string, mentions: readonly AssistantEntityRef[]) => void;
}) {
  const t = useT();
  const id = useId();
  const [picks, setPicks] = useState<(AskPick | undefined)[]>(() => items.map(() => undefined));
  const answer = askQuestionsAnswer(items, picks);

  const pick = (index: number, next: AskPick) =>
    setPicks((current) => current.map((value, at) => (at === index ? next : value)));

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (answer === null || disabled) {
      return;
    }
    onAnswer(answer.text, answer.mentions);
  };

  return (
    <form className="dk-ask-form dk-ask-qs" onSubmit={submit} noValidate>
      {items.map((item, index) => (
        <DeskAskQuestionRow
          key={item.question}
          id={`${id}-${index}`}
          number={index + 1}
          item={item}
          pick={picks[index]}
          disabled={disabled}
          onPick={(next) => pick(index, next)}
        />
      ))}
      <div className="dk-ask-send">
        <DeskCardButton ink type="submit" disabled={disabled || answer === null}>
          <DeskIcon name="enter" size={12} stroke={2} />
          {t("Send answers")}
        </DeskCardButton>
      </div>
    </form>
  );
}

/** One question of several: its choices as toggles, and a box for a value of the person's own. */
function DeskAskQuestionRow({
  id,
  number,
  item,
  pick,
  disabled,
  onPick,
}: {
  id: string;
  number: number;
  item: AskQuestionItem;
  pick: AskPick | undefined;
  disabled: boolean;
  onPick: (pick: AskPick) => void;
}) {
  const t = useT();
  const picked = pick !== undefined && "option" in pick ? pick.option : null;
  const own: AskValue = pick !== undefined && "own" in pick ? pick.own : null;
  const field: AskField = {
    ...item.input,
    key: "value",
    label: item.otherHint !== "" ? item.otherHint : t("Your own answer"),
    hint: "",
    optional: false,
  };

  return (
    <fieldset className="dk-ask-q" aria-labelledby={`${id}-q`}>
      <legend className="sr-only">{item.question}</legend>
      <span id={`${id}-q`} className="dk-ask-qt">
        <span className="dk-ask-qn" aria-hidden>
          {number}
        </span>
        {item.question}
      </span>
      {item.options.length > 0 && (
        <div className="dk-ask-opts" role="group" aria-labelledby={`${id}-q`}>
          {item.options.map((option, index) => {
            const detailId = option.detail !== "" ? `${id}-d${index}` : undefined;
            return (
              <Button
                key={option.value}
                type="button"
                variant="bare"
                size="bare"
                className={cn("dk-ask-opt items-start", detailId && "dk-d")}
                disabled={disabled}
                aria-pressed={picked === option.value}
                aria-describedby={detailId}
                onClick={() => onPick({ option: option.value })}
              >
                <span className="dk-ask-optt">
                  <b>{option.label}</b>
                  {detailId && <em id={detailId}>{option.detail}</em>}
                </span>
              </Button>
            );
          })}
        </div>
      )}
      {item.allowOther && (
        <DeskAskFieldRow
          id={`${id}-own`}
          field={field}
          value={own}
          disabled={disabled}
          onChange={(value) => onPick({ own: value })}
        />
      )}
    </fieldset>
  );
}
