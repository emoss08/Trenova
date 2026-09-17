import { useT } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Card } from "@trenova/shared/components/ui/card";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { fetchAgentControl, updateAgentControl } from "@/lib/graphql/agent-control";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { PauseCircleIcon } from "lucide-react";
import { useCallback } from "react";
import { toast } from "sonner";

const AGENT_CONTROL_QUERY_KEY = ["agent-control"];

export default function AgentControlForm() {
  const t = useT();
  const queryClient = useQueryClient();

  const { data } = useSuspenseQuery({
    queryKey: AGENT_CONTROL_QUERY_KEY,
    queryFn: ({ signal }) => fetchAgentControl({ signal }),
  });

  const { allowed: canUpdate } = usePermission(Resource.AgentControl, Operation.Update);

  const mutation = useApiMutation({
    mutationFn: (shadowMode: boolean) => updateAgentControl({ shadowMode }),
    onSuccess: async (_result, shadowMode) => {
      toast.success(shadowMode ? t("All agents paused") : t("Agents resumed"));
      await queryClient.invalidateQueries({ queryKey: AGENT_CONTROL_QUERY_KEY });
    },
    resourceName: "Agent Control",
  });

  const onChange = useCallback((checked: boolean) => mutation.mutate(checked), [mutation]);

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
            onCheckedChange={onChange}
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
    </div>
  );
}
