import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { usePendingValue } from "@trenova/shared/hooks/use-pending-value";
import { formatUnixMonthDay } from "@trenova/shared/lib/date";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useRef, useState, type KeyboardEvent } from "react";
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { DeskTip } from "@/components/desk-chat/desk-tip";
import { StreamedText } from "@/components/streamed-text";
import type { ChapterEntry } from "./chapter-work";
import "./desk-chapters.css";

/** The longest title the server keeps for a chapter. */
export const MAX_CHAPTER_TITLE = 60;

/** How recently a title must have been written to type itself in when it first shows. */
export const FRESH_TITLE_SECONDS = 15;

export type ChapterActions = {
  rename: (entry: ChapterEntry, title: string) => void;
  /** Carries the chapter on with the conversation's own agent in a conversation of its own. */
  continueAlone?: (entry: ChapterEntry) => void;
  /** Hands the chapter to another of the person's agents. */
  handOff?: (entry: ChapterEntry, agentId: string) => void;
  /** Ends the chapter where it starts; its messages join the chapter above. */
  merge: (entry: ChapterEntry) => void;
  /** The agents a chapter may be handed to. */
  agents: readonly AgentChoice[];
  /** A hand-off or merge is on its way; the menu waits for it. */
  busy: boolean;
};

/**
 * A chapter's heading, above the question that opens it: its number, its
 * title and a rule out to the margin. Double-clicking the title, or Enter or
 * F2 on it, turns it into a field: Enter or leaving it keeps the new title,
 * Escape keeps the old one. The menu beside it renames the chapter, carries
 * it on in a conversation of its own or hands it to another agent, and
 * merges it into the chapter above.
 */
export function DeskChapterHeading({
  entry,
  actions,
  timezone,
}: {
  entry: ChapterEntry;
  actions: ChapterActions;
  timezone: string;
}) {
  const t = useT();
  const { chapter } = entry;
  const [editing, setEditing] = useState(false);
  const settled = useRef(false);
  const [shown, setSent] = usePendingValue(chapter.title);
  const saved = entry.chapter.startSequence !== null;
  // A title the model has just written types itself in, as the briefing
  // does: when the heading first shows moments after it was named, and when
  // the model names it again while it is on screen. A title the person typed,
  // and one that was there before the page opened, shows whole.
  const [streaming, setStreaming] = useState(() => ({
    title: chapter.title,
    animate: Date.now() / 1000 - chapter.updatedAt < FRESH_TITLE_SECONDS,
  }));
  if (streaming.title !== chapter.title) {
    setStreaming({ title: chapter.title, animate: chapter.titleSource !== "Person" });
  }

  const edit = () => {
    settled.current = false;
    setEditing(true);
  };

  const finish = (value: string | null) => {
    if (settled.current) {
      return;
    }
    settled.current = true;
    setEditing(false);
    const title = (value ?? "").replace(/\s+/g, " ").trim();
    if (title !== "" && title !== shown) {
      setSent(title);
      actions.rename(entry, title);
    }
  };

  const onFieldKey = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") {
      event.preventDefault();
      finish(event.currentTarget.value);
    } else if (event.key === "Escape") {
      event.preventDefault();
      event.stopPropagation();
      finish(null);
    }
  };

  const onTitleKey = (event: KeyboardEvent<HTMLButtonElement>) => {
    if (event.key === "Enter" || event.key === "F2") {
      event.preventDefault();
      edit();
    }
  };

  return (
    <div className={cn("dk-chap-h", entry.live && "dk-chap-live")}>
      <span className="dk-chap-n" aria-hidden>
        {String(entry.number).padStart(2, "0")}
      </span>
      {editing ? (
        <input
          className="dk-chap-in"
          // oxlint-disable-next-line jsx-a11y/no-autofocus -- the title turned into this field on purpose
          autoFocus
          defaultValue={shown}
          maxLength={MAX_CHAPTER_TITLE}
          aria-label={t("Chapter title")}
          onFocus={(event) => event.target.select()}
          onBlur={(event) => finish(event.target.value)}
          onKeyDown={onFieldKey}
        />
      ) : (
        <h2 className="dk-chap-title">
          <DeskTip label={t("Double-click to rename")} side="top">
            <Button
              variant="bare"
              size="bare"
              className="dk-chap-t"
              aria-label={shown}
              aria-keyshortcuts="Enter F2"
              onDoubleClick={edit}
              onKeyDown={onTitleKey}
            >
              {shown === chapter.title ? (
                <StreamedText
                  as="span"
                  streamKey={`${chapter.id}:${chapter.title}`}
                  segments={[{ text: chapter.title }]}
                  animate={streaming.animate}
                />
              ) : (
                shown
              )}
            </Button>
          </DeskTip>
        </h2>
      )}
      {chapter.titleSource === "Schedule" && chapter.startedAt > 0 && (
        <span className="dk-chap-when">
          <DeskIcon name="clock" size={12} />
          {formatUnixMonthDay(chapter.startedAt, { timezone })}
        </span>
      )}
      <span className="dk-chap-rule" aria-hidden />
      <ChapterMenu entry={entry} actions={actions} saved={saved} onRename={edit} />
    </div>
  );
}

function ChapterMenu({
  entry,
  actions,
  saved,
  onRename,
}: {
  entry: ChapterEntry;
  actions: ChapterActions;
  saved: boolean;
  onRename: () => void;
}) {
  const t = useT();
  const { continueAlone, handOff, agents } = actions;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        disabled={actions.busy}
        render={
          <Button
            variant="quiet"
            size="icon-sm"
            className="dk-chap-menu"
            aria-label={t("Chapter actions")}
          />
        }
      >
        <DeskIcon name="more" size={14} />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" sideOffset={6} className="w-60">
        <DropdownMenuItem
          title={t("Rename")}
          startContent={<DeskIcon name="edit" size={14} />}
          onClick={onRename}
        />
        {continueAlone && (
          <DropdownMenuItem
            title={t("Continue in a new conversation")}
            description={t("Starts from this chapter alone")}
            startContent={<DeskIcon name="plus" size={14} />}
            disabled={!saved}
            onClick={() => continueAlone(entry)}
          />
        )}
        {handOff && agents.length > 0 && (
          <DropdownMenuSub>
            <DropdownMenuSubTrigger disabled={!saved}>
              <DeskIcon name="handoff" size={14} />
              {t("Hand off this chapter")}
            </DropdownMenuSubTrigger>
            <DropdownMenuSubContent className="w-56">
              {agents.map((agent) => (
                <DropdownMenuItem
                  key={agent.id}
                  title={agent.name}
                  startContent={<DeskAgentTile agent={agent} size="xs" />}
                  onClick={() => handOff(entry, agent.id)}
                />
              ))}
            </DropdownMenuSubContent>
          </DropdownMenuSub>
        )}
        <DropdownMenuSeparator />
        <DropdownMenuItem
          title={entry.number > 1 ? t("Merge into the chapter above") : t("Remove chapter")}
          startContent={<DeskIcon name="up" size={14} />}
          onClick={() => actions.merge(entry)}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
