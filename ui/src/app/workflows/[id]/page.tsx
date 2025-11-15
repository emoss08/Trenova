import { MetaTags } from "@/components/meta-tags";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { useParams } from "react-router";
import { WorkflowBuilderContent } from "../_components/workflow-builder-content";
import { WorkflowBuilderHeader } from "../_components/workflow-builder-header";

export default function WorkflowDetail() {
  const { id } = useParams<{ id: string }>();

  const { data: workflow, isLoading } = useQuery(
    queries.workflow.getById(id!, !!id),
  );

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <Loader2 className="size-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!workflow) {
    return (
      <div className="flex h-screen flex-col items-center justify-center gap-4">
        <h2 className="text-2xl font-semibold">Workflow Not Found</h2>
      </div>
    );
  }

  return (
    <>
      <MetaTags
        title={`Workflow: ${workflow.name}`}
        description={workflow.description || "Workflow details"}
      />
      <WorkflowBuilderOuter>
        <WorkflowBuilderHeader />
        <WorkflowBuilderContent workflow={workflow} />
      </WorkflowBuilderOuter>
    </>
  );
}

function WorkflowBuilderOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex h-full flex-col gap-2">{children}</div>;
}
