import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { ArrowUpIcon, SquareIcon } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

export type ComposerProps = {
  /** Text placed into the box from outside, such as a suggestion. */
  seed?: string;
  onSend: (content: string) => void;
  onStop: () => void;
  active: boolean;
  disabled?: boolean;
  disabledReason?: string;
  placeholder: string;
};

/**
 * Where a person types.
 *
 * Enter sends and Shift+Enter breaks a line, because that is what a chat box
 * does; the hint says so rather than making people discover it. While a reply
 * is in flight the send button becomes stop, so the one action that makes
 * sense is the one on screen.
 */
export function Composer({
  seed,
  onSend,
  onStop,
  active,
  disabled = false,
  disabledReason,
  placeholder,
}: ComposerProps) {
  const t = useT();
  const [draft, setDraft] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (seed !== undefined && seed !== "") {
      setDraft(seed);
      textareaRef.current?.focus();
    }
  }, [seed]);

  const canSend = draft.trim() !== "" && !active && !disabled;

  const submit = useCallback(() => {
    const content = draft.trim();
    if (content === "" || active || disabled) {
      return;
    }
    onSend(content);
    setDraft("");
  }, [active, disabled, draft, onSend]);

  return (
    <div className="border-border bg-background border-t px-4 py-3">
      <div className="mx-auto flex max-w-3xl flex-col gap-1.5">
        <div className="border-input bg-muted/40 focus-within:border-brand focus-within:ring-brand/20 flex flex-col rounded-lg border transition-[border-color,box-shadow] focus-within:ring-4">
          <Textarea
            ref={textareaRef}
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter" && !event.shiftKey) {
                event.preventDefault();
                submit();
              }
            }}
            placeholder={disabled ? (disabledReason ?? placeholder) : placeholder}
            disabled={disabled}
            minRows={1}
            maxRows={8}
            aria-label={t("Message the assistant")}
            className="resize-none border-0 bg-transparent px-3 py-2.5 text-sm shadow-none focus-visible:border-transparent focus-visible:ring-0 md:text-sm"
          />
          <div className="flex items-center justify-between gap-2 px-2 pb-2">
            <span className="text-muted-foreground hidden items-center gap-1 text-[11px] sm:flex">
              <Kbd>Enter</Kbd> {t("to send")} · <Kbd>Shift</Kbd>+<Kbd>Enter</Kbd>{" "}
              {t("for a new line")}
            </span>
            {active ? (
              <Button size="xs" variant="outline" onClick={onStop} className="ml-auto">
                <SquareIcon className="size-3 fill-current" />
                {t("Stop")}
              </Button>
            ) : (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      size="icon-sm"
                      className="ml-auto rounded-md"
                      onClick={submit}
                      disabled={!canSend}
                      aria-label={t("Send")}
                    />
                  }
                >
                  <ArrowUpIcon className="size-4" />
                </TooltipTrigger>
                <TooltipContent>{t("Send")}</TooltipContent>
              </Tooltip>
            )}
          </div>
        </div>
        <p className="text-muted-foreground text-center text-[11px]">
          {t(
            "The assistant reads only records you can already see, and never changes anything without your approval.",
          )}
        </p>
      </div>
    </div>
  );
}
