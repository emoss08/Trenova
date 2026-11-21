import { useT } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { SecurityAccessWorkspace } from "./security-access-workspace";

/** The page's body: the security console for the organization this session is in. */
export default function SecurityAccessBody() {
  const t = useT();
  const organizationId = useAuthStore((state) => state.user?.currentOrganizationId);

  if (!organizationId) {
    return (
      <p className="py-8 text-sm text-muted-foreground">
        {t("No active organization found for this session.")}
      </p>
    );
  }

  return <SecurityAccessWorkspace organizationId={organizationId} />;
}
