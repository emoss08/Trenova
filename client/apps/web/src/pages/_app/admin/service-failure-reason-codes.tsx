import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { ServiceFailureReasonCodesPage } from "@/routes/service-failure-reason-code/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/service-failure-reason-codes")({
  beforeLoad: gates(requirePermission(Resource.ServiceFailureReasonCode)),
  component: ServiceFailureReasonCodesPage,
});
