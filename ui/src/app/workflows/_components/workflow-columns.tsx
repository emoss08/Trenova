import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  type TriggerTypeType,
  type WorkflowSchema,
  type WorkflowStatusType,
} from "@/lib/schemas/workflow-schema";
import { api } from "@/services/api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";
import {
  Archive,
  MoreHorizontal,
  Pause,
  Play,
  PlayCircle,
  Workflow as WorkflowIcon,
} from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import { TriggerWorkflowDialog } from "./trigger-workflow-dialog";

const workflowStatusConfig: Record<
  WorkflowStatusType,
  { label: string; variant: "default" | "success" | "warning" | "destructive" }
> = {
  draft: { label: "Draft", variant: "default" },
  active: { label: "Active", variant: "success" },
  inactive: { label: "Inactive", variant: "warning" },
  archived: { label: "Archived", variant: "destructive" },
};

const triggerTypeLabels: Record<TriggerTypeType, string> = {
  manual: "Manual",
  scheduled: "Scheduled",
  shipment_status: "Shipment Status",
  document_uploaded: "Document Upload",
  entity_created: "Entity Created",
  entity_updated: "Entity Updated",
  webhook: "Webhook",
};

function WorkflowActions({ workflow }: { workflow: WorkflowSchema }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [triggerDialogOpen, setTriggerDialogOpen] = useState(false);

  const activateMutation = useMutation({
    mutationFn: () => api.workflows.activate(workflow.id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["workflows"] });
      toast.success("Workflow activated successfully");
    },
    onError: () => {
      toast.error("Failed to activate workflow");
    },
  });

  const deactivateMutation = useMutation({
    mutationFn: () => api.workflows.deactivate(workflow.id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["workflows"] });
      toast.success("Workflow deactivated successfully");
    },
    onError: () => {
      toast.error("Failed to deactivate workflow");
    },
  });

  const archiveMutation = useMutation({
    mutationFn: () => api.workflows.archive(workflow.id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["workflows"] });
      toast.success("Workflow archived successfully");
    },
    onError: () => {
      toast.error("Failed to archive workflow");
    },
  });

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" className="size-8 p-0">
            <span className="sr-only">Open menu</span>
            <MoreHorizontal className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel>Actions</DropdownMenuLabel>
          <DropdownMenuItem
            onClick={() => navigate(`/organization/workflows/${workflow.id}`)}
            startContent={<WorkflowIcon className="mr-2 size-4" />}
            title="View Builder"
          />
          <DropdownMenuItem
            startContent={<PlayCircle className="mr-2 size-4" />}
            title="Trigger Workflow"
            onClick={() => setTriggerDialogOpen(true)}
          />
          <DropdownMenuSeparator />
          {workflow.status !== "active" && workflow.status !== "archived" && (
            <DropdownMenuItem
              title="Activate Workflow"
              onClick={() => activateMutation.mutate()}
              startContent={<Play className="mr-2 size-4" />}
            />
          )}
          {workflow.status === "active" && (
            <DropdownMenuItem
              title="Deactivate Workflow"
              onClick={() => deactivateMutation.mutate()}
              startContent={<Pause className="mr-2 size-4" />}
            />
          )}
          {workflow.status !== "archived" && (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                title="Archive Workflow"
                onClick={() => archiveMutation.mutate()}
                startContent={<Archive className="mr-2 size-4" />}
              />
            </>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
      <TriggerWorkflowDialog
        workflow={workflow}
        open={triggerDialogOpen}
        onOpenChange={setTriggerDialogOpen}
      />
    </>
  );
}

export function getColumns(): ColumnDef<WorkflowSchema>[] {
  const columnHelper = createColumnHelper<WorkflowSchema>();
  const commonColumns = createCommonColumns<WorkflowSchema>();

  return [
    columnHelper.display({
      id: "status",
      header: "Status",
      cell: ({ row }) => {
        const status = row.original.status;
        const config = workflowStatusConfig[status];
        return <Badge>{config.label}</Badge>;
      },
    }),
    columnHelper.display({
      id: "name",
      header: "Name",
      cell: ({ row }) => <p className="font-medium">{row.original.name}</p>,
    }),
    commonColumns.description,
    columnHelper.display({
      id: "triggerType",
      header: "Trigger Type",
      cell: ({ row }) => (
        <Badge variant="outline">
          {triggerTypeLabels[row.original.triggerType]}
        </Badge>
      ),
    }),
    columnHelper.display({
      id: "versioning",
      header: "Version",
      cell: ({ row }) => {
        const hasPublished = !!row.original.publishedVersionId;
        return (
          <div className="flex flex-col gap-1">
            <span className="text-sm">
              {hasPublished ? "Published" : "Not Published"}
            </span>
          </div>
        );
      },
    }),
    commonColumns.createdAt,
    columnHelper.display({
      id: "actions",
      header: "Actions",
      cell: ({ row }) => <WorkflowActions workflow={row.original} />,
    }),
  ];
}
