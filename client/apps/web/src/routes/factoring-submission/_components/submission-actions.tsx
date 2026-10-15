import { retryFactoringSubmission, type FactoringSubmission } from "@/lib/graphql/factoring";
import { invalidateFactoringQueries } from "@/lib/queries/factoring-roots";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { RefreshCw01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { toast } from "sonner";

/**
 * Sends a failed schedule again. The server reads the factor's delivery
 * settings as they are now, so a fixed address or host is picked up.
 */
export function RetryFactoringSubmissionButton({
  submission,
}: {
  submission: FactoringSubmission;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const retry = useMutation({
    mutationFn: () => retryFactoringSubmission({ id: submission.id, version: submission.version }),
    onSuccess: (sent) => {
      invalidateFactoringQueries(queryClient);
      toast.success(t("Schedule {0} is being sent again", sent.number), {
        description: t("To {0}", sent.transportTarget),
      });
    },
    onError: (error) => {
      invalidateFactoringQueries(queryClient);
      const described = describeError(error);
      toast.error(t("The schedule was not retried"), {
        description: described.detail ?? described.message,
      });
    },
  });

  return (
    <Button
      type="button"
      size="sm"
      isLoading={retry.isPending}
      loadingText={t("Retrying...")}
      onClick={() => retry.mutate()}
    >
      <RefreshCw01Icon className="size-3.5" />
      {t("Retry")}
    </Button>
  );
}
