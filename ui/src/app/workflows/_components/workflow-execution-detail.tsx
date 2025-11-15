import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { queries } from "@/lib/queries";
import {
  type ExecutionStatusType,
  type WorkflowExecutionSchema,
} from "@/lib/schemas/workflow-schema";
import { useQuery } from "@tanstack/react-query";
import { format } from "date-fns";
import {
  AlertCircle,
  CheckCircle2,
  Circle,
  Clock,
  Loader2,
  XCircle,
} from "lucide-react";

const executionStatusConfig: Record<
  ExecutionStatusType,
  { label: string; icon: React.ComponentType<{ className?: string }> }
> = {
  pending: { label: "Pending", icon: Circle },
  running: { label: "Running", icon: Loader2 },
  completed: { label: "Completed", icon: CheckCircle2 },
  failed: { label: "Failed", icon: XCircle },
  cancelled: { label: "Cancelled", icon: XCircle },
  timeout: { label: "Timeout", icon: AlertCircle },
};

export function WorkflowExecutionDetail({
  execution,
}: {
  execution: WorkflowExecutionSchema;
}) {
  const { data: steps, isLoading } = useQuery(
    queries.workflowExecution.getSteps(execution.id),
  );

  const StatusIcon = executionStatusConfig[execution.status].icon;

  return (
    <div className="space-y-6">
      {/* Execution Overview */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <StatusIcon className="size-5" />
            Execution Details
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-sm text-muted-foreground">Status</p>
              <Badge>{executionStatusConfig[execution.status].label}</Badge>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Triggered By</p>
              <p className="text-sm font-medium">
                {execution.triggeredBy || "System"}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Started At</p>
              <p className="text-sm font-medium">
                {execution.startedAt
                  ? format(new Date(execution.startedAt), "PPpp")
                  : "-"}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Completed At</p>
              <p className="text-sm font-medium">
                {execution.completedAt
                  ? format(new Date(execution.completedAt), "PPpp")
                  : "-"}
              </p>
            </div>
          </div>

          {execution.error && (
            <div className="rounded-lg border-destructive bg-destructive/10 p-4">
              <p className="text-sm font-medium text-destructive">Error</p>
              <p className="mt-1 text-sm text-destructive">{execution.error}</p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Execution Steps */}
      <Card>
        <CardHeader>
          <CardTitle>Execution Steps</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="size-6 animate-spin text-muted-foreground" />
            </div>
          ) : steps && steps.length > 0 ? (
            <div className="space-y-4">
              {steps.map((step, index) => {
                const StepIcon = executionStatusConfig[step.status].icon;

                return (
                  <div key={step.id}>
                    <div className="flex gap-4">
                      <div className="flex flex-col items-center">
                        <div
                          className={`flex size-8 items-center justify-center rounded-full ${
                            step.status === "completed"
                              ? "bg-green-100 text-green-600"
                              : step.status === "failed"
                                ? "bg-red-100 text-red-600"
                                : step.status === "running"
                                  ? "bg-yellow-100 text-yellow-600"
                                  : "bg-gray-100 text-gray-600"
                          }`}
                        >
                          <StepIcon className="size-4" />
                        </div>
                        {index < steps.length - 1 && (
                          <div className="h-full w-px bg-border" />
                        )}
                      </div>

                      <div className="flex-1 pb-6">
                        <div className="flex items-start justify-between">
                          <div>
                            <p className="font-medium">{step.nodeId}</p>
                            <p className="text-sm text-muted-foreground">
                              Step {step.stepNumber}
                            </p>
                          </div>
                          <div className="flex items-center gap-2 text-xs text-muted-foreground">
                            <Clock className="size-3" />
                            {step.startedAt
                              ? format(new Date(step.startedAt), "HH:mm:ss")
                              : "-"}
                          </div>
                        </div>

                        {step.error && (
                          <div className="mt-2 rounded-md bg-destructive/10 p-3">
                            <p className="text-xs font-medium text-destructive">
                              Error
                            </p>
                            <p className="mt-1 text-xs text-destructive">
                              {step.error}
                            </p>
                          </div>
                        )}

                        {step.output && (
                          <div className="mt-2 rounded-md bg-muted p-3">
                            <p className="text-xs font-medium">Output</p>
                            <pre className="mt-1 overflow-auto font-mono text-xs">
                              {JSON.stringify(step.output, null, 2)}
                            </pre>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="py-8 text-center text-muted-foreground">
              No execution steps found
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
