import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { CornerDownLeftIcon } from "lucide-react";
import { useState } from "react";
import type { ThreadAskRequest } from "./ask-requests";

/**
 * A question the assistant needs answered before it can go on.
 *
 * A model that lacks a required value has three options: guess it, refuse, or
 * ask. It used to ask in prose — "common choices are 7, 14 or 30" — which put
 * the work back on the reader to retype one of them, and left the model free to
 * pick for them on the next turn. The choices are controls now, so answering is
 * one click and the answer is exact.
 *
 * "Your own" is always offered unless the value is genuinely closed, because a
 * model's idea of the common cases is a guess about the reader's work. The
 * person asking for a 45-day window should not have to argue with a list.
 *
 * An answer is sent as an ordinary message, which is the whole trick: nothing
 * new has to be stored, the thread reads the way it did before, and a
 * conversation that was answered by typing is indistinguishable from one
 * answered by clicking.
 */
export function ChoicePrompt({
  request,
  answered,
  onAnswer,
}: {
  request: ThreadAskRequest;
  /** True once a later message has arrived: the question has had its answer. */
  answered: boolean;
  onAnswer: (value: string) => void;
}) {
  const t = useT();
  const [own, setOwn] = useState("");

  const submitOwn = () => {
    const value = own.trim();
    if (value === "") {
      return;
    }
    setOwn("");
    onAnswer(value);
  };

  return (
    <div
      className={cn(
        "border-border/70 flex flex-col gap-2.5 rounded-lg border px-3 py-2.5",
        // Answered, it stays on screen as the record of what was asked, but it
        // stops inviting a second answer to a question already settled.
        answered && "opacity-60",
      )}
    >
      <p className="text-sm">{request.question}</p>

      {request.options.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {request.options.map((option) => (
            <Button
              key={option.value}
              type="button"
              size="sm"
              variant="outline"
              disabled={answered}
              onClick={() => onAnswer(option.value)}
              title={option.detail === "" ? undefined : option.detail}
            >
              {option.label}
            </Button>
          ))}
        </div>
      )}

      {request.allowOther && !answered && (
        <div className="flex items-center gap-1.5">
          <Input
            value={own}
            onChange={(event) => setOwn(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                event.preventDefault();
                submitOwn();
              }
            }}
            placeholder={request.otherHint === "" ? t("Your own answer") : request.otherHint}
            aria-label={t("Answer in your own words")}
            className="h-8 text-sm"
          />
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={own.trim() === ""}
            onClick={submitOwn}
            aria-label={t("Send this answer")}
          >
            <CornerDownLeftIcon className="size-3.5" />
          </Button>
        </div>
      )}
    </div>
  );
}
