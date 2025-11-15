import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { type WorkflowSchema } from "@/lib/schemas/workflow-schema";
import { api } from "@/services/api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

export function TriggerWorkflowDialog({
  workflow,
  open,
  onOpenChange,
}: {
  workflow: WorkflowSchema;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [inputData, setInputData] = useState("{}");
  const queryClient = useQueryClient();

  const triggerMutation = useMutation({
    mutationFn: async () => {
      let data = {};
      try {
        data = JSON.parse(inputData);
      } catch (e) {
        throw new Error("Invalid JSON input");
      }
      return api.workflowExecutions.trigger(workflow.id, data);
    },
    onSuccess: (execution) => {
      queryClient.invalidateQueries({ queryKey: ["workflow-executions"] });
      toast.success("Workflow triggered successfully", {
        description: `Execution ID: ${execution.id}`,
      });
      onOpenChange(false);
      setInputData("{}");
    },
    onError: (error: Error) => {
      toast.error("Failed to trigger workflow", {
        description: error.message,
      });
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>Trigger Workflow</DialogTitle>
          <DialogDescription>
            Manually trigger &quot;{workflow.name}&quot; with optional input
            data
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="input-data">Input Data (JSON)</Label>
            <Textarea
              id="input-data"
              value={inputData}
              onChange={(e) => setInputData(e.target.value)}
              placeholder='{"key": "value"}'
              className="font-mono text-sm"
              rows={8}
            />
            <p className="text-xs text-muted-foreground">
              Enter any input data required by the workflow in JSON format
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={triggerMutation.isPending}
          >
            Cancel
          </Button>
          <Button
            onClick={() => triggerMutation.mutate()}
            disabled={triggerMutation.isPending}
          >
            {triggerMutation.isPending && (
              <Loader2 className="mr-2 size-4 animate-spin" />
            )}
            Trigger Workflow
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
