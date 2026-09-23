import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { KeyboardIcon, MicOffIcon, XIcon } from "lucide-react";
import { AnimatePresence, m, useReducedMotion } from "motion/react";
import type { ReactNode } from "react";
import type { ComposerHintKind } from "./composer-hint";
import { dictationIssueText } from "./dictation-control";
import type { DictationIssue } from "./use-dictation";
import { DeskThinking } from "./voice/desk-thinking";

/** The keys in a hint sit a step smaller than a standalone Kbd, so the line reads as text. */
const HINT_KBD = "h-4 min-w-4 px-1";

function Dot() {
  return (
    <span aria-hidden className="text-foreground-subtle px-0.5">
      ·
    </span>
  );
}

function HintLine({ kind, children }: { kind: ComposerHintKind; children: ReactNode }) {
  const reduceMotion = useReducedMotion();

  return (
    <m.div
      data-hint={kind}
      initial={reduceMotion ? false : { opacity: 0, y: 3 }}
      animate={{ opacity: 1, y: 0 }}
      exit={reduceMotion ? { opacity: 0, transition: { duration: 0 } } : { opacity: 0, y: -3 }}
      transition={{ duration: 0.14, ease: [0.2, 0.8, 0.2, 1] }}
      className="absolute inset-0 flex min-w-0 items-center gap-1 whitespace-nowrap"
    >
      {children}
    </m.div>
  );
}

export type ComposerHintsProps = {
  kind: ComposerHintKind;
  issue: DictationIssue | null;
  onDismissIssue: () => void;
  canMention: boolean;
  canAttach: boolean;
  canDictate: boolean;
};

/**
 * The line under the composer: one hint that fits on one line at the
 * popover's width, and the shortcuts behind a button at the end of it.
 *
 * It used to list every key at once — send, new line, commands, records —
 * which wrapped into two or three ragged rows in the corner panel. Now it
 * says what is useful for what the person is doing, and swaps as that
 * changes; the full list is one click away and never in the way.
 */
export function ComposerHints({
  kind,
  issue,
  onDismissIssue,
  canMention,
  canAttach,
  canDictate,
}: ComposerHintsProps) {
  const t = useT();

  return (
    <div className="text-foreground-subtle flex h-6 items-center gap-2 pl-1 text-xs">
      {/* Only a problem is announced; the keyboard hints change with focus
          and typing, and reading each one aloud would be noise. */}
      <span role="status" className="sr-only">
        {kind === "issue" && issue !== null ? dictationIssueText(issue, t).detail : ""}
      </span>
      {/* The lines overlap while one gives way to the next, so the swap is a
          crossfade in place rather than a line that briefly goes blank. */}
      <div
        aria-hidden={kind !== "issue"}
        className="relative h-full min-w-0 flex-1 overflow-hidden"
      >
        <AnimatePresence initial={false}>
          {kind === "issue" && issue !== null ? (
            <HintLine key={kind} kind={kind}>
              <MicOffIcon
                aria-hidden
                className={cn(
                  "size-3 shrink-0",
                  issue === "no-speech" ? "text-foreground-muted" : "text-danger-foreground",
                )}
              />
              <span
                className={cn(
                  "min-w-0 truncate",
                  issue === "no-speech" ? "text-foreground-muted" : "text-danger-foreground",
                )}
              >
                {dictationIssueText(issue, t).short}
              </span>
              <button
                type="button"
                onClick={onDismissIssue}
                aria-label={t("Dismiss")}
                className="ui-focus-ring hover:text-foreground hover:bg-surface-hover inline-flex size-4 shrink-0 items-center justify-center rounded-full transition-colors"
              >
                <XIcon className="size-3" />
              </button>
            </HintLine>
          ) : kind === "starting" ? (
            <HintLine key={kind} kind={kind}>
              {t("Waiting for the microphone…")}
            </HintLine>
          ) : kind === "listening" ? (
            <HintLine key={kind} kind={kind}>
              <span className="text-foreground-muted">{t("Listening")}</span>
              <Dot />
              <Kbd className={HINT_KBD}>Esc</Kbd>
              {t("to stop")}
            </HintLine>
          ) : kind === "choosing" ? (
            <HintLine key={kind} kind={kind}>
              <KbdGroup>
                <Kbd className={HINT_KBD}>↑</Kbd>
                <Kbd className={HINT_KBD}>↓</Kbd>
              </KbdGroup>
              {t("to move")}
              <Dot />
              <Kbd className={HINT_KBD}>Enter</Kbd>
              {t("to choose")}
            </HintLine>
          ) : kind === "replying" ? (
            <HintLine key={kind} kind={kind}>
              <DeskThinking working pose="write" still decorative />
              <span className="truncate">{t("Replying. You can draft your next message.")}</span>
            </HintLine>
          ) : kind === "discover" ? (
            <HintLine key={kind} kind={kind}>
              <Kbd className={HINT_KBD}>/</Kbd>
              {t("for commands")}
              {canMention && (
                <>
                  <Dot />
                  <Kbd className={HINT_KBD}>@</Kbd>
                  {t("for records")}
                </>
              )}
            </HintLine>
          ) : kind === "compose" ? (
            <HintLine key={kind} kind={kind}>
              <Kbd className={HINT_KBD}>Enter</Kbd>
              {t("to send")}
              <Dot />
              <KbdGroup>
                <Kbd className={HINT_KBD}>Shift</Kbd>
                <Kbd className={HINT_KBD}>Enter</Kbd>
              </KbdGroup>
              {t("for a new line")}
            </HintLine>
          ) : null}
        </AnimatePresence>
      </div>

      <ShortcutsPopover canMention={canMention} canAttach={canAttach} canDictate={canDictate} />
    </div>
  );
}

function ShortcutRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <>
      <dt className="text-muted-foreground min-w-0 truncate">{label}</dt>
      <dd className="flex items-center justify-end gap-1">{children}</dd>
    </>
  );
}

/** Every key the composer answers to, for the person who wants the list. */
function ShortcutsPopover({
  canMention,
  canAttach,
  canDictate,
}: {
  canMention: boolean;
  canAttach: boolean;
  canDictate: boolean;
}) {
  const t = useT();

  return (
    <Popover>
      <PopoverTrigger
        aria-label={t("Keyboard shortcuts")}
        className="ui-focus-ring hover:text-foreground hover:bg-surface-hover data-popup-open:text-foreground data-popup-open:bg-surface-hover inline-flex size-5.5 shrink-0 items-center justify-center rounded-full transition-colors"
      >
        <KeyboardIcon className="size-3.5" />
      </PopoverTrigger>
      <PopoverContent side="top" align="end" className="w-64 gap-2 p-3">
        <p className="text-xs font-semibold">{t("Keyboard shortcuts")}</p>
        <dl className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1.5 text-xs">
          <ShortcutRow label={t("Send")}>
            <Kbd>Enter</Kbd>
          </ShortcutRow>
          <ShortcutRow label={t("New line")}>
            <Kbd>Shift</Kbd>
            <Kbd>Enter</Kbd>
          </ShortcutRow>
          <ShortcutRow label={t("Commands and starter questions")}>
            <Kbd>/</Kbd>
          </ShortcutRow>
          {canMention && (
            <ShortcutRow label={t("Name a record")}>
              <Kbd>@</Kbd>
            </ShortcutRow>
          )}
          {canDictate && (
            <ShortcutRow label={t("Stop dictating")}>
              <Kbd>Esc</Kbd>
            </ShortcutRow>
          )}
          {canAttach && (
            <ShortcutRow label={t("Attach a file")}>
              <span className="text-muted-foreground">{t("Paste or drop")}</span>
            </ShortcutRow>
          )}
        </dl>
      </PopoverContent>
    </Popover>
  );
}
