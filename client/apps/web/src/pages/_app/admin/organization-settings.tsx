import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { OrganizationSettingsPage, prefetch } from "@/routes/admin/organization-settings/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/organization-settings")({
  beforeLoad: gates(requirePermission(Resource.Organization, Operation.Read)),
  component: OrganizationSettingsPage,
  loader: pageLoader(prefetch),
});
