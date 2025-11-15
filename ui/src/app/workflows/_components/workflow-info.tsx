import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { type WorkflowSchema } from "@/lib/schemas/workflow-schema";
import { format } from "date-fns";

const workflowStatusConfig = {
  draft: { label: "Draft", variant: "default" as const },
  active: { label: "Active", variant: "success" as const },
  inactive: { label: "Inactive", variant: "warning" as const },
  archived: { label: "Archived", variant: "destructive" as const },
};

const triggerTypeLabels = {
  manual: "Manual",
  scheduled: "Scheduled (Cron)",
  shipment_status: "Shipment Status Change",
  document_uploaded: "Document Upload",
  entity_created: "Entity Created",
  entity_updated: "Entity Updated",
  webhook: "Webhook",
};

export function WorkflowInfo({ workflow }: { workflow: WorkflowSchema }) {
  const statusConfig = workflowStatusConfig[workflow.status];

  return (
    <div className="space-y-6">
      {/* Basic Information */}
      <Card>
        <CardHeader>
          <CardTitle>Basic Information</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-sm text-muted-foreground">Name</p>
              <p className="font-medium">{workflow.name}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Status</p>
              <Badge variant={statusConfig.variant}>{statusConfig.label}</Badge>
            </div>
            {workflow.description && (
              <div className="col-span-2">
                <p className="text-sm text-muted-foreground">Description</p>
                <p className="font-medium">{workflow.description}</p>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Trigger Configuration */}
      <Card>
        <CardHeader>
          <CardTitle>Trigger Configuration</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-sm text-muted-foreground">Trigger Type</p>
              <p className="font-medium">
                {triggerTypeLabels[workflow.triggerType]}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Trigger Config</p>
              {workflow.triggerConfig &&
              Object.keys(workflow.triggerConfig).length > 0 ? (
                <pre className="mt-1 overflow-auto rounded-md bg-muted p-2 font-mono text-xs">
                  {JSON.stringify(workflow.triggerConfig, null, 2)}
                </pre>
              ) : (
                <p className="text-sm text-muted-foreground">
                  No configuration
                </p>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Execution Settings */}
      <Card>
        <CardHeader>
          <CardTitle>Execution Settings</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-sm text-muted-foreground">Timeout</p>
              <p className="font-medium">{workflow.timeoutSeconds}s</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Max Retries</p>
              <p className="font-medium">{workflow.maxRetries}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Retry Delay</p>
              <p className="font-medium">{workflow.retryDelaySeconds}s</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Enable Logging</p>
              <Badge variant={workflow.enableLogging ? "success" : "default"}>
                {workflow.enableLogging ? "Enabled" : "Disabled"}
              </Badge>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Version Information */}
      <Card>
        <CardHeader>
          <CardTitle>Version Information</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-sm text-muted-foreground">
                Current Version ID
              </p>
              <p className="font-mono text-sm font-medium">
                {workflow.currentVersionId || "No version"}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">
                Published Version ID
              </p>
              <p className="font-mono text-sm font-medium">
                {workflow.publishedVersionId || "Not published"}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Metadata */}
      <Card>
        <CardHeader>
          <CardTitle>Metadata</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-sm text-muted-foreground">Created At</p>
              <p className="text-sm font-medium">
                {format(new Date(workflow.createdAt), "PPpp")}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Updated At</p>
              <p className="text-sm font-medium">
                {format(new Date(workflow.updatedAt), "PPpp")}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
