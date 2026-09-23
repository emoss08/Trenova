import { currentActivity, stepsFromSegments } from "@/components/assistant/activity";
import { useT } from "@trenova/shared/i18n/use-t";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { cn } from "@trenova/shared/lib/utils";
import { AssistantProse } from "@/components/assistant/message-items";
import type { TurnState } from "@/components/assistant/turn-stream";
import { LayoutPanelLeftIcon, SquareIcon } from "lucide-react";

/**
 * The answer to a quick question, inside the palette. It streams the way a
 * thread's turn does and, once a thread exists, offers to continue in the
 * Desk. What it does not do is pretend to be a conversation: one question,
 * one answer, and a door to the place where the rest can happen.
 */
export function AskAnswerCard({
  question,
  turn,
  onAsk,
  onStop,
  onOpenInDesk,
  className,
}: {
  question: string;
  turn: TurnState | null;
  onAsk: () => void;
  onStop: () => void;
  onOpenInDesk: () => void;
  className?: string;
}) {
  const t = useT();
  const active =
    turn !== null &&
    (turn.status === "guarding" || turn.status === "streaming" || turn.status === "working");
  const text =
    turn?.segments
      .flatMap((segment) => (segment.kind === "text" ? [segment.text] : []))
      .join("\n\n") ?? "";
  // What the agent did or is doing, in the words of what it did: an opened
  // page reads as opened, never as a record looked up.
  const activity = turn ? currentActivity(stepsFromSegments(turn.segments), t) : null;
  const canOpen = turn !== null && (turn.thread !== null || turn.result !== null);
  const answered = turn !== null && turn.userContent === question;

  return (
    <section
      aria-label={t("Ask the assistant")}
      className={cn("border-border bg-card mx-2 mt-2 rounded-lg border", className)}
    >
      <header className="flex items-center gap-2 px-3 py-2 text-xs">
        <AssistMark className="text-muted-foreground size-3.5" />
        <span className="text-muted-foreground min-w-0 flex-1 truncate">
          {answered ? question : t("Ask: {0}", question)}
        </span>
        {!answered && (
          <span className="text-muted-foreground flex items-center gap-1">
            <Kbd>Enter</Kbd> {t("to ask")}
          </span>
        )}
        {active && (
          <Button
            size="xs"
            variant="ghost"
            onClick={onStop}
            aria-label={t("Stop")}
            className="text-muted-foreground"
          >
            <SquareIcon className="size-3 fill-current" />
          </Button>
        )}
      </header>

      {answered && turn && (
        <div className="border-border flex flex-col gap-2 border-t px-3 py-2.5">
          {turn.status === "guarding" && (
            <p className="text-muted-foreground flex items-center gap-2 text-xs">
              <Spinner className="size-3" /> {t("Checking the question…")}
            </p>
          )}
          {activity && (
            <p className="text-muted-foreground truncate text-xs">
              {activity.phrase}
              {activity.detail !== "" && (
                <span className="text-foreground-subtle"> · {activity.detail}</span>
              )}
            </p>
          )}
          {text !== "" && <AssistantProse content={text} streaming={turn.status === "streaming"} />}
          {turn.status === "working" && text === "" && (
            <p className="text-muted-foreground flex items-center gap-2 text-xs">
              <Spinner className="size-3" /> {t("Working…")}
            </p>
          )}
          {turn.status === "refused" && turn.refusal && (
            <p className="text-warning-foreground text-sm">{turn.refusal.message}</p>
          )}
          {turn.status === "error" && turn.error && (
            <p className="text-danger-foreground text-sm">{turn.error}</p>
          )}
          {turn.status === "error" && (
            <div>
              <Button size="xs" variant="outline" onClick={onAsk}>
                {t("Ask again")}
              </Button>
            </div>
          )}
          {canOpen && (
            <div className="flex items-center gap-2 pt-1">
              <Button size="xs" variant="outline" onClick={onOpenInDesk}>
                <LayoutPanelLeftIcon className="size-3.5" />
                {t("Open in Desk")}
              </Button>
              <span className="text-muted-foreground text-xs">
                {t("Continue the conversation with room for what it produces.")}
              </span>
            </div>
          )}
        </div>
      )}
    </section>
  );
}
