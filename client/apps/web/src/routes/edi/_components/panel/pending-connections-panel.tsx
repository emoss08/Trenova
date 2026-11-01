import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { edi } from "@/lib/queries/edi";
import { apiService } from "@/services/api";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PendingEdiConnectionsQuery } from "@trenova/graphql/generated/graphql";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { useState } from "react";
import { toast } from "@trenova/shared/components/ui/toast";
import { formatUnix } from "../edi-display-utils";
import { invalidateEDIConnections } from "./edi-panel-invalidation";
import { EDIReasonDialog } from "./edi-reason-dialog";

type PendingEdiConnection = PendingEdiConnectionsQuery["pendingEdiConnections"][number];

export function PendingConnectionsPanel() {
  const t = useT();

  const queryClient = useQueryClient();
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Update),
  );
  const [rejecting, setRejecting] = useState<PendingEdiConnection | null>(null);
  const { data, isLoading } = useQuery(edi.pendingConnections());
  const pending = data ?? [];
  const acceptMutation = useApiMutation({
    mutationFn: (connectionId: string) => apiService.ediService.acceptConnection(connectionId),
    onSuccess: async () => {
      await invalidateEDIConnections(queryClient);
    },
    onError: () => toast.error(t("Failed to accept EDI connection")),
  });
  const rejectMutation = useApiMutation({
    mutationFn: ({ connection, reason }: { connection: PendingEdiConnection; reason: string }) =>
      apiService.ediService.rejectConnection(connection.id, { reason }),
    onSuccess: async () => {
      setRejecting(null);
      await invalidateEDIConnections(queryClient);
    },
    onError: () => toast.error(t("Failed to reject EDI connection")),
  });

  if (!isLoading && pending.length === 0) {
    return null;
  }

  return (
    <div className="bg-background rounded-md border">
      <div className="flex items-center justify-between gap-2 border-b px-3 py-2">
        <div>
          <div className="text-sm font-medium">{t("Pending EDI connection requests")}</div>
          <div className="text-muted-foreground text-xs">
            {t("Accepting creates reciprocal internal partners and communication profiles.")}
          </div>
        </div>
        <Badge variant="neutral" appearance="outline">
          {pending.length}
        </Badge>
      </div>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t("Requester")}</TableHead>
            <TableHead>{t("Target")}</TableHead>
            <TableHead>{t("Method")}</TableHead>
            <TableHead>{t("Requested")}</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {pending.map((connection) => (
            <TableRow key={connection.id}>
              <TableCell>
                {connection.sourceOrganization?.name ?? connection.sourceOrganizationId}
              </TableCell>
              <TableCell>
                {connection.targetOrganization?.name ?? connection.targetOrganizationId}
              </TableCell>
              <TableCell>{connection.method}</TableCell>
              <TableCell>{formatUnix(connection.requestedAt)}</TableCell>
              <TableCell className="text-right">
                {canUpdate && (
                  <div className="flex justify-end gap-2">
                    <Button variant="outline" size="sm" onClick={() => setRejecting(connection)}>
                      <XCloseIcon data-icon="inline-start" />
                      {t("Reject")}
                    </Button>
                    <Button
                      size="sm"
                      isLoading={acceptMutation.isPending}
                      onClick={() => acceptMutation.mutate(connection.id)}
                    >
                      <CheckIcon data-icon="inline-start" />
                      {t("Accept")}
                    </Button>
                  </div>
                )}
              </TableCell>
            </TableRow>
          ))}
          {isLoading && (
            <TableRow>
              <TableCell colSpan={5} className="text-muted-foreground h-16 text-center">
                {t("Loading connection requests.")}
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
      <EDIReasonDialog
        open={!!rejecting}
        onOpenChange={(open) => !open && setRejecting(null)}
        title={t("Reject EDI connection")}
        description={
          rejecting
            ? t(
                "Reject the connection request from {0}.",
                rejecting.sourceOrganization?.name ?? rejecting.sourceOrganizationId,
              )
            : undefined
        }
        placeholder={t("Reason shared with the requesting organization")}
        confirmLabel={t("Reject connection")}
        isPending={rejectMutation.isPending}
        onConfirm={(reason) =>
          rejecting && rejectMutation.mutate({ connection: rejecting, reason })
        }
      />
    </div>
  );
}
