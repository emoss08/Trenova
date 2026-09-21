import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowLeftIcon } from "lucide-react";
import type { CSSProperties, ReactNode } from "react";
import { Link } from "react-router";

/**
 * The Desk's chrome.
 *
 * It is deliberately thin. The Desk runs outside the app shell, so this is
 * the only frame between the window edge and the work — one strip holding
 * who you are talking to, what the conversation is called, and the way
 * back. Everything else the Desk does happens in the two columns below it.
 *
 * The strip is 48px and does not scroll, so the title and the agent stay
 * put while a long conversation runs underneath them. It carries the
 * agent's wash: while an agent is working the top of the room is lit in its
 * accent, and when it stops the light goes out over about half a second.
 * That is the whole of the "something is happening" signal at this level —
 * no spinner, no bar, just the room being awake.
 */
export function DeskShell({
  lead,
  title,
  actions,
  children,
  accent,
  working = false,
}: {
  /** The agent's mark, or the Desk's own when no conversation is open. */
  lead?: ReactNode;
  title?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  /** The open agent's accent, which lights the room while it works. */
  accent?: string;
  working?: boolean;
}) {
  const t = useT();

  return (
    <div
      className="bg-desk-canvas text-foreground flex h-dvh min-h-0 w-full flex-col overflow-hidden"
      style={accent ? ({ "--agent-accent": accent } as CSSProperties) : undefined}
    >
      <header
        data-working={working}
        className={cn(
          "border-desk-hairline flex h-12 shrink-0 items-center gap-2 border-b pr-2 pl-2.5",
          "ui-agent-glow",
        )}
      >
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                size="icon-sm"
                variant="ghost"
                nativeButton={false}
                aria-label={t("Back to Trenova")}
                className="text-muted-foreground hover:text-foreground shrink-0"
                render={<Link to="/" />}
              >
                <ArrowLeftIcon className="size-4" />
              </Button>
            }
          />
          <TooltipContent side="bottom">{t("Back to Trenova")}</TooltipContent>
        </Tooltip>

        <span aria-hidden className="bg-desk-hairline mx-0.5 h-5 w-px shrink-0" />

        {lead}

        <div className="min-w-0 flex-1">{title}</div>

        <div className="flex shrink-0 items-center gap-0.5">{actions}</div>
      </header>

      <div className="flex min-h-0 flex-1 flex-col">{children}</div>
    </div>
  );
}

/**
 * The two columns: the conversation and the work it produced.
 *
 * The split is the point of the room. The conversation is where you ask and
 * the workspace is where the answer lands, and they are both permanently
 * visible because the alternative — a pane that slides over the thing you
 * were reading — makes you choose between the question and the answer.
 *
 * The conversation is the narrower half on purpose. It is a column of prose
 * and prose is unreadable past about 70 characters, so the extra width goes
 * to the tables and drafts that can use it. Folded away, it takes the room
 * back rather than leaving a gap where the workspace was.
 */
export function DeskColumns({
  conversation,
  workspace,
  workspaceOpen,
}: {
  conversation: ReactNode;
  workspace: ReactNode;
  workspaceOpen: boolean;
}) {
  return (
    <div className="flex min-h-0 flex-1">
      <div
        className={cn(
          "bg-desk-column border-desk-hairline flex min-h-0 min-w-0 flex-col",
          workspaceOpen ? "w-full border-r lg:w-[clamp(26rem,38%,34rem)]" : "w-full",
        )}
      >
        {conversation}
      </div>

      {workspaceOpen && (
        <div className="animate-materialise hidden min-h-0 min-w-0 flex-1 lg:flex lg:flex-col">
          {workspace}
        </div>
      )}
    </div>
  );
}

/**
 * The live indicator: one dot in the agent's accent that breathes while
 * work is running and is simply absent when it is not.
 *
 * This is the only looping animation in the product, and it earns the
 * exception the same way a heartbeat monitor does — it loops because the
 * thing it describes is still going, and it stops the moment that stops.
 */
export function WorkingDot({ working, className }: { working: boolean; className?: string }) {
  if (!working) {
    return null;
  }

  return (
    <span
      aria-hidden
      className={cn(
        "animate-breathe size-1.5 rounded-full",
        "bg-[var(--agent-accent,var(--brand))]",
        className,
      )}
    />
  );
}
