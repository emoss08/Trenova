import { describeToolCall } from "@/components/assistant/tool-presentation";
import { AssistantStreamError } from "@/services/assistant";
import { apiService } from "@/services/api";
import type { SaveAgentDefinitionRequest } from "@/types/assistant";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { getNameInitials } from "@trenova/shared/lib/utils";
import { useCallback, useEffect, useRef, useState } from "react";
import { Callout } from "../../edit/callout";
import { Ic } from "../../kit/ic";
import { Tile } from "../../kit/marks";
import {
  INITIAL_DRY_RUN,
  OUTCOME_TONE,
  reduceDryRun,
  type DryRunOutcome,
  type DryRunState,
} from "./dry-run-model";
import { Button } from "@trenova/shared/components/ui/button";

type TryRun = DryRunState & {
  id: number;
  question: string;
  /** The draft as it was tried, to tell when it has changed since. */
  snapshot: string;
  name: string;
  icon: string;
  accent: string;
  mode: "live" | "shadow" | "sim";
  startedAt: number;
  finishedAt: number | null;
};

type TryPanelProps = {
  agentId: string | null;
  /** The draft as it stands, for the save request and its snapshot. */
  draft: SaveAgentDefinitionRequest;
  snapshot: string;
  /** An opening question the agent offers, when it has one. */
  starter: string | null;
  available: boolean;
  onTried: (snapshot: string) => void;
  onClose: () => void;
};

function outcomeLabel(outcome: DryRunOutcome, t: TranslateFn): string {
  switch (outcome) {
    case "Runs":
      return t("Runs");
    case "AskFirst":
      return t("Asks a person first");
    case "Propose":
      return t("Proposes only");
    case "Recorded":
      return t("Recorded, not offered");
    case "Simulated":
      return t("Simulated");
    case "NotHeld":
      return t("Not held · skipped");
    case "Failed":
      return t("Failed");
  }
}

/**
 * Asks the unsaved draft something against live records, with every write simulated, and
 * shows what each tool call would have come to beside the answer.
 */
