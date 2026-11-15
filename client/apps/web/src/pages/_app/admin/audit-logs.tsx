import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { AuditLogsPage, prefetch } from "@/routes/admin/audit-logs/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/audit-logs")({
  beforeLoad: gates(requirePermission(Resource.AuditLog, Operation.Read)),
  component: AuditLogsPage,
  loader: pageLoader(prefetch),
});
