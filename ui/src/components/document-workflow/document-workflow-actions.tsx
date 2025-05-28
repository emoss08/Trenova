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
import { api } from "@/services/api";
import { Document } from "@/types/document";
import { faEllipsis } from "@fortawesome/pro-solid-svg-icons";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";

export function DocumentActions({ document }: { document: Document }) {
  const { mutateAsync: removeDocument, isPending } = useMutation({
    mutationFn: async () => {
      return await api.documents.delete(document.id);
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
    <div className="absolute top-2 right-2">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="icon">
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
    </div>
  );
}