export function TryPanel({
  agentId,
  draft,
  snapshot,
  starter,
  available,
  onTried,
  onClose,
}: TryPanelProps) {
  const t = useT();
  const user = useAuthStore((state) => state.user);
  const [runs, setRuns] = useState<TryRun[]>([]);
  const [question, setQuestion] = useState("");
  const thread = useRef<HTMLDivElement>(null);
  const controllers = useRef(new Set<AbortController>());
  const sequence = useRef(0);
  const running = runs.some((run) => run.status === "running");
  const suggestions = [
    starter ?? t("What can you help me with?"),
    t("What can't you do?"),
    t("Walk me through your last decision"),
  ];

  useEffect(() => {
    const live = controllers.current;
    return () => {
      for (const controller of live) controller.abort();
      live.clear();
    };
  }, []);

  useEffect(() => {
    thread.current?.scrollTo({ top: thread.current.scrollHeight, behavior: "smooth" });
  }, [runs]);

  const patch = useCallback((id: number, next: (run: TryRun) => TryRun) => {
    setRuns((current) => current.map((run) => (run.id === id ? next(run) : run)));
  }, []);

  const ask = useCallback(
    (text?: string) => {
      const prompt = (text ?? question).trim();
      if (!prompt || !available || running) return;
      setQuestion("");
      sequence.current += 1;
      const id = sequence.current;
      const tried = snapshot;
      setRuns((current) => [
        ...current,
        {
          ...INITIAL_DRY_RUN,
          id,
          question: prompt,
          snapshot: tried,
          name: draft.name || t("New agent"),
          icon: draft.icon ?? "",
          accent: draft.accent ?? "",
          mode: draft.simulationMode ? "sim" : draft.shadowMode ? "shadow" : "live",
          startedAt: Date.now(),
          finishedAt: null,
        },
      ]);

      const controller = new AbortController();
      controllers.current.add(controller);
      void apiService.agentDefinitionService
        .dryRun(
          { agentId, draft, prompt },
          (event, data) => patch(id, (run) => ({ ...run, ...reduceDryRun(run, event, data) })),
          { signal: controller.signal },
        )
        .then(() => {
          patch(id, (run) => {
            const settled: TryRun =
              run.status === "running"
                ? { ...run, status: "failed", error: t("The dry run stopped before it finished.") }
                : run;
            return { ...settled, finishedAt: Date.now() };
          });
        })
        .catch((error: unknown) => {
          if (controller.signal.aborted) return;
          patch(id, (run) => ({
            ...run,
            status: "failed",
            error:
              error instanceof AssistantStreamError
                ? error.message
                : t("The dry run could not start. Try again in a moment."),
            finishedAt: Date.now(),
          }));
        })
        .finally(() => controllers.current.delete(controller));
    },
    [agentId, available, draft, patch, question, running, snapshot, t],
  );

  useEffect(() => {
    const last = runs.at(-1);
    if (last && last.status === "done") {
      onTried(last.snapshot);
    }
  }, [onTried, runs]);

  const clear = () => {
    for (const controller of controllers.current) controller.abort();
    controllers.current.clear();
    setRuns([]);
  };

  return (
    <aside className="tp2" aria-label={t("Try it")}>
      <header className="tp2-h">
        <Ic n="flask" s={13} />
        <b>{t("Try it")}</b>
        <span className="tp2-p">{t("Simulation")}</span>
        <span className="sp" />
        {runs.length > 0 && (
          <button type="button" className="lnk" onClick={clear}>
            {t("Clear")}
          </button>
        )}
        <Button type="button" variant="ghost" size="icon-sm" className="text-muted-foreground hover:text-foreground" title={t("Hide")} aria-label={t("Hide")} onClick={onClose}>
          <Ic n="x" s={13} />
        </Button>
      </header>
      <div className="tp2-b" ref={thread} aria-live="polite">
        {runs.length === 0 ? (
          <div className="tp2-e">
            <Tile agent={{ name: draft.name, icon: draft.icon, accent: draft.accent }} s={40} />
            <b>{t("Ask {0} something", draft.name || t("it"))}</b>
            <p>
              {t(
                "It runs your unsaved draft against live records. Nothing is written, sent or offered to anyone.",
              )}
            </p>
            <div className="tp2-sg">
              {suggestions.map((suggestion) => (
                <button
                  key={suggestion}
                  type="button"
                  disabled={!available}
                  onClick={() => ask(suggestion)}
                >
                  {suggestion}
                  <Ic n="arrowR" s={11} />
                </button>
              ))}
            </div>
            {!available && <Callout tone="w">{t("Connect a provider to try agents.")}</Callout>}
          </div>
        ) : (
          runs.map((run, index) => {
            const finished = run.status !== "running";
            const stale = finished && index === runs.length - 1 && run.snapshot !== snapshot;
            const elapsed = run.finishedAt ? (run.finishedAt - run.startedAt) / 1000 : null;
            return (
              <div key={run.id} className="tp2-r">
                <div className="tp2-q">
                  <span className="me sm">{getNameInitials(user?.name ?? "", "?")}</span>
                  <p>{run.question}</p>
                </div>
                <div className="tp2-a">
                  <div className="tp2-w">
                    <Tile agent={{ name: run.name, icon: run.icon, accent: run.accent }} s={18} />
                    <b>{run.name}</b>
                    <span className="mono">
                      {run.mode === "live" ? t("live draft") : run.mode === "shadow" ? t("shadow") : t("simulation")}
                    </span>
                  </div>
                  <div className="tp2-l">
                    {run.steps.map((step) => {
                      const going = step.outcome === null;
                      const skipped = step.outcome === "NotHeld" || step.outcome === "Failed";
                      return (
                        <div key={step.callId} className={going ? "tr-s cur" : "tr-s"}>
                          <span className="ck">
                            {going ? (
                              <span className="border-border-strong border-t-foreground inline-block size-3 animate-spin rounded-full border-[1.5px]" />
                            ) : skipped ? (
                              <Ic n="ban" s={12} />
                            ) : (
                              <Ic n="check" s={12} w={2.4} />
                            )}
                          </span>
                          <span className="tx">
                            <b>{describeToolCall(step.tool, null).title}</b>
                            <em className="mono">{step.tool}</em>
                          </span>
                          {step.outcome && (
                            <span className={`oc ${OUTCOME_TONE[step.outcome]}`}>
                              {outcomeLabel(step.outcome, t)}
                            </span>
                          )}
                        </div>
                      );
                    })}
                  </div>
                  {run.reply && (
                    <p className="tp2-t">
                      {run.reply}
                      {!finished && <span className="car" />}
                    </p>
                  )}
                  {run.status === "failed" && run.error && <Callout tone="d">{run.error}</Callout>}
                  {run.status === "done" && elapsed !== null && (
                    <span className="tp2-m mono">
                      {run.steps.length === 1
                        ? t("{0}s · 1 tool call · nothing written", elapsed.toFixed(1))
                        : t("{0}s · {1} tool calls · nothing written", elapsed.toFixed(1), run.steps.length)}
                    </span>
                  )}
                  {stale && (
                    <div className="tp2-st">
                      <Ic n="refresh" s={12} />
                      <span>{t("You changed the draft since this run")}</span>
                      <Button type="button" variant="outline" size="sm" onClick={() => ask(run.question)}>
                        {t("Run again")}
                      </Button>
                    </div>
                  )}
                </div>
              </div>
            );
          })
        )}
      </div>
      <div className="tp2-c">
        <textarea
          rows={1}
          value={question}
          disabled={!available}
          aria-label={t("Ask the draft")}
          placeholder={
            available
              ? t("Ask as {0}…", user?.name ?? t("yourself"))
              : t("Connect a provider first")
          }
          onChange={(event) => setQuestion(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter" && !event.shiftKey) {
              event.preventDefault();
              ask();
            }
          }}
        />
        <button
          type="button"
          className="tp2-go"
          disabled={!question.trim() || !available || running}
          title={t("Run (Enter)")}
          aria-label={t("Run (Enter)")}
          onClick={() => ask()}
        >
          <Ic n="up" s={14} w={2.2} />
        </button>
      </div>
    </aside>
  );
}
