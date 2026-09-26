import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  approveJournalEntries,
  JOURNAL_REVIEW_TABLE_KEY,
  postJournalEntries,
  type JournalReviewResult,
} from "@/lib/graphql/journal-review";
import { journalReviewMessage, type JournalReviewAction } from "@/lib/journal-review";
import { queries } from "@/lib/queries";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback } from "react";
import { toast } from "sonner";

export function useJournalReviewActions() {
  const t = useT();
  const queryClient = useQueryClient();

  const report = useCallback(
    async (action: JournalReviewAction, result: JournalReviewResult) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queries.journalEntry._def }),
        queryClient.invalidateQueries({ queryKey: [JOURNAL_REVIEW_TABLE_KEY] }),
      ]);
      const message = journalReviewMessage(t, action, result);
      toast[message.tone](message.title, { description: message.description });
    },
    [queryClient, t],
  );

  const approve = useApiMutation({
    mutationFn: (entryIds: string[]) => approveJournalEntries(entryIds),
    resourceName: t("Journal entry"),
    onSuccess: (result) => report("approve", result),
  });

  const post = useApiMutation({
    mutationFn: (entryIds: string[]) => postJournalEntries(entryIds),
    resourceName: t("Journal entry"),
    onSuccess: (result) => report("post", result),
  });

  return { approve, post };
}
