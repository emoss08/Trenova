import { PageLayout } from "@/components/navigation/sidebar-layout";
import { usePermission } from "@/hooks/use-permission";
import type { SupportAccessGrant } from "@/lib/graphql/support-access";
import { supportAccess } from "@/lib/queries/support-access";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { PlusIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useCallback, useState } from "react";
import { GrantSupportDialog } from "./_components/grant-support-dialog";
import { RevokeSupportDialog } from "./_components/revoke-support-dialog";
import { SendLogsSection } from "./_components/send-logs-section";
import { SupportGrantList } from "./_components/support-grant-list";
import { SupportGrantsSkeleton } from "./_components/support-grant-skeleton";

export const prefetch: RoutePrefetch = () => [supportAccess.settings()];

/**
 * Who at Trenova may enter this organization, for how long and why, and what each did. Only a
 * person here grants access; staff never hold a password, and each entry is checked against a
 * live grant.
 */
export function SupportAccessPage() {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canGrant } = usePermission(Resource.SupportAccess, Operation.Create);
  const { allowed: canRevoke } = usePermission(Resource.SupportAccess, Operation.Delete);
  const { allowed: canSendLogs } = usePermission(Resource.Organization, Operation.Update);
  const [granting, setGranting] = useState(false);
  const [revoking, setRevoking] = useState<SupportAccessGrant | null>(null);

  const settingsQuery = useQuery(supportAccess.settings());
  const settings = settingsQuery.data;

  const refresh = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: supportAccess._def });
  }, [queryClient]);

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Trenova support access"),
        description: t(
          "Let a named Trenova staff member into this organization for a set time, and see everything they did",
        ),
        actions:
          canGrant && settings?.available ? (
            <Button size="sm" onClick={() => setGranting(true)}>
              <PlusIcon className="size-3.5" />
              {t("Grant access")}
            </Button>
          ) : undefined,
      }}
    >
      <div className="flex flex-col gap-4">
        {canSendLogs ? <SendLogsSection /> : null}
        {settingsQuery.isLoading || settings === undefined ? (
          <SupportGrantsSkeleton />
        ) : !settings.available ? (
          <Alert variant="info" size="sm">
            <AlertDescription>
              {t(
                "Trenova support access is not available on this installation, so nobody at Trenova can enter it.",
              )}
            </AlertDescription>
          </Alert>
        ) : (
          <SupportGrantList canRevoke={canRevoke} onRevoke={setRevoking} />
        )}
      </div>

      {settings?.available ? (
        <GrantSupportDialog
          open={granting}
          settings={settings}
          onClose={() => setGranting(false)}
          onGranted={refresh}
        />
      ) : null}
      <RevokeSupportDialog grant={revoking} onClose={() => setRevoking(null)} onRevoked={refresh} />
    </PageLayout>
  );
}
