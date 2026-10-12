import "../../_styles/triage-briefing.css";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import type { TriageBriefing, TriageRuleInput } from "@/lib/graphql/inbox";
import { useTriageStore } from "@/stores/triage-store";
import { CheckIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useMemo, useRef, useState, type CSSProperties, type ReactNode } from "react";
import type { TriageFolder } from "../folders";
import { useTriageActions } from "../use-triage-actions";
import { useTriageBriefing, useTriageCounts } from "../use-triage-data";
import {
  deckEntries,
  firstDayStage,
  novaLine,
  ringProgress,
  type DeckEntry,
  type DeckGroup,
} from "./briefing-model";
import { CarryingOver } from "./carrying-over";
import { Deck, DeckChips, type DeckChip } from "./deck";
import { FirstDay } from "./first-day";
import { FooterPills } from "./footer-pills";
import { Motes } from "./motes";
import { NovaLineText } from "./nova-line";
import { Ring } from "./ring";
import { River } from "./river";
import { usePausedWhenAway } from "./use-briefing-motion";

/**
 * The Briefing's contract with the page: it opens messages and folders
 * through these, and reads its data from `useTriageBriefing()` and performs
 * every action through `useTriageActions()`.
 */
export type BriefingViewProps = {
  /** Opens a message in Waiting on you, as every name, dot and card row does. */
  onOpenMessage: (id: string) => void;
  /** Opens a folder of the queue, as a count or a footer pill does. */
  onOpenFolder: (folder: TriageFolder) => void;
};

/** How long a card takes to leave the stack after "File all". */
const FLY_MS = 420;
/** How long "Done reading" shows before the first briefing. */
const DONE_READING_MS = 1100;

type Filed = { ids: string[]; count: number };

function Stage({
  on,
  delay,
  className,
  children,
}: {
  on: boolean;
  delay: number;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div
      className={["tb-stage", className].filter(Boolean).join(" ")}
      data-on={on}
      style={{ "--tb-delay": `${delay}ms` } as CSSProperties}
    >
      {on && children}
    </div>
  );
}

/**
 * Triage's landing page: how much is left for you and how the day has gone,
 * what Nova handled, and the work left grouped by what it needs, one card at
 * a time. On a first day it is the way in; at the end of the day it is what
 * carries over and who it is handed to.
 */
export function BriefingView({ onOpenMessage, onOpenFolder }: BriefingViewProps) {
  const root = useRef<HTMLDivElement>(null);
  const paused = usePausedWhenAway(root);
  const briefing = useTriageBriefing().data;
  const counts = useTriageCounts().data;
  const now = useNowSeconds(30_000);
  const connectedAt = useTriageStore((store) => store.sourceConnectedAt);
  const canConnect = usePermissionStore((store) =>
    store.hasPermission(Resource.InboundMailbox, Operation.Create),
  );
  const [finished, setFinished] = useState(false);

  const stage =
    briefing === undefined
      ? null
      : firstDayStage({
          firstDay: briefing.firstDay,
          connectedAt,
          counts: { total: counts?.total ?? 0, processing: counts?.processing ?? 0 },
          now,
          finished,
        });

  useEffect(() => {
    if (stage !== "done") {
      return;
    }
    const timer = setTimeout(() => setFinished(true), DONE_READING_MS);
    return () => clearTimeout(timer);
  }, [stage]);

  const openConnect = () => useTriageStore.getState().openOverlay({ kind: "connect", mailboxId: null });

  return (
    <div
      ref={root}
      data-paused={paused}
      className="tb-briefing scrollbar-overlay relative min-h-0 flex-1 overflow-x-hidden overflow-y-auto"
    >
      <Motes />
      {stage !== null && stage !== "briefing" ? (
        <FirstDay stage={stage} counts={counts} canConnect={canConnect} onConnect={openConnect} />
      ) : (
        <BriefingMain
          briefing={briefing}
          now={now}
          onOpenMessage={onOpenMessage}
          onOpenFolder={onOpenFolder}
        />
      )}
    </div>
  );
}

