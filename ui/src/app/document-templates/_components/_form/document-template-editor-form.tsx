import { DocumentTemplateEditorControls } from "../document-template-panels";
import { DocumentTemplateEditorTabContent } from "./document-template-editor-tabs";
import { DocumentTemplateTabContent } from "./document-template-tabs";

export function DocumentTemplateEditorForm() {
  return (
    <DocumentTemplateEditorOuter>
      <DocumentTemplateEditorInner>
        <DocumentTemplateEditorTabContent />
        <DocumentTemplateEditorControls />
      </DocumentTemplateEditorInner>
      <DocumentTemplateTabContent />
    </DocumentTemplateEditorOuter>
  );
}

function DocumentTemplateEditorOuter({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex flex-col">{children}</div>;
}

function DocumentTemplateEditorInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex h-[55px] shrink-0 items-center justify-between border-b border-border bg-background px-4 py-2">
      {children}
    </div>
  );
}
