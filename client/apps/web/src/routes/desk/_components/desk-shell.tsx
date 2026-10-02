import { EASE_SETTLE } from "@/lib/motion";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { PanelLeftOpenIcon } from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import type { CSSProperties, ReactNode } from "react";
import { DESK_RAIL_WIDTH, DESK_WORKSPACE_WIDTH } from "./desk-dimensions";
import { RAIL_KEYSHORTCUTS, RAIL_SHORTCUT } from "./desk-rail";

/** A fold arrives the way a sheet does; under reduced motion it is a cut. */
const FOLD = { duration: 0.24, ease: EASE_SETTLE } as const;
const CUT = { duration: 0 } as const;

/**
 * The Desk's chrome.
 *
 * The Desk runs outside the app shell, so this is the only frame between the
 * window edge and the work: the rail down the left, and over the room beside
 * it one strip holding who you are talking to, what the conversation is
 * called, and what can be done to it. Everything else happens in the columns
 * below the strip.
 *
 * The strip is 48px and does not scroll, so the title and the agent stay put
 * while a long conversation runs underneath them. It carries the agent's
 * wash: while an agent is working the top of the room is lit in its accent,
 * and when it stops the light goes out over about half a second. That is the
 * whole of the "something is happening" signal at this level — no spinner,
 * no bar, just the room being awake.
 *
 * The strip's left edge is where the rail comes back from. With the rail
 * open on a wide screen there is nothing there, because the rail's own
 * header holds the fold; folded, or on a screen too narrow for a rail, the
 * control to open it stands first.
 */
export function DeskShell({
  rail,
  railOpen,
  onShowRail,
  lead,
  title,
  actions,
  children,
  accent,
  working = false,
}: {
  /** The rail, already folded or unfolded; the shell only places it. */
  rail: ReactNode;
  /** Whether the rail stands open beside the room on a wide screen. */
  railOpen: boolean;
  /** Opens the rail: unfolds it on a wide screen, slides it in on a narrow one. */
  onShowRail: () => void;
  /** The agent's mark, when a conversation is open. */
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
      className="bg-desk-canvas text-foreground flex h-dvh min-h-0 w-full overflow-hidden"
      style={accent ? ({ "--agent-accent": accent } as CSSProperties) : undefined}
    >
      {rail}

      <div className="flex min-h-0 min-w-0 flex-1 flex-col">
        <header
          data-working={working}
          className={cn(
            "border-desk-hairline flex h-12 shrink-0 items-center gap-2 border-b pr-2 pl-2",
            "ui-agent-glow",
          )}
        >
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  size="icon-sm"
                  variant="ghost"
                  aria-label={t("Show the rail")}
                  aria-keyshortcuts={RAIL_KEYSHORTCUTS}
                  className={cn(
                    "text-muted-foreground hover:text-foreground shrink-0",
                    railOpen && "lg:hidden",
                  )}
                  onClick={onShowRail}
                />
              }
            >
              <PanelLeftOpenIcon className="size-4" />
            </TooltipTrigger>
            <TooltipContent side="bottom" className="flex items-center gap-2">
              {t("Show the rail")}
              <Kbd>{RAIL_SHORTCUT}</Kbd>
            </TooltipContent>
          </Tooltip>

          {lead && <span className="flex shrink-0 items-center pl-0.5">{lead}</span>}

          <div className="min-w-0 flex-1">{title}</div>

          <div className="flex shrink-0 items-center gap-0.5">{actions}</div>
        </header>

        <div className="flex min-h-0 flex-1 flex-col">{children}</div>
      </div>
    </div>
  );
}

/**
 * The rail's place on a wide screen, folding to nothing and back.
 *
 * The width animates rather than the display, so the conversation beside it
 * widens as the rail goes rather than jumping when it has gone. The rail
 * inside keeps its own width the whole way, so its rows are clipped by the
 * fold rather than reflowed by it. Folded, it is taken out of the tab order
 * and off the accessibility tree, because a column of links at zero width
 * is still a column of links to a keyboard.
 */
export function DeskRailFold({ open, children }: { open: boolean; children: ReactNode }) {
  const t = useT();
  const reduceMotion = useReducedMotion();

  return (
    <m.aside
      aria-label={t("Conversations")}
      aria-hidden={!open}
      inert={!open || undefined}
      data-state={open ? "open" : "closed"}
      initial={false}
      animate={{ width: open ? DESK_RAIL_WIDTH : 0 }}
      transition={reduceMotion ? CUT : FOLD}
      className="hidden h-full shrink-0 overflow-hidden lg:block"
    >
      <div
        className="border-desk-hairline h-full border-r"
        style={{ width: DESK_RAIL_WIDTH, minWidth: DESK_RAIL_WIDTH }}
      >
        {children}
      </div>
    </m.aside>
  );
}

/**
 * The two columns of the room: the conversation and the work it produced.
 *
 * The split is the point of the room. The conversation is where you ask and
 * the workspace is where the answer lands, and they are both visible because
 * the alternative — a pane that slides over the thing you were reading —
 * makes you choose between the question and the answer.
 *
 * The conversation takes what the workspace leaves. The workspace is capped
 * so the conversation never has to carry prose past about 70 characters,
 * and floored so a table in it is never narrower than it can be read.
 * Folded with ⌘\, it gives its width back as it goes rather than leaving a
 * gap where it was; the pane inside keeps the floor of its width so the fold
 * reads as a curtain rather than a crush.
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
  const t = useT();
  const reduceMotion = useReducedMotion();

  return (
    <div className="flex min-h-0 flex-1">
      <div className="bg-desk-column flex min-h-0 min-w-0 flex-1 flex-col">{conversation}</div>

      <m.div
        role="complementary"
        aria-label={t("Workspace")}
        aria-hidden={!workspaceOpen}
        inert={!workspaceOpen || undefined}
        data-state={workspaceOpen ? "open" : "closed"}
        initial={false}
        animate={{ width: workspaceOpen ? DESK_WORKSPACE_WIDTH : 0 }}
        transition={reduceMotion ? CUT : FOLD}
        className="hidden min-h-0 shrink-0 overflow-hidden lg:flex lg:flex-col"
      >
        <div className="border-desk-hairline flex h-full min-h-0 min-w-104 flex-1 flex-col border-l">
          {workspace}
        </div>
      </m.div>
    </div>
  );
}
