import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { DataRetentionPage, prefetch } from "@/routes/organization/data-retention/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/organization/data-retention")({
  beforeLoad: gates(requirePermission(Resource.Organization)),
  component: DataRetentionPage,
  loader: pageLoader(prefetch),
});
