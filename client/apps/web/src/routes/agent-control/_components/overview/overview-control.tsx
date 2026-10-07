import { useApiMutation } from "@/hooks/use-api-mutation";
import { AGENT_CONTROL_QUERY_KEY, updateAgentControl, type AgentControl } from "@/lib/graphql/agent-control";
import { queries } from "@/lib/queries";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { controlInput } from "../agent-control-options";
import { Ic } from "../kit/ic";
import { Hold } from "../kit/layout";

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
        <button type="button" className="btn ink lg" onClick={onConnectProvider}>
          <Ic n="plus" s={13} />
          {t("Connect a provider")}
        </button>
        <span>{t("Takes about a minute")}</span>
      </>
    );
  }

  if (!control || !canUpdate) {
    return null;
  }

  if (control.shadowMode) {
    return (
      <>
        <button
          type="button"
          className="btn ink lg"
          disabled={pause.isPending}
          onClick={() => pause.mutate(false)}
        >
          <Ic n="play" s={12} />
          {t("Resume agents")}
        </button>
        <span>{t("Every agent is paused")}</span>
      </>
    );
  }

  return (
    <>
      <Hold
        label={t("Hold to pause all agents")}
        disabled={pause.isPending}
        onDone={() => pause.mutate(true)}
      />
      <span>{t("They keep running in shadow")}</span>
    </>
  );
}
