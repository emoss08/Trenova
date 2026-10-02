import { EASE_SETTLE } from "@/lib/motion";
import {
  MAX_WORKSPACE_SIZE,
  MIN_WORKSPACE_SIZE,
  clampWorkspaceSize,
} from "@/stores/desk-store";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@trenova/shared/components/ui/resizable";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { PanelLeftOpenIcon } from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import type { CSSProperties, ReactNode } from "react";
import { DESK_RAIL_STRIP_WIDTH, DESK_RAIL_WIDTH } from "./desk-dimensions";
import { RAIL_KEYSHORTCUTS, RAIL_SHORTCUT } from "./desk-rail";

/** A fold arrives the way a sheet does; under reduced motion it is a cut. */
const FOLD = { duration: 0.24, ease: EASE_SETTLE } as const;
const CUT = { duration: 0 } as const;

/**
 * The Desk's chrome.
 *
 * The Desk runs outside the app shell, so this is the only frame between the
 * window edge and the work: the rail down the left, and over the room beside
 * it one strip of 44px holding who you are talking to, what the conversation
 * is called, and what can be done to it. It carries the agent's wash: while
 * an agent is working the top of the room is lit in its accent, and when it
 * stops the light goes out. That is the whole of the "something is
 * happening" signal at this level.
 *
 * With the rail open on a wide screen the strip starts at the agent; folded
 * to its strip of places, or on a screen too narrow for a rail, the control
 * to open it stands first.
 */
export function DeskShell({
  rail,
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
  /** Whether the rail stands open beside the room on a wide screen; the strip holds the unfold there. */
  railOpen?: boolean;
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
            "border-desk-hairline flex h-11 shrink-0 items-center gap-1.5 border-b pr-2 pl-2",
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
                    "lg:hidden",
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

          {lead && <span className="flex shrink-0 items-center">{lead}</span>}

          <div className="flex min-w-0 flex-1 items-center">{title}</div>

          <div className="flex shrink-0 items-center gap-0.5">{actions}</div>
        </header>

        <div className="flex min-h-0 flex-1 flex-col">{children}</div>
      </div>
    </div>
  );
}

/**
 * The rail's place on a wide screen: open, or folded to a strip of its places.
 *
 * The width animates rather than the display, so the room beside it widens
 * as the rail goes rather than jumping when it has gone. Both forms of the
 * rail are rendered inside; which one shows is decided by the width, so the
 * fold reads as the list sliding away behind the strip that stays.
 */
export function DeskRailFold({
  open,
  rail,
  strip,
}: {
  open: boolean;
  /** The rail at full width. */
  rail: ReactNode;
  /** The rail folded to its strip of places. */
  strip: ReactNode;
}) {
  const t = useT();
  const reduceMotion = useReducedMotion();

  return (
    <m.aside
      aria-label={t("Conversations")}
      data-state={open ? "open" : "collapsed"}
      initial={false}
      animate={{ width: open ? DESK_RAIL_WIDTH : DESK_RAIL_STRIP_WIDTH }}
      transition={reduceMotion ? CUT : FOLD}
      className="border-desk-hairline bg-desk-rail relative hidden h-full shrink-0 overflow-hidden border-r lg:block"
    >
      <div
        aria-hidden={!open}
        inert={!open || undefined}
        className={cn(
          "absolute inset-y-0 left-0 transition-opacity duration-200",
          open ? "opacity-100" : "pointer-events-none opacity-0",
        )}
        style={{ width: DESK_RAIL_WIDTH, minWidth: DESK_RAIL_WIDTH }}
      >
        {rail}
      </div>
      <div
        aria-hidden={open}
        inert={open || undefined}
        className={cn(
          "absolute inset-y-0 left-0 transition-opacity duration-200",
          open ? "pointer-events-none opacity-0" : "opacity-100",
        )}
        style={{ width: DESK_RAIL_STRIP_WIDTH, minWidth: DESK_RAIL_STRIP_WIDTH }}
      >
        {strip}
      </div>
    </m.aside>
  );
}

/**
 * The two columns of the room: the conversation and the work it produced,
 * with a handle between them.
 *
 * The split is the point of the room. The conversation is where you ask and
 * the workspace is where the answer lands, and they are both visible because
 * the alternative — a pane that slides over the thing you were reading —
 * makes you choose between the question and the answer. The handle lets a
 * person reading a wide table give it the room, and the share they settle
 * on is remembered. Folded with ⌘\, the workspace gives its width back.
 */
export function DeskColumns({
  conversation,
  workspace,
  workspaceOpen,
  workspaceSize,
  onWorkspaceResize,
}: {
  conversation: ReactNode;
  workspace: ReactNode;
  workspaceOpen: boolean;
  /** The workspace's share of the columns, in percent. */
  workspaceSize: number;
  onWorkspaceResize: (size: number) => void;
}) {
  const t = useT();
  const share = clampWorkspaceSize(workspaceSize);

  if (!workspaceOpen) {
    return (
      <div className="flex min-h-0 flex-1">
        <div className="bg-desk-column flex min-h-0 min-w-0 flex-1 flex-col">{conversation}</div>
      </div>
    );
  }

  return (
    <ResizablePanelGroup
      orientation="horizontal"
      className="min-h-0 flex-1"
      onLayoutChanged={(layout) => {
        const next = layout["desk-workspace"];
        if (typeof next === "number") {
          onWorkspaceResize(next);
        }
      }}
    >
      <ResizablePanel
        id="desk-conversation"
        defaultSize={`${100 - share}`}
        minSize={`${100 - MAX_WORKSPACE_SIZE}`}
        className="bg-desk-column flex min-h-0 min-w-0 flex-col"
      >
        {conversation}
      </ResizablePanel>
      <ResizableHandle
        aria-label={t("Resize the workspace")}
        className="bg-desk-hairline data-[separator=hover]:bg-foreground/20 data-[separator=active]:bg-foreground/30 transition-colors"
      />
      <ResizablePanel
        id="desk-workspace"
        defaultSize={`${share}`}
        minSize={`${MIN_WORKSPACE_SIZE}`}
        maxSize={`${MAX_WORKSPACE_SIZE}`}
        className="animate-materialise hidden min-h-0 min-w-0 lg:flex lg:flex-col"
        role="complementary"
        aria-label={t("Workspace")}
      >
        {workspace}
      </ResizablePanel>
    </ResizablePanelGroup>
  );
}
