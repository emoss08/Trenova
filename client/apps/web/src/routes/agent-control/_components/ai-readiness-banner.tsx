import { useT } from "@trenova/shared/i18n/use-t";
import { queries } from "@/lib/queries";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useQuery } from "@tanstack/react-query";
import { CheckCircle2Icon, PlugZapIcon, TriangleAlertIcon } from "lucide-react";
import { assessReadiness } from "./ai-readiness";

type AIReadinessBannerProps = {
  onOpenProviders: () => void;
};

/**
 * Answers the question an administrator opening this page has: if I turn an
 * agent on, will it work?
 *
 * Agents and providers are configured separately on purpose — one is what an
 * agent may do, the other is where the work goes — but that split is exactly
 * what lets an organization enable an agent that can never answer. This joins
 * the two and says so before anyone finds out from a failed conversation.
 */
export function AIReadinessBanner({ onOpenProviders }: AIReadinessBannerProps) {
  const t = useT();

  const providersQuery = useQuery(queries.aiProvider.list());
  const catalogQuery = useQuery(queries.aiProvider.catalog());

  if (providersQuery.isLoading || catalogQuery.isLoading) {
    return <Skeleton className="h-20" />;
  }

  const readiness = assessReadiness(providersQuery.data ?? [], catalogQuery.data?.tasks ?? []);

  if (!readiness.hasProviders) {
    return (
      <Alert variant="warning">
        <PlugZapIcon className="size-4" />
        <AlertTitle>{t("No AI provider is connected")}</AlertTitle>
        <AlertDescription className="flex flex-col gap-3">
          <p>
            {t(
              "Agents, the assistant and insight explanations all route through a provider. Connect one — a hosted API, a gateway, or a model server on your own hardware — then switch agents on.",
            )}
          </p>
          <div>
            <Button size="sm" variant="outline" onClick={onOpenProviders}>
              <PlugZapIcon className="size-3.5" />
              {t("Connect a provider")}
            </Button>
          </div>
        </AlertDescription>
      </Alert>
    );
  }

  if (readiness.uncovered.length === 0) {
    return (
      <Alert>
        <CheckCircle2Icon className="size-4 text-emerald-500" />
        <AlertTitle>{t("Every AI task has a provider")}</AlertTitle>
        <AlertDescription>
          {t("Agents you enable will run. Routing is managed on the Providers tab.")}
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <Alert variant={readiness.assistantReady ? "default" : "warning"}>
      <TriangleAlertIcon className="size-4" />
      <AlertTitle>
        {readiness.assistantReady
          ? t("Some AI work has nowhere to go")
          : t("The assistant cannot answer yet")}
      </AlertTitle>
      <AlertDescription className="flex flex-col gap-3">
        <p>
          {readiness.assistantReady
            ? t(
                "Conversations will work, but the tasks below have no enabled provider assigned and will fail when something needs them.",
              )
            : t(
                "Assistant chat needs an enabled provider before a conversation can start. Assign it on a provider and the agents become usable.",
              )}
        </p>
        <div className="flex flex-wrap gap-1.5">
          {readiness.uncovered.map((task) => (
            <Badge key={task.task} variant="neutral" appearance="outline">
              {task.label}
              {task.requiresTrust ? ` · ${t("needs a trusted provider")}` : ""}
            </Badge>
          ))}
        </div>
        <div>
          <Button size="sm" variant="outline" onClick={onOpenProviders}>
            {t("Assign tasks to a provider")}
          </Button>
        </div>
      </AlertDescription>
    </Alert>
  );
}
