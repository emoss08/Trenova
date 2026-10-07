import {
  dismissAIProviderFailure,
  restoreAIProviderFailure,
  type AIProviderFailure,
} from "@/lib/graphql/ai-control";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { toast } from "sonner";
import { Ic } from "../kit/ic";
import { Mark } from "../kit/marks";

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
    <>
      {failures.map((failure) => (
        <FailureRow key={failure.providerId} failure={failure} onEditProvider={onEditProvider} />
      ))}
    </>
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

  const testing = test.isPending;

  return (
    <div className="ovb" role="status">
      <span className="ovb-i">
        <Mark provider={provider ?? { name: failure.name }} s={28} />
        <i />
      </span>
      <div className="ovb-t">
        <b>{t("{0} can't connect", failure.name)}</b>
        <span>
          {failure.failedCalls === 1
            ? t("1 failed call in the last day · its tasks fall to the next provider in line")
            : t(
                "{0} failed calls in the last day · its tasks fall to the next provider in line",
                failure.failedCalls.toLocaleString(),
              )}
        </span>
      </div>
      <div className="ovb-a">
        <button type="button" className="btn sm" disabled={testing} onClick={() => test.mutate()}>
          {testing ? (
            <>
              <i className="spn" />
              {t("Testing")}
            </>
          ) : (
            <>
              <Ic n="plug" s={12} />
              {t("Test again")}
            </>
          )}
        </button>
        <button type="button" className="btn sm" onClick={() => onEditProvider(failure.providerId)}>
          {t("Edit connection")}
        </button>
        <button
          type="button"
          className="ib"
          title={t("Hide until it fails again")}
          aria-label={t("Hide until it fails again")}
          disabled={dismiss.isPending}
          onClick={() => dismiss.mutate()}
        >
          <Ic n="x" s={13} />
        </button>
      </div>
    </div>
  );
}
