import { fetchMyPortalFeatures, type PortalFeatures } from "@/lib/graphql/driver-portal";
import { useQuery } from "@tanstack/react-query";

const ALL_ENABLED: PortalFeatures = {
  requireLoadAcknowledgment: true,
  allowLoadRefusals: true,
  allowStopActions: true,
  allowLoadDocumentUpload: true,
  allowLoadComments: true,
  showLoadPay: true,
  showPayEstimates: true,
  allowExpenseSubmission: true,
  requireExpenseReceipt: false,
  allowSettlementDisputes: true,
  allowProfileDocumentUpload: true,
  allowContactInfoEdit: true,
  allowPtoRequests: true,
};

// The server enforces every toggle; this only decides what UI to render, so
// falling back to all-enabled while loading just means a button may appear a
// beat early — never that a driver can do something the carrier disabled.
export function useDashFeatures(): PortalFeatures {
  const features = useQuery({
    queryKey: ["dash-features"],
    queryFn: fetchMyPortalFeatures,
    staleTime: 5 * 60 * 1000,
  });
  return features.data ?? ALL_ENABLED;
}