function BriefingMain({
  briefing,
  now,
  onOpenMessage,
  onOpenFolder,
}: {
  briefing: TriageBriefing | undefined;
  now: number;
  onOpenMessage: (id: string) => void;
  onOpenFolder: (folder: TriageFolder) => void;
}) {
  const t = useT();
  const actions = useTriageActions();
  const counts = useTriageCounts().data;
  const [filed, setFiled] = useState<Filed | null>(null);
  const [drafting, setDrafting] = useState<readonly string[] | null>(null);
  const [flying, setFlying] = useState<DeckGroup | null>(null);
  const ready = briefing !== undefined;

  const serverDrafts = briefing?.deck.find((card) => card.group === "Draft")?.messageIds;
  const draftsDone =
    drafting !== null &&
    briefing !== undefined &&
    !drafting.some((id) => serverDrafts?.includes(id) ?? false);

  const acted = useMemo(() => {
    const ids = new Set(filed?.ids ?? []);
    if (draftsDone && drafting !== null) {
      for (const id of drafting) {
        ids.add(id);
      }
    }
    return ids;
  }, [filed, drafting, draftsDone]);

  const entries = useMemo(
    () => (briefing === undefined ? [] : deckEntries(briefing.deck, acted)),
    [briefing, acted],
  );

  const chips: DeckChip[] = [];
  if (filed !== null) {
    chips.push({
      group: "File",
      label: t("{0, plural, one {Filed # document} other {Filed # documents}}", filed.count),
      firstId: filed.ids[0] ?? null,
      busy: false,
    });
  }
  if (draftsDone && drafting !== null) {
    chips.push({
      group: "Draft",
      label: t("{0, plural, one {# draft ready} other {# drafts ready}}", drafting.length),
      firstId: drafting[0] ?? null,
      busy: false,
    });
  }

  const act = (entry: DeckEntry) => {
    const first = entry.messageIds[0];
    switch (entry.group) {
      case "File": {
        const ids = entry.messageIds;
        setFlying("File");
        setTimeout(() => {
          setFlying(null);
          setFiled((current) => ({
            ids: [...(current?.ids ?? []), ...ids],
            count: (current?.count ?? 0) + ids.length,
          }));
        }, FLY_MS);
        actions.fileMany(ids).catch(() => {
          setFiled((current) => {
            if (current === null) {
              return current;
            }
            const left = current.ids.filter((id) => !ids.includes(id));
            return left.length === 0 ? null : { ids: left, count: left.length };
          });
        });
        return;
      }
      case "Draft": {
        const ids = entry.messageIds;
        setDrafting(ids);
        actions.draftMany(ids).catch(() => setDrafting(null));
        return;
      }
      case "Link":
        if (first !== undefined) {
          onOpenMessage(first);
          useTriageStore.getState().openOverlay({ kind: "link", messageId: first });
        }
        return;
      case "Decide":
      case "Held":
        if (first !== undefined) {
          onOpenMessage(first);
        }
        return;
    }
  };

  const preview = (input: TriageRuleInput) =>
    useTriageStore.getState().openOverlay({ kind: "ruleMatches", target: { input } });

  const line =
    briefing === undefined
      ? null
      : novaLine({ ...briefing, onCall: briefing.onCall ?? null });
  const cleared = briefing?.yours.filter((point) => point.done).length ?? 0;
  const endOfDay = briefing?.phase === "EndOfDay";

  return (
    <div className="relative mx-auto flex w-full max-w-220 flex-col gap-7.5 px-8 pt-11 pb-18 @max-2xl:gap-6 @max-2xl:px-4 @max-2xl:pt-6">
      <header className="flex items-center gap-7 @max-2xl:flex-col @max-2xl:items-start @max-2xl:gap-4">
        <Ring
          progress={briefing === undefined ? 0 : ringProgress(cleared, briefing.waiting)}
          count={briefing?.waiting ?? 0}
          busy={!ready}
        />
        <div className="min-w-0 flex-1">
          <div className="text-foreground-subtle flex items-center gap-2 text-sm">
            <b className="text-foreground font-medium">{t("Nova")}</b>
            <span className="text-2xs font-mono tabular-nums">{formatUnixTime(now)}</span>
          </div>
          <NovaLineText line={line} now={now} onOpenMessage={onOpenMessage} />
        </div>
      </header>

      <Stage on={ready} delay={80}>
        {briefing !== undefined && (
          <River briefing={briefing} now={now} onOpenMessage={onOpenMessage} />
        )}
      </Stage>

      <Stage on={ready} delay={260} className="flex min-h-84 flex-col items-center gap-3.5 @max-2xl:min-h-0">
        {briefing !== undefined &&
          (endOfDay ? (
            <CarryingOver
              items={briefing.carryOver}
              onCall={briefing.onCall ?? null}
              now={now}
              onOpenMessage={onOpenMessage}
              onHandOff={actions.handOff}
            />
          ) : entries.length > 0 ? (
            <Deck
              entries={entries}
              now={now}
              drafting={drafting !== null && !draftsDone}
              flying={flying}
              onAct={act}
              onOpenMessage={onOpenMessage}
            />
          ) : (
            <div className="animate-confirm flex flex-col items-center gap-2 py-12 text-center">
              <span className="bg-success-subtle text-success flex size-11 items-center justify-center rounded-full">
                <CheckIcon aria-hidden className="size-5" strokeWidth={2.4} />
              </span>
              <b className="text-lg font-semibold">{t("Nothing left in the deck")}</b>
              <span className="text-foreground-subtle text-base">
                {t(
                  "{0, plural, one {Nova is watching # source} other {Nova is watching # sources}}",
                  briefing.activeSources,
                )}
              </span>
            </div>
          ))}
        {!endOfDay && (
          <DeckChips
            chips={chips}
            onOpen={(chip) => {
              if (chip.firstId !== null) {
                onOpenMessage(chip.firstId);
              }
            }}
          />
        )}
      </Stage>

      <Stage on={ready} delay={460}>
        {briefing !== undefined && (
          <FooterPills
            briefing={briefing}
            counts={counts}
            onOpenFolder={onOpenFolder}
            onPreview={preview}
            onTeach={actions.createRule}
          />
        )}
      </Stage>
    </div>
  );
}
