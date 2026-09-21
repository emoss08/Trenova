import { useT } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Card } from "@trenova/shared/components/ui/card";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  AGENT_CONTROL_QUERY_KEY,
  agentControlQueryOptions,
  updateAgentControl,
  type AgentControl,
} from "@/lib/graphql/agent-control";
import type { AgentControlInput } from "@trenova/graphql/generated/graphql";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { AwardIcon, PauseCircleIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { promotionThresholdOptions } from "./agent-control-options";

type ControlPatch = Partial<
  Pick<AgentControlInput, "shadowMode" | "earnedAutonomy" | "promotionThreshold">
>;

/** The input the mutation sends: the current switches with one of them changed. */
function controlInput(current: AgentControl, patch: ControlPatch): AgentControlInput {
  return {
    shadowMode: patch.shadowMode ?? current.shadowMode,
    earnedAutonomy: patch.earnedAutonomy ?? current.earnedAutonomy,
    promotionThreshold: patch.promotionThreshold ?? current.promotionThreshold,
  };
}

export default function AgentControlForm() {
  const t = useT();
  const queryClient = useQueryClient();

  const { data } = useSuspenseQuery(agentControlQueryOptions());

  const { allowed: canUpdate } = usePermission(Resource.AgentControl, Operation.Update);

  const mutation = useApiMutation({
    mutationFn: (patch: ControlPatch) => updateAgentControl(controlInput(data, patch)),
    onSuccess: async (_result, patch) => {
      if (patch.shadowMode !== undefined) {
        toast.success(patch.shadowMode ? t("All agents paused") : t("Agents resumed"));
      } else if (patch.earnedAutonomy !== undefined) {
        toast.success(patch.earnedAutonomy ? t("Earned autonomy on") : t("Earned autonomy off"));
      } else {
        toast.success(t("Promotion threshold saved"));
      }
      await queryClient.invalidateQueries({ queryKey: AGENT_CONTROL_QUERY_KEY });
    },
    resourceName: "Agent Control",
  });

  const onShadow = useCallback(
    (checked: boolean) => mutation.mutate({ shadowMode: checked }),
    [mutation],
  );
  const onEarned = useCallback(
    (checked: boolean) => mutation.mutate({ earnedAutonomy: checked }),
    [mutation],
  );
  const onThreshold = useCallback(
    (value: string) => mutation.mutate({ promotionThreshold: Number(value) }),
    [mutation],
  );

  const thresholdLocked = !canUpdate || !data.earnedAutonomy || mutation.isPending;
  const thresholdItems = useMemo(
    () =>
      promotionThresholdOptions(data.promotionThreshold).map((option) => ({
        value: String(option.value),
        label: option.label,
        disabled: thresholdLocked,
      })),
    [data.promotionThreshold, thresholdLocked],
  );

  return (
    <div className="flex flex-col gap-3">
      <Card size="sm" className="gap-3 px-4 py-3">
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-3">
            <span className="bg-muted text-muted-foreground flex size-9 shrink-0 items-center justify-center rounded-lg">
              <PauseCircleIcon className="size-4" />
            </span>
            <div className="max-w-prose">
              <p className="text-sm font-semibold">{t("Pause all agents")}</p>
              <p className="text-muted-foreground text-xs">
                {t(
                  "Agents keep running and keep recording what they would do, but nothing they propose is offered for a decision and nothing they propose can be executed. Each agent also has its own shadow switch.",
                )}
              </p>
            </div>
          </div>
          <Switch
            checked={data.shadowMode}
            disabled={!canUpdate || mutation.isPending}
            onCheckedChange={onShadow}
            aria-label={t("Pause all agents")}
          />
        </div>
      </Card>

      {data.shadowMode && (
        <Alert variant="warning">
          <AlertTitle>{t("Proposals are hidden")}</AlertTitle>
          <AlertDescription>
            {t(
              "Every agent in this organization is paused. Runs finish and their proposals are stored for review later, but none appear for a decision. Turn this off to let agents surface their work.",
            )}
          </AlertDescription>
        </Alert>
      )}

      <Card size="sm" className="gap-3 px-4 py-3">
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-3">
            <span className="bg-muted text-muted-foreground flex size-9 shrink-0 items-center justify-center rounded-lg">
              <AwardIcon className="size-4" />
            </span>
            <div className="max-w-prose">
              <p className="text-sm font-semibold">{t("Earned autonomy")}</p>
              <p className="text-muted-foreground text-xs">
                {t(
                  "When a tool's proposals are approved unchanged this many times in a row, the tool moves up one tier on that agent, never above the agent's ceiling. A rejection or a failed run takes an earned tier back. Each change is audited and announced.",
                )}
              </p>
            </div>
          </div>
          <Switch
            checked={data.earnedAutonomy}
            disabled={!canUpdate || mutation.isPending}
            onCheckedChange={onEarned}
            aria-label={t("Earned autonomy")}
          />
        </div>
        <div className="flex flex-wrap items-center justify-between gap-3 pl-12">
          <span className="text-muted-foreground text-xs">
            {t("Clean approvals in a row before a tool moves up")}
          </span>
          <div className="w-full max-w-xs">
            <SegmentedControl<string>
              fullWidth
              aria-label={t("Promotion threshold")}
              value={String(data.promotionThreshold)}
              onValueChange={onThreshold}
              items={thresholdItems}
            />
          </div>
        </div>
      </Card>
    </div>
  );
}
