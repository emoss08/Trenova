import { HoldButton } from "@/components/hold-button";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { AGENT_CONTROL_QUERY_KEY, updateAgentControl, type AgentControl } from "@/lib/graphql/agent-control";
import { queries } from "@/lib/queries";
import { PauseCircleIcon, PlayCircleIcon, PlusIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { controlInput } from "../agent-control-options";

type OverviewControlProps = {
  control: AgentControl | undefined;
  noProvider: boolean;
  canUpdate: boolean;
  onConnectProvider: () => void;
};

/**
 * The one control beside the overview's sentence: connect a provider when there is none,
 * resume when every agent is paused, and otherwise a hold to pause them all.
 */
export function OverviewControl({
  control,
  noProvider,
  canUpdate,
  onConnectProvider,
}: OverviewControlProps) {
  const t = useT();
  const queryClient = useQueryClient();

  const pause = useApiMutation({
    mutationFn: (shadowMode: boolean) => {
      if (!control) {
        throw new Error("The organization's settings have not loaded");
      }
      return updateAgentControl(controlInput(control, { shadowMode }));
    },
    resourceName: t("Organization-wide settings"),
    onSuccess: async (saved, shadowMode) => {
      queryClient.setQueryData(AGENT_CONTROL_QUERY_KEY, saved);
      await queryClient.invalidateQueries({ queryKey: queries.aiControl._def });
      toast.success(
        shadowMode
          ? t("All agents paused · they keep running in shadow")
          : t("Agents resumed · held proposals are back in Desk"),
      );
    },
  });

  if (noProvider) {
    return (
      <>
        <Button size="lg" onClick={onConnectProvider}>
          <PlusIcon className="size-3.5" />
          {t("Connect a provider")}
        </Button>
        <span className="text-xs text-muted-foreground">{t("Takes about a minute")}</span>
      </>
    );
  }

  if (!control || !canUpdate) {
    return null;
  }

  if (control.shadowMode) {
    return (
      <>
        <Button size="lg" isLoading={pause.isPending} onClick={() => pause.mutate(false)}>
          <PlayCircleIcon className="size-3.5" />
          {t("Resume agents")}
        </Button>
        <span className="text-xs text-muted-foreground">{t("Every agent is paused")}</span>
      </>
    );
  }

  return (
    <>
      <div className="w-60">
        <HoldButton
          tone="warning"
          icon={PauseCircleIcon}
          label={t("Hold to pause all agents")}
          doneLabel={t("Pausing")}
          done={pause.isPending}
          disabled={pause.isPending}
          onConfirm={() => pause.mutate(true)}
        />
      </div>
      <span className="text-xs text-muted-foreground">{t("They keep running in shadow")}</span>
    </>
  );
}
