import { cloudTrialStatus, type CloudTrialStatus } from "@/lib/cloud-trial";
import { queries } from "@/lib/queries";
import type { BillingSummary } from "@/types/platform-billing";
import { useQuery } from "@tanstack/react-query";
import { usePublicConfig } from "@trenova/shared/hooks/use-public-config";
import { useNowSeconds } from "./use-now-seconds";

const BILLING_SUMMARY_STALE_MS = 5 * 60 * 1000;

export type CloudPlanState = {
  isCloud: boolean;
  summary: BillingSummary | undefined;
  trial: CloudTrialStatus | null;
  isLoading: boolean;
  isError: boolean;
  refetch: () => void;
  isFetching: boolean;
};

/**
 * The organization's plan, subscription and usage on Trenova Cloud. Outside cloud
 * mode nothing is fetched: every organization is unlimited and there is no trial.
 */
export function useCloudPlan(): CloudPlanState {
  const { isCloud } = usePublicConfig();
  const now = useNowSeconds();
  const summaryQuery = useQuery({
    ...queries.platformBilling.summary(),
    enabled: isCloud,
    staleTime: BILLING_SUMMARY_STALE_MS,
    retry: false,
  });

  return {
    isCloud,
    summary: summaryQuery.data,
    trial: isCloud ? cloudTrialStatus(summaryQuery.data, now) : null,
    isLoading: isCloud && summaryQuery.isPending,
    isError: summaryQuery.isError,
    refetch: () => void summaryQuery.refetch(),
    isFetching: summaryQuery.isFetching,
  };
}
