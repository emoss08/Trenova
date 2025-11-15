/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { MetaTags } from "@/components/meta-tags";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, Loader2 } from "lucide-react";
import { useNavigate, useParams } from "react-router";
import { WorkflowBuilder } from "../_components/workflow-builder";
import { WorkflowExecutionDetail } from "../_components/workflow-execution-detail";
import { WorkflowInfo } from "../_components/workflow-info";

export default function WorkflowDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

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
        <Button onClick={() => navigate("/organization/workflows")}>
          <ArrowLeft className="mr-2 size-4" />
          Back to Workflows
        </Button>
      </div>
    );
  }

  return (
    <>
      <MetaTags
        title={`Workflow: ${workflow.name}`}
        description={workflow.description || "Workflow details"}
      />
      <div className="flex h-full flex-col">
        <div className="border-b p-4">
          <div className="flex items-center gap-4">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => navigate("/organization/workflows")}
            >
              <ArrowLeft className="mr-2 size-4" />
              Back
            </Button>
            <div>
              <h1 className="text-2xl font-bold">{workflow.name}</h1>
              {workflow.description && (
                <p className="text-sm text-muted-foreground">
                  {workflow.description}
                </p>
              )}
            </div>
          </div>
        </div>

        <div className="flex-1 overflow-auto">
          <Tabs defaultValue="builder" className="h-full">
            <div className="border-b px-4">
              <TabsList>
                <TabsTrigger value="builder">Builder</TabsTrigger>
                <TabsTrigger value="info">Information</TabsTrigger>
                <TabsTrigger value="executions">Executions</TabsTrigger>
              </TabsList>
            </div>

            <TabsContent value="builder" className="h-full p-0">
              <WorkflowBuilder
                workflowId={workflow.id}
                versionId={workflow.currentVersionId || undefined}
              />
            </TabsContent>

            <TabsContent value="info" className="p-6">
              <WorkflowInfo workflow={workflow} />
            </TabsContent>

            <TabsContent value="executions" className="p-6">
              <ExecutionsList workflowId={workflow.id!} />
            </TabsContent>
          </Tabs>
        </div>
      </div>
    </>
  );
}

function ExecutionsList({ workflowId }: { workflowId: string }) {
  const { data: executions, isLoading } = useQuery(
    queries.workflowExecution.list({ workflowId }),
  );

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="size-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!executions?.items || executions.items.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>No Executions</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            This workflow hasn&apos;t been executed yet.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {executions.items.map((execution) => (
        <WorkflowExecutionDetail key={execution.id} execution={execution} />
      ))}
    </div>
  );
}
