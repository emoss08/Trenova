import {
  dismissAIProviderFailure,
  restoreAIProviderFailure,
  type AIProviderFailure,
} from "@/lib/graphql/ai-control";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { PlugIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { toast } from "sonner";
import { ProviderMark } from "../providers/provider-mark";

type ProviderFailureStripProps = {
  failures: readonly AIProviderFailure[];
  onEditProvider: (providerId: string) => void;
};

/**
 * A provider that is on and failing, and what to do about it: test it again, fix its
 * connection, or put the notice away until it fails again. Only shown while one fails.
 */
export function ProviderFailureStrip({ failures, onEditProvider }: ProviderFailureStripProps) {
  if (failures.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-2">
      {failures.map((failure) => (
        <FailureRow key={failure.providerId} failure={failure} onEditProvider={onEditProvider} />
      ))}
    </div>
  );
}

function FailureRow({
  failure,
  onEditProvider,
}: {
  failure: AIProviderFailure;
  onEditProvider: (providerId: string) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const providersQuery = useQuery(queries.aiProvider.list());
  const catalogQuery = useQuery(queries.aiProvider.catalog());
  const provider = providersQuery.data?.find((candidate) => candidate.id === failure.providerId);

  const refresh = useCallback(
    () => queryClient.invalidateQueries({ queryKey: queries.aiControl._def }),
    [queryClient],
  );

  const test = useApiMutation({
    mutationFn: () => apiService.aiProviderService.test(failure.providerId),
    resourceName: t("AI provider"),
    onSuccess: async (result) => {
      if (result.success) {
        toast.success(t("{0} answered", failure.name));
      } else {
        toast.error(result.message, { description: result.detail || undefined });
      }
      await refresh();
    },
  });

  const restore = useApiMutation({
    mutationFn: () => restoreAIProviderFailure(failure.providerId),
    resourceName: t("AI provider"),
    onSuccess: refresh,
  });

  const dismiss = useApiMutation({
    mutationFn: () => dismissAIProviderFailure(failure.providerId, failure.lastFailureAt),
    resourceName: t("AI provider"),
    onSuccess: async () => {
      await refresh();
      toast(t("Hidden until {0} fails again", failure.name), {
        action: { label: t("Undo"), onClick: () => restore.mutate() },
      });
    },
  });

  return (
    <div
      role="status"
      className="flex flex-wrap items-center gap-3 rounded-surface border border-danger-border bg-danger-subtle px-4 py-3"
    >
      <span className="relative shrink-0">
        {provider ? (
          <ProviderMark provider={provider} presets={catalogQuery.data?.presets ?? []} />
        ) : (
          <span className="block size-7 rounded-md bg-muted" />
        )}
        <span className="absolute -top-0.5 -right-0.5 size-2.5 rounded-full border-2 border-card bg-danger motion-safe:animate-pulse" />
      </span>
      <div className="flex min-w-0 flex-1 flex-col">
        <b className="text-sm font-semibold">{t("{0} can't connect", failure.name)}</b>
        <span className="text-xs text-muted-foreground">
          {failure.failedCalls === 1
            ? t("1 failed call in the last day · its tasks fall to the next provider in line")
            : t(
                "{0} failed calls in the last day · its tasks fall to the next provider in line",
                failure.failedCalls.toLocaleString(),
              )}
        </span>
      </div>
      <div className="flex items-center gap-1.5">
        <Button
          size="sm"
          variant="outline"
          isLoading={test.isPending}
          loadingText={t("Testing")}
          onClick={() => test.mutate()}
        >
          <PlugIcon className="size-3" />
          {t("Test again")}
        </Button>
        <Button size="sm" variant="outline" onClick={() => onEditProvider(failure.providerId)}>
          {t("Edit connection")}
        </Button>
        <Button
          size="icon-sm"
          variant="ghost"
          aria-label={t("Hide until it fails again")}
          disabled={dismiss.isPending}
          onClick={() => dismiss.mutate()}
        >
          <XCloseIcon className="size-3.5" />
        </Button>
      </div>
    </div>
  );
}
