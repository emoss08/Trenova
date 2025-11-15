/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
import { api } from "@/services/api";
import {
  type WorkflowSchema,
  type WorkflowStatusType,
  type TriggerTypeType,
} from "@/lib/schemas/workflow-schema";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";
import {
  Archive,
  CheckCircle2,
  MoreHorizontal,
  Pause,
  Play,
  Workflow as WorkflowIcon,
  PlayCircle,
} from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router";
import { TriggerWorkflowDialog } from "./trigger-workflow-dialog";
import { toast } from "sonner";

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
    mutationFn: () => api.workflows.activate(workflow.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["workflows"] });
      toast.success("Workflow activated successfully");
    },
    onError: () => {
      toast.error("Failed to activate workflow");
    },
  });

  const deactivateMutation = useMutation({
    mutationFn: () => api.workflows.deactivate(workflow.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["workflows"] });
      toast.success("Workflow deactivated successfully");
    },
    onError: () => {
      toast.error("Failed to deactivate workflow");
    },
  });

  const archiveMutation = useMutation({
    mutationFn: () => api.workflows.archive(workflow.id),
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
          >
            <WorkflowIcon className="mr-2 size-4" />
            View Builder
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setTriggerDialogOpen(true)}>
            <PlayCircle className="mr-2 size-4" />
            Trigger Workflow
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          {workflow.status !== "active" && workflow.status !== "archived" && (
            <DropdownMenuItem onClick={() => activateMutation.mutate()}>
              <Play className="mr-2 size-4" />
              Activate
            </DropdownMenuItem>
          )}
          {workflow.status === "active" && (
            <DropdownMenuItem onClick={() => deactivateMutation.mutate()}>
              <Pause className="mr-2 size-4" />
              Deactivate
            </DropdownMenuItem>
          )}
          {workflow.status !== "archived" && (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => archiveMutation.mutate()}>
                <Archive className="mr-2 size-4" />
                Archive
              </DropdownMenuItem>
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
        return <Badge variant={config.variant}>{config.label}</Badge>;
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
    commonColumns.updatedAt,
    columnHelper.display({
      id: "actions",
      header: "Actions",
      cell: ({ row }) => <WorkflowActions workflow={row.original} />,
    }),
  ];
}
