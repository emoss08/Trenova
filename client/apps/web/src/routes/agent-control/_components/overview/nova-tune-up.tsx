import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { patchAIProvider, type AIProviderPatch } from "@/lib/graphql/ai-provider";
import { aiControl } from "@/lib/queries/ai-control";
import { aiProvider } from "@/lib/queries/ai-provider";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { novaSetup, type NovaSetup } from "./nova-setup";
import { titled } from "./tune-up-copy";
import { invalidateAIControlCounts } from "./use-ai-control-stats";

/**
 * The tune-up that gets Nova running: run it on an Anthropic provider the organization
 * already has, turn its provider back on, or add one. It is never put away, because
 * Nova does nothing until it is done.
 */
export function useNovaTuneUp(): NovaSetup | null {
  const providersQuery = useQuery(aiProvider.list());
  const catalogQuery = useQuery(aiProvider.catalog());

  return useMemo(() => {
    if (!providersQuery.data || !catalogQuery.data) {
      return null;
    }
    const setup = novaSetup(providersQuery.data, catalogQuery.data.nova.platforms);

    return setup.state === "ready" ? null : setup;
  }, [catalogQuery.data, providersQuery.data]);
}

export function NovaTuneUp({ setup }: { setup: NovaSetup }) {
  const t = useT();
  const queryClient = useQueryClient();
  const go = useAIControlNavigation();
  const catalogQuery = useQuery(aiProvider.catalog());
  const model = catalogQuery.data?.nova.model ?? "";
  const { allowed: canUpdate } = usePermission(Resource.AIProvider, Operation.Update);
  const { allowed: canCreate } = usePermission(Resource.AIProvider, Operation.Create);

  const save = useApiMutation({
    mutationFn: (input: { provider: { id: string; version: number }; patch: AIProviderPatch }) =>
      patchAIProvider(input.provider, input.patch),
    resourceName: t("AI provider"),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: aiProvider._def }),
        queryClient.invalidateQueries({ queryKey: aiControl._def }),
        invalidateAIControlCounts(queryClient),
      ]);
    },
  });

  const row = (() => {
    switch (setup.state) {
      case "choose":
        return {
          title: titled(t("Run Nova on {0}", "{0}"), setup.provider.name),
          evidence: t(
            "Nova reads paperwork, reviews the billing queue, sorts mail and writes briefings on {0}, with this provider's key, limits and cap.",
            model,
          ),
          gain: t("Turns Nova on"),
          action: t("Run Nova here"),
          allowed: canUpdate,
          onClick: () => save.mutate({ provider: setup.provider, patch: { nova: true } }),
        };
      case "paused":
        return {
          title: titled(t("Turn on {0}", "{0}"), setup.provider.name),
          evidence: t("It runs Nova, so Nova is paused while it is off."),
          gain: t("Resumes Nova"),
          action: t("Turn on"),
          allowed: canUpdate,
          onClick: () => save.mutate({ provider: setup.provider, patch: { enabled: true } }),
        };
      case "connect":
        return {
          title: titled(t("Add an Anthropic provider for {0}", "{0}"), t("Nova")),
          evidence: t(
            "Nova runs on {0} through the Claude API, Claude Platform on AWS or Google Cloud, or Vertex AI, with the key of the provider you add.",
            model,
          ),
          gain: t("Turns Nova on"),
          action: t("Add provider"),
          allowed: canCreate,
          onClick: () => go({ tab: "providers", panel: { mode: "create", preset: "anthropic" } }),
        };
      default:
        return null;
    }
  })();

  if (!row) {
    return null;
  }

  return (
    <div className="nd-w">
      <div className="nd tu">
        <span className="grid size-7.5 place-items-center">
          <NovaBadge size="xl" />
        </span>
        <div className="nd-b">
          <b className="tu-t">
            <span>
              {row.title.map((part, index) =>
                part.strong ? <b key={index}>{part.text}</b> : <span key={index}>{part.text}</span>,
              )}
            </span>
          </b>
          <span>{row.evidence}</span>
        </div>
        <span className="tu-g b">{row.gain}</span>
        <div className="nd-a">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={!row.allowed || save.isPending}
            isLoading={save.isPending}
            loadingText={row.action}
            onClick={row.onClick}
          >
            {row.action}
          </Button>
        </div>
      </div>
    </div>
  );
}
