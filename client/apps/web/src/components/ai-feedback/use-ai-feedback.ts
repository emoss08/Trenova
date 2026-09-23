import {
  AI_FEEDBACK_BATCH_LIMIT,
  aiFeedbackTargetKey,
  clearMyAiFeedback,
  fetchMyAiFeedback,
  setMyAiFeedback,
  type AiFeedback,
  type AiFeedbackRating,
  type AiFeedbackSubmission,
  type AiFeedbackTarget,
  type AiFeedbackTargetType,
} from "@/lib/graphql/ai-feedback";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { toast } from "sonner";

export const AI_FEEDBACK_QUERY_ROOT = ["ai-feedback", "mine"] as const;

type Waiter = {
  resolve: (feedback: AiFeedback | null) => void;
  reject: (error: unknown) => void;
  signal?: AbortSignal;
  settled: boolean;
};

type Queued = { target: AiFeedbackTarget; waiters: Waiter[] };

type Fetcher = (
  targets: readonly AiFeedbackTarget[],
  options?: { signal?: AbortSignal },
) => Promise<AiFeedback[]>;

/**
 * Coalesces every rating a screen asks for in one pass into one request.
 *
 * A thread draws a control under each answer and a list draws one per row;
 * each control asks for its own rating, and the loader answers all of them
 * from a single myAIFeedback read per tick, split only at the server's limit.
 * A request is cancelled only once every query waiting on it has been.
 */
export function createAiFeedbackLoader(fetcher: Fetcher = fetchMyAiFeedback) {
  let queue = new Map<string, Queued>();
  let scheduled = false;

  const settle = async (batch: Queued[]) => {
    const waiters = batch.flatMap((entry) => entry.waiters);
    const waiting = () => waiters.some((waiter) => !waiter.settled);
    if (!waiting()) {
      return;
    }

    const controller = new AbortController();
    for (const waiter of waiters) {
      waiter.signal?.addEventListener(
        "abort",
        () => {
          if (!waiting()) {
            controller.abort();
          }
        },
        { once: true },
      );
    }

    try {
      const rows = await fetcher(
        batch.map((entry) => entry.target),
        { signal: controller.signal },
      );
      const byKey = new Map(rows.map((row) => [aiFeedbackTargetKey(row), row]));
      for (const entry of batch) {
        const found = byKey.get(aiFeedbackTargetKey(entry.target)) ?? null;
        for (const waiter of entry.waiters) {
          if (!waiter.settled) {
            waiter.settled = true;
            waiter.resolve(found);
          }
        }
      }
    } catch (error) {
      for (const waiter of waiters) {
        if (!waiter.settled) {
          waiter.settled = true;
          waiter.reject(error);
        }
      }
    }
  };

  const flush = () => {
    scheduled = false;
    const entries = [...queue.values()];
    queue = new Map();
    for (let start = 0; start < entries.length; start += AI_FEEDBACK_BATCH_LIMIT) {
      void settle(entries.slice(start, start + AI_FEEDBACK_BATCH_LIMIT));
    }
  };

  return {
    load(target: AiFeedbackTarget, signal?: AbortSignal): Promise<AiFeedback | null> {
      return new Promise((resolve, reject) => {
        if (signal?.aborted) {
          reject(signal.reason);

          return;
        }

        const waiter: Waiter = { resolve, reject, signal, settled: false };
        signal?.addEventListener(
          "abort",
          () => {
            if (!waiter.settled) {
              waiter.settled = true;
              reject(signal.reason);
            }
          },
          { once: true },
        );

        const key = aiFeedbackTargetKey(target);
        const queued = queue.get(key) ?? { target, waiters: [] };
        queued.waiters.push(waiter);
        queue.set(key, queued);
        if (!scheduled) {
          scheduled = true;
          setTimeout(flush, 0);
        }
      });
    },
  };
}

const loader = createAiFeedbackLoader();

export function aiFeedbackQueryKey(target: AiFeedbackTarget) {
  return [
    ...AI_FEEDBACK_QUERY_ROOT,
    target.targetType,
    target.targetId,
    target.targetPart ?? "",
  ] as const;
}

function aiFeedbackQueryOptions(
  targetType: AiFeedbackTargetType,
  targetId: string,
  targetPart: string,
) {
  return {
    queryKey: [...AI_FEEDBACK_QUERY_ROOT, targetType, targetId, targetPart] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      loader.load({ targetType, targetId, targetPart }, signal),
    staleTime: 5 * 60 * 1000,
  };
}

/** What a rating will read as before the server confirms it. */
export function optimisticFeedback(
  previous: AiFeedback | null,
  target: AiFeedbackTarget,
  submission: AiFeedbackSubmission,
): AiFeedback {
  return {
    id: previous?.id ?? `pending:${aiFeedbackTargetKey(target)}`,
    targetType: target.targetType,
    targetId: target.targetId,
    targetPart: target.targetPart ?? "",
    rating: submission.rating,
    reasons: submission.reasons,
    comment: submission.comment,
    version: previous?.version ?? 0,
    updatedAt: Math.floor(Date.now() / 1000),
  };
}

type MutationContext = { previous: AiFeedback | null };

/**
 * One person's rating of one thing an AI wrote: read with every other rating
 * on the screen in one request, changed optimistically, and put back as it
 * was when the server refuses the change.
 */
export function useAiFeedback(target: AiFeedbackTarget) {
  const t = useT();
  const queryClient = useQueryClient();
  const { targetType, targetId } = target;
  const targetPart = target.targetPart ?? "";
  const options = aiFeedbackQueryOptions(targetType, targetId, targetPart);
  const queryKey = options.queryKey;
  const query = useQuery(options);

  const mutation = useMutation<
    AiFeedback | null,
    unknown,
    AiFeedbackSubmission | null,
    MutationContext
  >({
    scope: { id: queryKey.join(":") },
    mutationFn: async (next) => {
      const current = { targetType, targetId, targetPart };
      if (next === null) {
        await clearMyAiFeedback(current);

        return null;
      }

      return setMyAiFeedback(current, next);
    },
    onMutate: async (next) => {
      await queryClient.cancelQueries({ queryKey });
      const previous = queryClient.getQueryData<AiFeedback | null>(queryKey) ?? null;
      queryClient.setQueryData<AiFeedback | null>(
        queryKey,
        next === null
          ? null
          : optimisticFeedback(previous, { targetType, targetId, targetPart }, next),
      );

      return { previous };
    },
    onError: (_error, _next, context) => {
      queryClient.setQueryData<AiFeedback | null>(queryKey, context?.previous ?? null);
      toast.error(t("Your rating could not be saved"));
    },
    onSuccess: (saved) => {
      queryClient.setQueryData<AiFeedback | null>(queryKey, saved);
    },
  });

  const { mutate } = mutation;
  const submit = useCallback(
    (submission: AiFeedbackSubmission, options?: { onError?: () => void }) =>
      mutate(submission, { onError: options?.onError }),
    [mutate],
  );
  const clear = useCallback(() => mutate(null), [mutate]);

  const feedback = query.data ?? null;

  return {
    feedback,
    rating: ratingOf(feedback),
    isLoading: query.isLoading,
    isSaving: mutation.isPending,
    submit,
    clear,
  };
}

function ratingOf(feedback: AiFeedback | null): AiFeedbackRating | null {
  if (feedback?.rating === 1 || feedback?.rating === -1) {
    return feedback.rating;
  }

  return null;
}
