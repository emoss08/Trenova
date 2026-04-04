import { DataTable } from "@/components/data-table/data-table";
import { fetchOptions } from "@/components/fields/autocomplete/autocomplete-content";
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
import { DocumentPacketRuleService } from "@/services/document-packet-rule";
import type { RowAction } from "@/types/data-table";
import type { DocumentPacketRule } from "@/types/document-packet-rule";
import type { DocumentType } from "@/types/document-type";
import { Resource } from "@/types/permission";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { Row } from "@tanstack/react-table";
import { Loader2Icon, TrashIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./document-packet-rule-columns";
import { DocumentPacketRulePanel } from "./document-packet-rule-panel";

const service = new DocumentPacketRuleService();

export default function DocumentPacketRuleTable() {
  const queryClient = useQueryClient();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedRule, setSelectedRule] = useState<DocumentPacketRule | null>(null);

  const { data: documentTypesData } = useQuery({
    queryKey: ["document-types-select-options"],
    queryFn: () => fetchOptions<DocumentType>("/document-types/select-options/", "", 1, 100),
    staleTime: 5 * 60 * 1000,
  });

  const documentTypeMap = useMemo(() => {
    const map = new Map<string, DocumentType>();
    for (const dt of documentTypesData?.results ?? []) {
      if (dt.id) map.set(dt.id, dt);
    }
    return map;
  }, [documentTypesData]);

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await service.delete(id);
    },
    onSuccess: () => {
      toast.success("Document packet rule deleted");
      void queryClient.invalidateQueries({
        queryKey: ["document-packet-rule-list"],
      });
      setDeleteDialogOpen(false);
      setSelectedRule(null);
    },
    onError: (error) => {
      toast.error("Failed to delete document packet rule", {
        description: error instanceof Error ? error.message : "An unexpected error occurred",
      });
    },
  });

  const handleDelete = useCallback((row: Row<DocumentPacketRule>) => {
    setSelectedRule(row.original);
    setDeleteDialogOpen(true);
  }, []);

  const columns = useMemo(() => getColumns(documentTypeMap), [documentTypeMap]);

  const contextMenuActions = useMemo<RowAction<DocumentPacketRule>[]>(
    () => [
      {
        id: "delete",
        label: "Delete",
        icon: TrashIcon,
        variant: "destructive",
        onClick: handleDelete,
      },
    ],
    [handleDelete],
  );

  return (
    <>
      <DataTable<DocumentPacketRule>
        name="Document Packet Rule"
        link="/document-packet-rules/"
        queryKey="document-packet-rule-list"
        exportModelName="document-packet-rule"
        resource={Resource.DocumentType}
        columns={columns}
        contextMenuActions={contextMenuActions}
        TablePanel={DocumentPacketRulePanel}
      />
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <TrashIcon />
            </AlertDialogMedia>
            <AlertDialogTitle>Delete Document Packet Rule</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this packet rule? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => {
                if (selectedRule?.id) {
                  deleteMutation.mutate(selectedRule.id);
                }
              }}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending && <Loader2Icon className="mr-2 size-4 animate-spin" />}
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
