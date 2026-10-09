import { describeToolCall } from "@/components/assistant/tool-presentation";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  applyAITuneUp,
  dismissAITuneUp,
  restoreAITuneUp,
  type AITuneUp,
} from "@/lib/graphql/ai-control";
import { queries } from "@/lib/queries";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { Ic } from "../kit/ic";
import { SecH } from "../kit/layout";
import { Mark, Tile } from "../kit/marks";
import { tuneUpCopy, type TuneUpLabels, type TuneUpTone } from "./tune-up-copy";
import { invalidateAIControlCounts } from "./use-ai-control-stats";
import { Button } from "@trenova/shared/components/ui/button";

/** How long a row takes to fold away once it is applied or put away. */
const COLLAPSE_MS = 260;

const GAIN_TONE: Record<TuneUpTone, string> = {
  ok: "k",
  warn: "w",
  brand: "b",
  neutral: "",
};

const KIND_ICON = {
  RaiseToolTier: "award",
  ReorderProviders: "swap",
  LeaveShadow: "eyeOff",
  AssignTask: "search",
  TurnOffIdleAgent: "clock",
} as const;

/**
 * Changes to how AI is set up that the last 30 days of runs argue for: a tool people keep
 * approving, a provider that keeps catching another's failures, an agent ready to leave
 * shadow, a task nothing serves, an agent nobody asks. Applying makes the change; putting
 * one away keeps it from being suggested again for 30 days. Operational work stays in
 * Watchtower.
 */
export function TuneUps() {
  const t = useT();
  const tuneUpsQuery = useQuery(queries.aiControl.tuneUps());
  const items = tuneUpsQuery.data?.items ?? [];

  return (
    <section className="sec">
      <SecH
        t={t("Tune-ups")}
        n={items.length || null}
        r={
          <span className="sh2-n">
            {t("From the last {0} days of runs", tuneUpsQuery.data?.windowDays ?? 30)}
          </span>
        }
      />
      {tuneUpsQuery.isLoading ? null : items.length === 0 ? (
        <div className="clr">
          <span className="clr-i">
            <Ic n="check" s={16} w={2.4} />
          </span>
          <div>
            <b>{t("Nothing to tune")}</b>
            <span>
              {t(
                "Routing, autonomy and the roster all look right for how your agents are being used.",
              )}
            </span>
          </div>
        </div>
      ) : (
        <div className="nds">
          {items.map((tuneUp) => (
            <TuneUpRow key={tuneUp.id} tuneUp={tuneUp} />
          ))}
        </div>
      )}
    </section>
  );
}

function TuneUpRow({ tuneUp }: { tuneUp: AITuneUp }) {
  const t = useT();
  const queryClient = useQueryClient();
  const [leaving, setLeaving] = useState(false);
  const [now] = useState(() => Math.floor(Date.now() / 1000));
  const catalogQuery = useQuery(queries.aiProvider.catalog());
  const changesProviders = tuneUp.kind === "ReorderProviders" || tuneUp.kind === "AssignTask";
  const { allowed: canChange } = usePermission(
    changesProviders ? Resource.AIProvider : Resource.AgentDefinition,
    Operation.Update,
  );
  const { allowed: canDecide } = usePermission(Resource.AgentControl, Operation.Update);

  const taskLabels = useMemo(
    () => new Map((catalogQuery.data?.tasks ?? []).map((task) => [task.task, task.label])),
    [catalogQuery.data?.tasks],
  );
  const labels = useMemo<TuneUpLabels>(
    () => ({
      tool: (name) => describeToolCall(name, null).title,
      task: (task) => taskLabels.get(task as never) ?? task,
      day: (unix) => formatUnixDateMedium(unix),
      now,
    }),
    [now, taskLabels],
  );
  const copy = tuneUpCopy(tuneUp, t, labels);

  const refresh = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queries.aiControl._def }),
      queryClient.invalidateQueries({ queryKey: queries.aiProvider._def }),
      queryClient.invalidateQueries({ queryKey: ["agent-definitions"] }),
      invalidateAIControlCounts(queryClient),
    ]);
  }, [queryClient]);

  const fold = useCallback((after: () => void) => {
    setLeaving(true);
    window.setTimeout(after, COLLAPSE_MS);
  }, []);

  const restore = useApiMutation({
    mutationFn: restoreAITuneUp,
    resourceName: t("Tune-up"),
    onSuccess: refresh,
  });

  const dismiss = useApiMutation({
    mutationFn: dismissAITuneUp,
    resourceName: t("Tune-up"),
    onSuccess: (dismissed) =>
      fold(() => {
        void refresh();
        toast(t("Dismissed · it won't be suggested again for 30 days"), {
          action: {
            label: t("Undo"),
            onClick: () => restore.mutate({ id: dismissed.id, version: dismissed.version }),
          },
        });
      }),
  });

  const apply = useApiMutation({
    mutationFn: applyAITuneUp,
    resourceName: t("Tune-up"),
    onSuccess: () =>
      fold(() => {
        void refresh();
        toast.success(copy.applied);
      }),
  });

  const busy = apply.isPending || dismiss.isPending;

  return (
    <div className={cn("nd-w", leaving && "out")}>
      <div className="nd tu">
        {tuneUp.agent ? (
          <Tile agent={tuneUp.agent} s={30} />
        ) : tuneUp.provider ? (
          <Mark provider={tuneUp.provider} s={30} />
        ) : (
          <span />
        )}
        <div className="nd-b">
          <b className="tu-t">
            <Ic n={KIND_ICON[tuneUp.kind]} s={12} />
            <span>
              {copy.title.map((part, index) =>
                part.strong ? <b key={index}>{part.text}</b> : <span key={index}>{part.text}</span>,
              )}
            </span>
          </b>
          <span>{copy.evidence}</span>
        </div>
        <span className={cn("tu-g", GAIN_TONE[copy.tone])}>{copy.gain}</span>
        <div className="nd-a">
          <Button
            type="button"
            variant="ghost" size="icon-sm" className="text-muted-foreground hover:text-foreground"
            title={t("Dismiss for 30 days")}
            aria-label={t("Dismiss for 30 days")}
            disabled={busy || !canDecide}
            onClick={() => dismiss.mutate({ id: tuneUp.id, version: tuneUp.version })}
          >
            <Ic n="x" s={13} />
          </Button>
          <Button
            type="button"
            variant="outline" size="sm"
            disabled={busy || !canDecide || !canChange}
            isLoading={apply.isPending}
            loadingText={copy.action}
            onClick={() => apply.mutate({ id: tuneUp.id, version: tuneUp.version })}
          >
            {copy.action}
          </Button>
        </div>
      </div>
    </div>
  );
}
