import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowUpIcon, SquareIcon, XIcon } from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import { useCallback, useRef, useState } from "react";
import type { Suggestion } from "./suggestions";

export type ComposerProps = {
  onSend: (content: string) => void;
  onStop: () => void;
  active: boolean;
  disabled?: boolean;
  disabledReason?: string;
  placeholder: string;
  /** Opening questions shown above the box until one is used or dismissed. */
  suggestions?: Suggestion[];
  onDismissSuggestion?: (prompt: string) => void;
  compact?: boolean;
};

/**
 * Where a person types.
 *
 * Enter sends and Shift+Enter breaks a line, because that is what a chat box
 * does; Cmd+Enter sends too, for people who expect it. While a reply is in
 * flight the send button becomes stop, so the one action that makes sense is
 * the one on screen.
 */
export function Composer({
  onSend,
  onStop,
  active,
  disabled = false,
  disabledReason,
  placeholder,
  suggestions = [],
  onDismissSuggestion,
  compact = false,
}: ComposerProps) {
  const t = useT();
  const [draft, setDraft] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);

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
    <div
      className={cn(
        "border-border/70 bg-background border-t",
        compact ? "px-3 py-2.5" : "px-4 py-3",
      )}
    >
      <div className={cn("mx-auto flex flex-col gap-2", !compact && "max-w-3xl")}>
        <AnimatePresence initial={false}>
          {suggestions.length > 0 && draft === "" && !active && (
            <m.div
              key="suggestions"
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: "auto" }}
              exit={{ opacity: 0, height: 0 }}
              className="flex flex-wrap gap-1.5 overflow-hidden"
            >
              {suggestions.map((suggestion) => (
                <span
                  key={suggestion.prompt}
                  className="bg-card text-muted-foreground hover:text-foreground border-border/70 group inline-flex items-center rounded-full border text-xs transition-colors"
                >
                  <button
                    type="button"
                    onClick={() => onSend(suggestion.prompt)}
                    className="hover:bg-muted rounded-l-full py-1 pr-1 pl-2.5"
                  >
                    {t(suggestion.label)}
                  </button>
                  {onDismissSuggestion && (
                    <button
                      type="button"
                      aria-label={t("Dismiss suggestion")}
                      onClick={() => onDismissSuggestion(suggestion.prompt)}
                      className="hover:bg-muted rounded-r-full py-1 pr-1.5 pl-0.5 opacity-60 hover:opacity-100"
                    >
                      <XIcon className="size-3" />
                    </button>
                  )}
                </span>
              ))}
            </m.div>
          )}
        </AnimatePresence>

        <div className="border-input bg-muted/40 focus-within:border-primary focus-within:ring-primary/20 flex flex-col rounded-xl border transition-[border-color,box-shadow] focus-within:ring-4">
          <Textarea
            ref={textareaRef}
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter" && (event.metaKey || event.ctrlKey || !event.shiftKey)) {
                event.preventDefault();
                submit();
              }
            }}
            placeholder={disabled ? (disabledReason ?? placeholder) : placeholder}
            disabled={disabled}
            minRows={1}
            maxRows={compact ? 5 : 8}
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
                      className="ml-auto rounded-lg"
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
        {!compact && (
          <p className="text-muted-foreground text-center text-[11px]">
            {t(
              "The assistant reads only records you can already see, and never changes anything without your approval.",
            )}
          </p>
        )}
      </div>
    </div>
  );
}
