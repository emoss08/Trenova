import { useTriageAccess } from "../use-triage-access";
import "@/routes/desk/_components/today/today.css";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import type { TriageBriefing, TriageRuleInput } from "@/lib/graphql/inbox";
import { TodaySection } from "@/routes/desk/_components/today/today-section";
import { useTriageStore } from "@/stores/triage-store";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { formatUnixInUserTimezone, formatUnixTime } from "@trenova/shared/lib/date";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useMemo, useState } from "react";
import type { TriageFolder } from "../folders";
import { useTriageActions } from "../use-triage-actions";
import { useTriageBriefing, useTriageCounts } from "../use-triage-data";
import { dayFigures, deckEntries, firstDayStage, novaLine, type DeckEntry } from "./briefing-model";
import { BriefingStrip } from "./briefing-strip";
import { CarryingOver } from "./carrying-over";
import { Deck, DeckChips, type DeckChip } from "./deck";
import { FirstDay } from "./first-day";
import { NovaLineText } from "./nova-line";
import { BriefingNotices, hasNotices } from "./notices";
import { River } from "./river";

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

/** How long "Done reading" shows before the first briefing. */
const DONE_READING_MS = 1100;

type Filed = { ids: string[]; count: number };

/** The briefing's one-line headline: how much is yours, in the Desk's greeting type. */
function headline(t: TranslateFn, briefing: TriageBriefing): string {
  if (briefing.phase === "EndOfDay") {
    return t("Wrapping up the day");
  }
  if (briefing.waiting === 0) {
    return t("You're all caught up");
  }
  return t(
    "{0, plural, one {# message is waiting on you} other {# messages are waiting on you}}",
    briefing.waiting,
  );
}

/**
 * Triage's landing page: how much is left for you and how the day has gone,
 * what Nova handled, and the work left grouped by what it needs, one card at
 * a time. On a first day it is the way in; at the end of the day it is what
 * carries over and who it is handed to.
 */
export function BriefingView({ onOpenMessage, onOpenFolder }: BriefingViewProps) {
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

  const openConnect = () =>
    useTriageStore.getState().openOverlay({ kind: "connect", mailboxId: null });

  return (
    <div className="dk-dy-w @container min-h-0 flex-1">
      {/* The Desk's scroll area, so the page scrolls as its lists do, fading at an edge with more beyond it. */}
      <ScrollArea className="min-h-0" maskClassName="before:from-dsk-canvas after:from-dsk-canvas">
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
      </ScrollArea>
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
  const access = useTriageAccess();
  const counts = useTriageCounts().data;
  const [filed, setFiled] = useState<Filed | null>(null);
  const [drafting, setDrafting] = useState<readonly string[] | null>(null);

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
    });
  }
  if (draftsDone && drafting !== null) {
    chips.push({
      group: "Draft",
      label: t("{0, plural, one {# draft ready} other {# drafts ready}}", drafting.length),
      firstId: drafting[0] ?? null,
    });
  }

  const act = (entry: DeckEntry) => {
    const first = entry.messageIds[0];
    switch (entry.group) {
      case "File": {
        const ids = entry.messageIds;
        setFiled((current) => ({
          ids: [...(current?.ids ?? []), ...ids],
          count: (current?.count ?? 0) + ids.length,
        }));
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

  if (briefing === undefined) {
    return <BriefingSkeleton />;
  }

  const line = novaLine({ ...briefing, onCall: briefing.onCall ?? null });
  const figures = dayFigures(briefing.yours, briefing.deck, now);
  const endOfDay = briefing.phase === "EndOfDay";
  const quietDay = briefing.novaPoints.length === 0 && briefing.yours.length === 0;

  return (
    <div className="dk-dy animate-rise">
      <div className="dk-dy-date">
        {t(
          "Triage · {0}",
          formatUnixInUserTimezone(now, { weekday: "long", month: "long", day: "numeric" }, ""),
        )}
      </div>
      <h1 className="dk-dy-greet">{headline(t, briefing)}</h1>
      <NovaLineText line={line} now={now} onOpenMessage={onOpenMessage} />
      <p className="dk-dy-hl-note">
        <NovaBadge size="xs" />
        {t("Nova, as of {0}", formatUnixTime(now))}
      </p>

      <BriefingStrip
        briefing={briefing}
        counts={counts}
        figures={figures}
        onOpenFolder={onOpenFolder}
      />

      {hasNotices(briefing) && (
        <TodaySection id="triage-notices" title={t("Worth a look")}>
          <BriefingNotices
            briefing={briefing}
            onOpenFolder={onOpenFolder}
            onPreview={preview}
            onTeach={actions.createRule}
            canTeach={access.canManage}
          />
        </TodaySection>
      )}

      {endOfDay ? (
        <TodaySection
          id="triage-carrying-over"
          title={t("Carrying over")}
          count={String(briefing.carryOver.length)}
        >
          <CarryingOver
            items={briefing.carryOver}
            onCall={briefing.onCall ?? null}
            now={now}
            onOpenMessage={onOpenMessage}
            onHandOff={actions.handOff}
            canHandOff={access.canAssign}
          />
        </TodaySection>
      ) : (
        <TodaySection
          id="triage-left"
          title={t("What is left, by what it needs")}
          count={entries.length > 0 ? String(briefing.waiting) : undefined}
          aside={
            line.kind === "day" && line.minutes > 0
              ? t("{0, plural, one {About # minute} other {About # minutes}}", line.minutes)
              : undefined
          }
        >
          <div className="flex flex-col gap-2.5">
            {entries.length > 0 ? (
              <Deck
                entries={entries}
                now={now}
                drafting={drafting !== null && !draftsDone}
                canAct={access.canWork}
                onAct={act}
                onOpenMessage={onOpenMessage}
              />
            ) : (
              <p className="dk-dy-quiet">
                {t(
                  "{0, plural, one {Nothing left. Nova is watching # source.} other {Nothing left. Nova is watching # sources.}}",
                  briefing.activeSources,
                )}
              </p>
            )}
            {chips.length > 0 && (
              <DeckChips
                chips={chips}
                onOpen={(chip) => {
                  if (chip.firstId !== null) {
                    onOpenMessage(chip.firstId);
                  }
                }}
              />
            )}
          </div>
        </TodaySection>
      )}

      <TodaySection
        id="triage-day"
        title={t("Today so far")}
        aside={quietDay ? undefined : t("Nova's work above, yours below")}
      >
        {quietDay ? (
          <p className="dk-dy-quiet">{t("No mail has arrived today.")}</p>
        ) : (
          <River briefing={briefing} now={now} onOpenMessage={onOpenMessage} />
        )}
      </TodaySection>
    </div>
  );
}

/** The briefing while it is read: the page's own shape, in shimmer. */
function BriefingSkeleton() {
  const t = useT();

  return (
    <div className="dk-dy" aria-busy aria-label={t("Going through the day")}>
      <span className="mx-auto h-3 w-40 ui-shimmer rounded-[4px]" />
      <span className="mx-auto mt-4 h-11 w-[min(30rem,90%)] ui-shimmer rounded-[6px]" />
      <span className="mx-auto mt-4 h-4.5 w-[min(26rem,80%)] ui-shimmer rounded-[4px]" />
      <span className="mt-9 h-18 ui-shimmer rounded-[6px]" />
      <span className="mt-9 h-28 ui-shimmer rounded-[8px]" />
      <span className="mt-2.5 h-28 ui-shimmer rounded-[8px]" />
    </div>
  );
}
