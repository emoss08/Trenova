import { useApiMutation } from "@/hooks/use-api-mutation";
import { formatUsd } from "@/lib/ai-usage-format";
import {
  AI_RETRIEVAL_FAILED_LIST_KEY,
  reindexAIRetrievalSource,
  type AIRetrievalReindexEstimate,
  type AIRetrievalSourceType,
  type AIRetrievalStatus,
} from "@/lib/graphql/ai-retrieval";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { CircleAlertIcon } from "lucide-react";
import { toast } from "sonner";
import { SOURCE_LABEL, estimateExceedsBudget } from "./retrieval-model";

type ReindexDialogProps = {
  sourceType: AIRetrievalSourceType | null;
  paused: boolean;
  onClose: () => void;
};

/**
 * Asks before re-indexing a source, with what it could cost. The estimate is
 * the most a full re-embed would cost at the provider's input price; the
 * indexer embeds again only the chunks whose text changed.
 */
export function ReindexDialog({ sourceType, paused, onClose }: ReindexDialogProps) {
  return (
    <Dialog open={sourceType !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent size="md">
        {sourceType ? (
          <ReindexConfirm sourceType={sourceType} paused={paused} onClose={onClose} />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function ReindexConfirm({
  sourceType,
  paused,
  onClose,
}: {
  sourceType: AIRetrievalSourceType;
  paused: boolean;
  onClose: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const label = t(SOURCE_LABEL[sourceType].label);
  const estimate = useQuery({
    ...queries.aiRetrieval.reindexEstimate(sourceType),
    staleTime: 0,
  });

  const reindex = useApiMutation<AIRetrievalStatus, AIRetrievalSourceType>({
    mutationFn: (type) => reindexAIRetrievalSource(type),
    onSuccess: async (status) => {
      queryClient.setQueryData(queries.aiRetrieval.status().queryKey, status);
      await queryClient.invalidateQueries({ queryKey: [AI_RETRIEVAL_FAILED_LIST_KEY] });
      toast.success(t("Re-indexing {0}", label));
      onClose();
    },
    resourceName: t("Re-index"),
  });

  return (
    <>
      <DialogHeader>
        <DialogTitle>{t("Re-index {0}?", label)}</DialogTitle>
        <DialogDescription>
          {t(
            "Every item is checked again. Only chunks whose text changed since they were indexed are embedded, so the estimate below is the most it could cost.",
          )}
        </DialogDescription>
      </DialogHeader>

      {estimate.isError ? (
        <Alert variant="destructive" size="sm">
          <CircleAlertIcon />
          <AlertDescription>{t("The cost estimate could not be loaded.")}</AlertDescription>
        </Alert>
      ) : estimate.data ? (
        <EstimateDetails estimate={estimate.data} />
      ) : (
        <div className="flex flex-col gap-2" aria-busy>
          <Skeleton className="h-5" />
          <Skeleton className="h-5" />
          <Skeleton className="h-5" />
        </div>
      )}

      {paused ? (
        <Alert variant="info" size="sm">
          <AlertDescription>
            {t("Indexing is paused, so the re-index waits until it resumes.")}
          </AlertDescription>
        </Alert>
      ) : null}

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={reindex.isPending}>
          {t("Cancel")}
        </Button>
        <Button
          type="button"
          onClick={() => reindex.mutate(sourceType)}
          disabled={reindex.isPending}
          isLoading={reindex.isPending}
        >
          {t("Re-index")}
        </Button>
      </DialogFooter>
    </>
  );
}

function EstimateDetails({ estimate }: { estimate: AIRetrievalReindexEstimate }) {
  const t = useT();
  const cost = estimate.estimatedCostUsd === null ? null : formatUsd(estimate.estimatedCostUsd);
  const price =
    estimate.inputCostPerMillionUsd === null ? null : formatUsd(estimate.inputCostPerMillionUsd);

  return (
    <div className="flex flex-col gap-2">
      <DescriptionList layout="split">
        <DescriptionItem label={t("Items to embed")} numeric>
          {formatNumber(estimate.sources)}
        </DescriptionItem>
        <DescriptionItem label={t("Chunks per item")} numeric>
          {estimate.chunksMeasured
            ? t("{0}, measured from indexed items", formatNumber(estimate.averageChunks))
            : t("{0}, estimated from their text", formatNumber(estimate.averageChunks))}
        </DescriptionItem>
        <DescriptionItem label={t("Tokens per chunk")} numeric>
          {formatNumber(estimate.averageTokensPerChunk)}
        </DescriptionItem>
        <DescriptionItem label={t("Tokens in all")} numeric>
          {formatNumber(estimate.estimatedTokens)}
        </DescriptionItem>
        <DescriptionItem label={t("Model")}>
          {estimate.modelKey ? (
            <span className="font-mono text-xs break-all">{estimate.modelKey}</span>
          ) : (
            <DescriptionEmpty />
          )}
        </DescriptionItem>
        <DescriptionItem label={t("Input price per million tokens")} numeric>
          {price ?? t("Not priced")}
        </DescriptionItem>
        <DescriptionItem label={t("Estimated cost, at most")} numeric>
          {cost ?? t("Unknown")}
        </DescriptionItem>
        <DescriptionItem label={t("Left of this month's budget")} numeric>
          {formatUsd(estimate.remainingBudgetUsd) ?? "—"}
        </DescriptionItem>
      </DescriptionList>
      {cost === null ? (
        <p className="text-muted-foreground text-xs">
          {estimate.modelKey
            ? t("The provider has no input price, so the cost is unknown until it is priced.")
            : t("No embedding model is routed, so nothing would be embedded yet.")}
        </p>
      ) : null}
      {estimateExceedsBudget(estimate) ? (
        <Alert variant="warning" size="sm">
          <CircleAlertIcon />
          <AlertDescription>
            {t(
              "This could cost more than is left of this month's budget. Indexing pauses when the budget is spent and resumes next month or when the budget is raised.",
            )}
          </AlertDescription>
        </Alert>
      ) : null}
    </div>
  );
}
