import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Icon } from "@/components/ui/icons";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { queries } from "@/lib/queries";
import { deleteDocument } from "@/services/document";
import { Document } from "@/types/document";
import { faEllipsis } from "@fortawesome/pro-solid-svg-icons";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";

export function DocumentActions({ document }: { document: Document }) {
  const { mutateAsync: removeDocument, isPending } = useMutation({
    mutationFn: async () => {
      await deleteDocument(document.id);
    },
    onSuccess: () => {
      toast.success("Document deleted successfully");
      broadcastQueryInvalidation({
        queryKey: [...queries.document.documentsByResourceID._def],
        options: {
          correlationId: `update-document-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });
    },
    onError: (error) => {
      toast.error(
        `Failed to delete document: ${error instanceof Error ? error.message : "Unknown error"}`,
      );
    },
  });

  const handleDelete = () => {
    removeDocument();
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" className="p-2">
          <Icon icon={faEllipsis} className="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem
          title="Delete"
          description="Delete the document"
          color="danger"
          onClick={handleDelete}
          disabled={isPending}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
