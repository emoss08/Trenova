import { createFileRoute } from "@tanstack/react-router";
import { requireCapability, requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { RandomTestingPage, prefetch } from "@/routes/random-testing/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/hr/random-testing")({
  beforeLoad: gates(
    requireCapability(OrganizationCapability.AssetOperations),
    requirePermission(Resource.DOTRandomPool),
  ),
  component: RandomTestingPage,
  loader: pageLoader(prefetch),
});
