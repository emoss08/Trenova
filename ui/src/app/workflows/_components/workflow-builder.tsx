import { ReactFlowProvider } from "@xyflow/react";
import React, { lazy, Suspense } from "react";
import {
  ReactFlowSkeleton,
  WorkflowOptionsSkeleton,
} from "./workflow-skeletons";

const ReactFlowContent = lazy(() => import("./workflow-flow-content"));
const WorkflowOptions = lazy(() => import("./workflow-options"));

export function WorkflowBuilder({
  workflowId,
  versionId,
}: {
  workflowId?: string;
  versionId?: string;
}) {
  return (
    <ReactFlowProvider>
      <WorkflowBuilderOuter>
        <Suspense fallback={<ReactFlowSkeleton />}>
          <ReactFlowContent workflowId={workflowId!} versionId={versionId!} />
        </Suspense>
        <Suspense fallback={<WorkflowOptionsSkeleton />}>
          <WorkflowOptions />
        </Suspense>
      </WorkflowBuilderOuter>
    </ReactFlowProvider>
  );
}

function WorkflowBuilderOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex h-full flex-row gap-4">{children}</div>;
}
