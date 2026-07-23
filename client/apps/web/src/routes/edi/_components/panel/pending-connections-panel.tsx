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
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { EDIConnection } from "@trenova/shared/types/edi";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, XIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { formatUnix } from "../edi-display-utils";
import { invalidateEDIConnections } from "./edi-panel-invalidation";
import { EDIReasonDialog } from "./edi-reason-dialog";

export function PendingConnectionsPanel() {
  const queryClient = useQueryClient();
  const currentOrganizationId = useAuthStore((state) => state.user?.currentOrganizationId) ?? "";
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Update),
  );
  const [rejecting, setRejecting] = useState<EDIConnection | null>(null);
  const { data, isLoading } = useQuery(queries.edi.connections("?limit=25"));
  const pending = (data?.results ?? []).filter(
    (connection) =>
      connection.status === "PendingAcceptance" &&
      connection.targetOrganizationId === currentOrganizationId,
  );
  const acceptMutation = useApiMutation({
    mutationFn: (connectionId: string) => apiService.ediService.acceptConnection(connectionId),
    onSuccess: async () => {
      toast.success("EDI connection accepted");
      await invalidateEDIConnections(queryClient);
    },
    onError: () => toast.error("Failed to accept EDI connection"),
  });
  const rejectMutation = useApiMutation({
    mutationFn: ({ connection, reason }: { connection: EDIConnection; reason: string }) =>
      apiService.ediService.rejectConnection(connection.id, { reason }),
    onSuccess: async () => {
      toast.success("EDI connection rejected");
      setRejecting(null);
      await invalidateEDIConnections(queryClient);
    },
    onError: () => toast.error("Failed to reject EDI connection"),
  });

  if (!isLoading && pending.length === 0) {
    return null;
  }

  return (
    <div className="rounded-md border bg-background">
      <div className="flex items-center justify-between gap-2 border-b px-3 py-2">
        <div>
          <div className="text-sm font-medium">Pending EDI connection requests</div>
          <div className="text-xs text-muted-foreground">
            Accepting creates reciprocal internal partners and communication profiles.
          </div>
        </div>
        <Badge variant="outline">{pending.length}</Badge>
      </div>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Requester</TableHead>
            <TableHead>Target</TableHead>
            <TableHead>Method</TableHead>
            <TableHead>Requested</TableHead>
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
                      <XIcon data-icon="inline-start" />
                      Reject
                    </Button>
                    <Button
                      size="sm"
                      isLoading={acceptMutation.isPending}
                      onClick={() => acceptMutation.mutate(connection.id)}
                    >
                      <CheckIcon data-icon="inline-start" />
                      Accept
                    </Button>
                  </div>
                )}
              </TableCell>
            </TableRow>
          ))}
          {isLoading && (
            <TableRow>
              <TableCell colSpan={5} className="h-16 text-center text-muted-foreground">
                Loading connection requests.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
      <EDIReasonDialog
        open={!!rejecting}
        onOpenChange={(open) => !open && setRejecting(null)}
        title="Reject EDI Connection"
        description={
          rejecting
            ? `Reject the connection request from ${rejecting.sourceOrganization?.name ?? rejecting.sourceOrganizationId}.`
            : undefined
        }
        placeholder="Reason shared with the requesting organization"
        confirmLabel="Reject Connection"
        isPending={rejectMutation.isPending}
        onConfirm={(reason) => rejecting && rejectMutation.mutate({ connection: rejecting, reason })}
      />
    </div>
  );
}
