import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import type { EDIConnection } from "@/types/edi";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, XIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { formatUnix } from "../edi-display-utils";
import { invalidateEDIConnections } from "./edi-panel-invalidation";

export function PendingConnectionsPanel() {
  const queryClient = useQueryClient();
  const currentOrganizationId = useAuthStore((state) => state.user?.currentOrganizationId) ?? "";
  const [rejecting, setRejecting] = useState<EDIConnection | null>(null);
  const [reason, setReason] = useState("");
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
    mutationFn: (connection: EDIConnection) =>
      apiService.ediService.rejectConnection(connection.id, { reason }),
    onSuccess: async () => {
      toast.success("EDI connection rejected");
      setRejecting(null);
      setReason("");
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
      <Sheet open={!!rejecting} onOpenChange={(open) => !open && setRejecting(null)}>
        <SheetContent>
          <SheetHeader>
            <SheetTitle>Reject EDI Connection</SheetTitle>
            <SheetDescription>
              {rejecting?.sourceOrganization?.name ?? rejecting?.id}
            </SheetDescription>
          </SheetHeader>
          <div className="px-4">
            <Input
              placeholder="Rejection reason"
              value={reason}
              onChange={(event) => setReason(event.target.value)}
            />
          </div>
          <SheetFooter>
            <Button variant="outline" onClick={() => setRejecting(null)}>
              Cancel
            </Button>
            <Button
              disabled={!reason.trim() || !rejecting}
              isLoading={rejectMutation.isPending}
              onClick={() => rejecting && rejectMutation.mutate(rejecting)}
            >
              Reject
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </div>
  );
}
