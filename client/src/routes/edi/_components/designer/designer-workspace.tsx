import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ArchiveIcon, Layers3Icon } from "lucide-react";
import { lazy, Suspense } from "react";
import { DesignerWorkspaceSkeleton } from "./components/designer-workspace-skeleton";
import { useEDIDesignerUrlState } from "./hooks/use-edi-designer-url-state";

const TemplateDesignerTab = lazy(() => import("./components/template-designer-tab"));
const DocumentPreviewArchiveTab = lazy(() => import("./archive/document-preview-archive-tab"));

function DesignerLoadingBlock() {
  return <DesignerWorkspaceSkeleton />;
}

export function DesignerWorkspace() {
  const [{ designerTab }, setDesignerUrlState] = useEDIDesignerUrlState();

  return (
    <Tabs
      value={designerTab}
      onValueChange={(tab) => void setDesignerUrlState({ designerTab: tab as typeof designerTab })}
      className="grid h-[calc(100vh-11rem)] min-h-0 grid-rows-[auto_minmax(0,1fr)] gap-0 overflow-hidden"
    >
      <TabsList variant="underline" className="w-full justify-start border-b border-border">
        <TabsTrigger value="templates" className="max-w-34">
          <Layers3Icon data-icon="inline-start" />
          Templates
        </TabsTrigger>
        <TabsTrigger value="documents" className="max-w-52">
          <ArchiveIcon data-icon="inline-start" />
          Document Preview & Archive
        </TabsTrigger>
      </TabsList>
      <TabsContent value="templates" className="m-0 min-h-0 overflow-hidden pt-3">
        <Suspense fallback={<DesignerLoadingBlock />}>
          <TemplateDesignerTab />
        </Suspense>
      </TabsContent>
      <TabsContent value="documents" className="m-0 min-h-0 overflow-hidden pt-3">
        <Suspense fallback={<DesignerLoadingBlock />}>
          <DocumentPreviewArchiveTab />
        </Suspense>
      </TabsContent>
    </Tabs>
  );
}

export default DesignerWorkspace;
