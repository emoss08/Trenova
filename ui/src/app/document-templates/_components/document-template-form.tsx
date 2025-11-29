import { documentTemplateEditorParser } from "@/app/workers/_components/pto/use-document-template-state";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import { cn } from "@/lib/utils";
import { useQueryStates } from "nuqs";
import { Activity } from "react";
import { DocumentTemplateEditorForm } from "./_form/document-template-editor-form";
import { DocumentTemplateSettingsForm } from "./_form/document-template-settings-form";
import { DocumentTemplatePreview } from "./_preview/document-template-preview";

export function DocumentTemplateForm() {
  const [searchParams] = useQueryStates(documentTemplateEditorParser);

  return (
    <DocumentTemplateOuter>
      <ResizablePanelGroup direction="horizontal">
        <ResizablePanel
          className="bg-sidebar"
          defaultSize={20}
          minSize={18}
          maxSize={35}
        >
          <DocumentTemplateSettingsForm />
        </ResizablePanel>
        <ResizableHandle withHandle />
        <ResizablePanel
          defaultSize={searchParams.showPreview ? 48 : 80}
          minSize={25}
        >
          <DocumentTemplateEditorForm />
        </ResizablePanel>
        <Activity mode={searchParams.showPreview ? "visible" : "hidden"}>
          <ResizableHandle withHandle />
          <ResizablePanel defaultSize={32} minSize={20} maxSize={50}>
            <DocumentTemplatePreview />
          </ResizablePanel>
        </Activity>
      </ResizablePanelGroup>
    </DocumentTemplateOuter>
  );
}

function DocumentTemplateOuter({ children }: { children: React.ReactNode }) {
  const [searchParams] = useQueryStates(documentTemplateEditorParser);

  return (
    <div
      className={cn(
        searchParams.isFullscreen && "fixed inset-0 z-50 bg-background",
      )}
    >
      {children}
    </div>
  );
}
