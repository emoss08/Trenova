"use no memo";

import { useT } from "@trenova/shared/i18n/use-t";
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
} from "@trenova/shared/components/ui/alert-dialog";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiKeyTableGraphQLConfig, type ApiKeyRow } from "@/lib/graphql/api-key-table";
import { apiService } from "@/services/api";
import type { RowAction, Row } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { ShieldAlertIcon, ShieldOffIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./api-key-columns";
import { APIKeyPanel } from "./api-key-panel";

export default function APIKeyTable() {
  const t = useT();

  const queryClient = useQueryClient();
  const [selectedKey, setSelectedKey] = useState<ApiKeyRow | null>(null);
  const [revokeDialogOpen, setRevokeDialogOpen] = useState(false);

  const revokeMutation = useApiMutation({
    mutationFn: async (id: ApiKeyRow["id"]) => apiService.apiKeyService.revoke(id),
    onSuccess: async () => {
      toast.success(t("API key revoked"));
      setRevokeDialogOpen(false);
      setSelectedKey(null);
      await queryClient.invalidateQueries({ queryKey: ["api-key-list"] });
    },
    resourceName: "API Key",
  });

  const handleRevoke = useCallback((row: Row<ApiKeyRow>) => {
    setSelectedKey(row.original);
    setRevokeDialogOpen(true);
  }, []);

  const columns = useMemo(() => getColumns(), []);

  const contextMenuActions = useMemo<RowAction<ApiKeyRow>[]>(
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
      <DataTable<ApiKeyRow>
        name="API Key"
        queryKey="api-key-list"
        graphql={apiKeyTableGraphQLConfig}
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
            <AlertDialogTitle>{t("Revoke API Key")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("Revoke {0} now. Any integration using this bearer token will begin failing authentication immediately.", selectedKey?.name ?? "this key")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => {
                if (selectedKey?.id) {
                  revokeMutation.mutate(selectedKey.id);
                }
              }}
              disabled={revokeMutation.isPending}
            >
              {t("Revoke Key")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
