/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { AlertCircle } from "lucide-react";

/**
 * Workflow Builder Component
 *
 * This is a placeholder for the workflow builder.
 * Once @xyflow/react is installed, this will be replaced with
 * a full-featured visual workflow builder with drag-and-drop nodes and edges.
 *
 * Features to implement:
 * - Visual canvas with React Flow
 * - Node palette (trigger, action, condition, loop, delay, end)
 * - Drag and drop nodes
 * - Connect nodes with edges
 * - Node configuration panels
 * - Save/load workflow definitions
 * - Version management
 * - Publish/unpublish workflows
 */
export function WorkflowBuilder({
  workflowId,
  versionId,
}: {
  workflowId?: string;
  versionId?: string;
}) {
  return (
    <div className="flex h-full flex-col">
      <div className="border-b p-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="font-semibold text-lg">Workflow Builder</h2>
            <p className="text-muted-foreground text-sm">
              Visual workflow automation designer
            </p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" size="sm">
              Save Draft
            </Button>
            <Button variant="outline" size="sm">
              Publish
            </Button>
          </div>
        </div>
      </div>

      <div className="flex flex-1 items-center justify-center p-8">
        <Card className="max-w-lg">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <AlertCircle className="size-5" />
              Workflow Builder Coming Soon
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-muted-foreground text-sm">
              The visual workflow builder requires the @xyflow/react package to
              be installed.
            </p>
            <div className="rounded-lg bg-muted p-4">
              <p className="font-medium text-sm">Installation Required:</p>
              <code className="mt-2 block rounded bg-background p-2 font-mono text-xs">
                npm install @xyflow/react
              </code>
            </div>
            <div className="space-y-2">
              <p className="font-medium text-sm">Planned Features:</p>
              <ul className="ml-4 list-disc space-y-1 text-muted-foreground text-sm">
                <li>Drag-and-drop visual workflow designer</li>
                <li>Pre-built node types (trigger, action, condition, etc.)</li>
                <li>Node configuration panels</li>
                <li>Real-time validation</li>
                <li>Version control and publishing</li>
                <li>Template library</li>
              </ul>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
