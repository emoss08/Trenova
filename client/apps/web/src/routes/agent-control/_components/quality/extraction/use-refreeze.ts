import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  EXTRACTION_EVAL_CASE_LIST_KEY,
  EXTRACTION_EVAL_REFREEZE_KEY,
  fetchExtractionEvalRefreeze,
  refreezeExtractionEvalCases,
} from "@/lib/graphql/extraction-eval";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useRef } from "react";
import { ACTIVE_RUN_POLL_MS } from "./extraction-model";

/**
 * The organization's latest re-freeze through Docling, read again while it runs.
 * When it finishes the case list is refreshed, since the cases now carry Docling
 * pages.
 */
export function useRefreezeProgress() {
  const queryClient = useQueryClient();
  const progress = useQuery({
    queryKey: [EXTRACTION_EVAL_REFREEZE_KEY],
    queryFn: ({ signal }) => fetchExtractionEvalRefreeze({ signal }),
    refetchInterval: (query) =>
      query.state.data?.status === "Running" ? ACTIVE_RUN_POLL_MS : false,
  });

  const status = progress.data?.status;
  const previous = useRef(status);
  useEffect(() => {
    if (previous.current === "Running" && status !== "Running") {
      void queryClient.invalidateQueries({ queryKey: [EXTRACTION_EVAL_CASE_LIST_KEY] });
    }
    previous.current = status;
  }, [queryClient, status]);

  return progress;
}

/**
 * Starts converting every active case's original file through Docling. The
 * server keeps one re-freeze per organization, so a second click while one runs
 * returns that one rather than starting another.
 */
export function useRefreezeCases() {
  const t = useT();
  const queryClient = useQueryClient();

  return useApiMutation({
    mutationFn: () => refreezeExtractionEvalCases({}),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [EXTRACTION_EVAL_REFREEZE_KEY] }),
    resourceName: t("Docling re-conversion"),
  });
}
