import { Info } from "lucide-react";

export function CustomPathNotice() {
  return (
    <div className="p-2">
      <div className="flex items-start gap-2 rounded-md border border-border bg-sidebar p-2">
        <Info className="mt-0.5 size-3.5 shrink-0" />
        <div className="text-xs">
          <p className="font-medium">Custom Paths</p>
          <p className="mt-1 text-muted-foreground">
            You can access nested fields using dot notation:{" "}
            <code className="rounded border border-border bg-background px-1 py-0.5 font-mono">
              {"{"}
              {"{"}trigger.customer.email{"}}"}
            </code>
          </p>
        </div>
      </div>
    </div>
  );
}

export function WorkflowVariableHeader() {
  return (
    <div className="space-y-1 p-2">
      <p className="text-sm font-semibold">Workflow Variables</p>
      <p className="text-xs text-muted-foreground">
        Click a variable to insert it at the cursor position. Variables are
        resolved when the workflow executes.
      </p>
    </div>
  );
}
