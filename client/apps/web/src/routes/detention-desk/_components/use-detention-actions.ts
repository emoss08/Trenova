import { handleMutationError, useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { pluralize } from "@trenova/shared/lib/utils";
import type { DetentionOccurrence } from "@trenova/shared/types/detention";
import { useCallback } from "react";
import { toast } from "sonner";

/**
 * Sending a notice writes evidence, changes the occurrence, and moves the stop
 * between desk lanes — so every detention query is stale afterwards, not just
 * the one the caller happened to be looking at.
 */
export function useInvalidateDetention() {
  const queryClient = useQueryClient();

  return useCallback(
    () => void queryClient.invalidateQueries({ queryKey: queries.detention._def }),
    [queryClient],
  );
}

/** The single-stop notice action, shared by the card, the table, and the claim file. */
export function useSendDetentionNotice(occurrenceId: string) {
  const invalidate = useInvalidateDetention();

  return useApiMutation<DetentionOccurrence, undefined>({
    mutationFn: () => apiService.detentionService.sendNotice(occurrenceId),
    onSuccess: () => {
      toast.success("Notice sent", {
        description: "The customer notice went out and was recorded as evidence.",
      });
      invalidate();
    },
    resourceName: "Detention Notice",
  });
}

export type BulkNoticeResult = {
  sent: number;
  failed: number;
};

/**
 * Clears the notice queue in one action. Failures are counted rather than
 * thrown: one bounced recipient must not hide the fact that the other nine
 * notices went out and are now defensible.
 */
export function useSendDetentionNotices() {
  const invalidate = useInvalidateDetention();

  return useMutation<BulkNoticeResult, unknown, string[]>({
    mutationFn: async (occurrenceIds) => {
      const results = await Promise.allSettled(
        occurrenceIds.map((id) => apiService.detentionService.sendNotice(id)),
      );

      const sent = results.filter((result) => result.status === "fulfilled").length;

      return { sent, failed: results.length - sent };
    },
    onSuccess: ({ sent, failed }) => {
      if (sent > 0) {
        toast.success(`${sent} ${pluralize("notice", sent)} sent`, {
          description:
            failed > 0
              ? `${failed} could not be sent and ${failed === 1 ? "is" : "are"} still in the queue.`
              : "Every stop in the notice window is now on record.",
        });
      }

      if (sent === 0 && failed > 0) {
        toast.error("No notices could be sent", {
          description: "The stops are still in the notice window — try again in a moment.",
        });
      }

      invalidate();
    },
    onError: (error) => handleMutationError({ error, resourceName: "Detention Notices" }),
  });
}
