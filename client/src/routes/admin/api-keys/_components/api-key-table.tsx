"use no memo";

import { DataTable } from "@/components/data-table/data-table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import type { ApiKey } from "@/types/api-key";
import type { RowAction } from "@/types/data-table";
import { Resource } from "@/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import type { Row } from "@tanstack/react-table";
import { ShieldAlertIcon, ShieldOffIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./api-key-columns";
import { APIKeyPanel } from "./api-key-panel";

export default function APIKeyTable() {
  const queryClient = useQueryClient();
  const [selectedKey, setSelectedKey] = useState<ApiKey | null>(null);
  const [revokeDialogOpen, setRevokeDialogOpen] = useState(false);

  const revokeMutation = useApiMutation({
    mutationFn: async (id: ApiKey["id"]) => apiService.apiKeyService.revoke(id),
    onSuccess: async () => {
      toast.success("API key revoked");
      setRevokeDialogOpen(false);
      setSelectedKey(null);
      await queryClient.invalidateQueries({ queryKey: ["api-key-list"] });
    },
    resourceName: "API Key",
  });

  const handleRevoke = useCallback((row: Row<ApiKey>) => {
    setSelectedKey(row.original);
    setRevokeDialogOpen(true);
  }, []);

  const columns = useMemo(() => getColumns(), []);

  const contextMenuActions = useMemo<RowAction<ApiKey>[]>(
    () => [
      {
        id: "revoke",
        label: "Revoke",
        icon: ShieldOffIcon,
        variant: "destructive",
        onClick: handleRevoke,
        hidden: (row) => row.original.status === "revoked",
      },
    ],
    [handleRevoke],
  );

  return (
    <>
      <DataTable<ApiKey>
        name="API Key"
        link="/api-keys/"
        queryKey="api-key-list"
        exportModelName="api-key"
        resource={Resource.Integration}
        columns={columns}
        TablePanel={APIKeyPanel}
        contextMenuActions={contextMenuActions}
      />

      <AlertDialog open={revokeDialogOpen} onOpenChange={setRevokeDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <ShieldAlertIcon />
            </AlertDialogMedia>
            <AlertDialogTitle>Revoke API Key</AlertDialogTitle>
            <AlertDialogDescription>
              Revoke {selectedKey?.name ?? "this key"} now. Any integration using this bearer token
              will begin failing authentication immediately.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => {
                if (selectedKey?.id) {
                  revokeMutation.mutate(selectedKey.id);
                }
              }}
              disabled={revokeMutation.isPending}
            >
              Revoke Key
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
