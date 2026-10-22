import { ReviewHistoryView } from "@/components/tariff-studio/review-history-view";
import { formulaTemplate } from "@/lib/queries/formula-template";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";

/** The template's review conversation, newest round first. */
export function ReviewHistory({ templateId }: { templateId: string }) {
  const t = useT();
  const { data, isLoading, isError } = useQuery({
    ...formulaTemplate.reviews(templateId),
    enabled: !!templateId,
    staleTime: 0,
  });

  return (
    <ReviewHistoryView
      reviews={data}
      isLoading={isLoading}
      isError={isError}
      emptyText={t("This template has never been submitted for review.")}
    />
  );
}
