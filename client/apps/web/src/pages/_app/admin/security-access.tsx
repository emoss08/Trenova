import { createFileRoute } from "@tanstack/react-router";
import { anyOfGates, requirePermission } from "@/lib/route-permission";
import { SecurityAccessPage } from "@/routes/admin/security-access/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

// Each part of the page answers to its own permission and is shown to whoever may read it,
// so the page opens for anyone who may read one of them.
export const Route = createFileRoute("/_app/admin/security-access")({
  beforeLoad: gates(
    anyOfGates(
      requirePermission(Resource.Organization, Operation.Read),
      requirePermission(Resource.IdentityProvider, Operation.Read),
      requirePermission(Resource.SCIMDirectory, Operation.Read),
      requirePermission(Resource.ProvisioningAudit, Operation.Read),
      requirePermission(Resource.AccessPolicy, Operation.Read),
      requirePermission(Resource.AuthEvent, Operation.Read),
      requirePermission(Resource.RiskDecision, Operation.Read),
      requirePermission(Resource.ExternalIdentity, Operation.Read),
      requirePermission(Resource.MFAAuthenticator, Operation.Read),
    ),
  ),
  component: SecurityAccessPage,
});
