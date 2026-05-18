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
      className="min-h-[calc(100vh-11rem)] gap-3"
    >
      <TabsList className="grid w-fit max-w-full grid-cols-2 overflow-x-auto">
        <TabsTrigger value="templates">
          <Layers3Icon data-icon="inline-start" />
          Templates
        </TabsTrigger>
        <TabsTrigger value="documents">
          <ArchiveIcon data-icon="inline-start" />
          Document Preview & Archive
        </TabsTrigger>
      </TabsList>
      <TabsContent value="templates" className="min-h-0">
        <Suspense fallback={<DesignerLoadingBlock />}>
          <TemplateDesignerTab />
        </Suspense>
      </TabsContent>
      <TabsContent value="documents" className="min-h-0">
        <Suspense fallback={<DesignerLoadingBlock />}>
          <DocumentPreviewArchiveTab />
        </Suspense>
      </TabsContent>
    </Tabs>
  );
}

export default DesignerWorkspace;
