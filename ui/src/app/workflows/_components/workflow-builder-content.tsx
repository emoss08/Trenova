import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { WorkflowSchema } from "@/lib/schemas/workflow-schema";
import { InfoIcon, WorkflowIcon } from "lucide-react";
import { useQueryState } from "nuqs";
import { WorkflowBuilder } from "./workflow-builder";
import { WorkflowInfo } from "./workflow-info";

export function WorkflowBuilderContent({
  workflow,
}: {
  workflow: WorkflowSchema;
}) {
  return (
    <WorkflowBuilderContentOuter>
      <WorkflowBuilderTabs workflow={workflow} />
    </WorkflowBuilderContentOuter>
  );
}

function WorkflowBuilderTabs({ workflow }: { workflow: WorkflowSchema }) {
  const [tab, setTab] = useQueryState("tab", {
    defaultValue: "builder",
    shallow: true, // So it doesn't trigger a full page reload
  });

  return (
    <Tabs value={tab} onValueChange={setTab}>
      <TabsList className="relative h-auto w-full justify-start gap-0.5 bg-transparent p-0 before:absolute before:inset-x-0 before:bottom-0 before:h-px before:bg-border">
        <TabsTrigger
          value="builder"
          className="overflow-hidden rounded-b-none border-x border-t bg-muted py-2 data-[state=active]:z-10 data-[state=active]:shadow-none"
        >
          <WorkflowIcon
            className="-ms-0.5 mb-0.5 opacity-60"
            size={16}
            aria-hidden="true"
          />
          Builder
        </TabsTrigger>
        <TabsTrigger
          value="info"
          className="overflow-hidden rounded-b-none border-x border-t bg-muted py-2 data-[state=active]:z-10 data-[state=active]:shadow-none"
        >
          <InfoIcon
            className="-ms-0.5 mb-0.5 opacity-60"
            size={16}
            aria-hidden="true"
          />
          Information
        </TabsTrigger>
      </TabsList>
      <TabsContent value="builder">
        <WorkflowBuilder
          workflowId={workflow.id}
          versionId={workflow.currentVersionId ?? undefined}
        />
      </TabsContent>
      <TabsContent value="info">
        <WorkflowInfo workflow={workflow} />
      </TabsContent>
    </Tabs>
  );
}

export function WorkflowBuilderContentOuter({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex-1 overflow-auto">{children}</div>;
}
