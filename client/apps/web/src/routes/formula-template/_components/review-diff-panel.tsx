import { ReviewDiffView } from "@/components/tariff-studio/review-diff-view";
import { formulaTemplate } from "@/lib/queries/formula-template";
import { useQuery } from "@tanstack/react-query";

/** A template's review diff: its source and pinned libraries against the approved version. */
export function ReviewDiffPanel({ templateId }: { templateId: string }) {
  const { data, isLoading, isError } = useQuery({
    ...formulaTemplate.reviewDiff(templateId),
    enabled: !!templateId,
    staleTime: 0,
  });
  return <ReviewDiffView data={data} isLoading={isLoading} isError={isError} />;
}
