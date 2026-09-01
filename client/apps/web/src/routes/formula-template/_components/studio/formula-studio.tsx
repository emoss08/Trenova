import { insertSnippet } from "@/components/formula-editor/insert-at-cursor";
import { UnsavedChangesGuard } from "@/components/unsaved-changes-guard";
import { useKnownIdentifiers } from "@/hooks/use-formula-schema";
import { apiService } from "@/services/api";
import {
  buildTemplateExport,
  downloadJson,
  getExportFilename,
} from "@/lib/formula-template-export";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@trenova/shared/components/ui/resizable";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import type {
  FormulaTemplate,
  FormulaTemplateFormValues,
  VariableDefinition,
} from "@trenova/shared/types/formula-template";
import { useCallback, useRef, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { ApprovalActionDialog, type ApprovalAction } from "../approval-action-dialog";
import { ForkLineageDialog } from "../fork-lineage-dialog";
import { ForkTemplateDialog } from "../fork-template-dialog";
import { VersionHistoryPanel } from "../version/version-history-panel";
import { AiGeneratePanel } from "./ai/ai-generate-panel";
import { BacktestSheet } from "./backtest-sheet";
import { ImportTemplateDialog } from "./import-template-dialog";
import { StudioEditorPane } from "./studio-editor-pane";
import { StudioHeader } from "./studio-header";
import { StudioPreviewPane } from "./studio-preview-pane";
import { StudioReferencePane } from "./studio-reference-pane";
import { StudioScenariosPane } from "./studio-scenarios-pane";
import { useLivePreview } from "./use-live-preview";
import { Button } from "@trenova/shared/components/ui/button";
import { cn } from "@trenova/shared/lib/utils";

type FormulaStudioProps = {
  mode: "create" | "edit";
  template: FormulaTemplate | null;
  isSubmitting: boolean;
  onSave: () => void;
  onTemplateChanged?: (template: FormulaTemplate) => void;
};

export function FormulaStudio({
  mode,
  template,
  isSubmitting,
  onSave,
  onTemplateChanged,
}: FormulaStudioProps) {
  const form = useFormContext<FormulaTemplateFormValues>();
  const editorRef = useRef<ReactCodeMirrorRef>(null);

  const [approvalAction, setApprovalAction] = useState<ApprovalAction | null>(null);
  const [versionHistoryOpen, setVersionHistoryOpen] = useState(false);
  const [forkDialogOpen, setForkDialogOpen] = useState(false);
  const [lineageDialogOpen, setLineageDialogOpen] = useState(false);
  const [importDialogOpen, setImportDialogOpen] = useState(false);
  const [backtestOpen, setBacktestOpen] = useState(false);
  const [aiGenerateOpen, setAiGenerateOpen] = useState(false);
  const [rightTab, setRightTab] = useState<"preview" | "scenarios">("preview");

  const templateName = useWatch({ control: form.control, name: "name" });
  const schemaId = useWatch({ control: form.control, name: "schemaId" });
  const templateType = useWatch({ control: form.control, name: "type" });
  const customVariables = useWatch({ control: form.control, name: "variableDefinitions" });

  const known = useKnownIdentifiers(schemaId || "shipment", customVariables ?? []);
  const preview = useLivePreview();

  const handleInsert = useCallback((text: string, cursorOffset?: number) => {
    const view = editorRef.current?.view;
    if (!view) return;
    insertSnippet(view, text, cursorOffset);
  }, []);

  const handleAiInsert = useCallback(
    (result: { expression: string; variableDefinitions: VariableDefinition[] }) => {
      form.setValue("expression", result.expression, { shouldDirty: true, shouldValidate: true });

      if (result.variableDefinitions.length > 0) {
        const existing = form.getValues("variableDefinitions") ?? [];
        const existingNames = new Set(existing.map((variable) => variable.name));
        const additions = result.variableDefinitions.filter(
          (variable) => !existingNames.has(variable.name),
        );
        if (additions.length > 0) {
          form.setValue("variableDefinitions", [...existing, ...additions], {
            shouldDirty: true,
          });
        }
      }
    },
    [form],
  );

  const handleExport = useCallback(async () => {
    if (!template) return;
    try {
      const testCases = await apiService.formulaTemplateService.listTestCases(template.id);
      const exportData = buildTemplateExport(template, { testCases });
      const filename = getExportFilename(template, false);
      downloadJson(exportData, filename);
      toast.success("Template exported", { description: filename });
    } catch {
      toast.error("Export failed", {
        description: "Could not export the template. Please try again.",
      });
    }
  }, [template]);

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <UnsavedChangesGuard when={form.formState.isDirty} />
      <StudioHeader
        mode={mode}
        template={template}
        templateName={templateName ?? ""}
        isSubmitting={isSubmitting}
        isDirty={form.formState.isDirty}
        onSave={onSave}
        onApprovalAction={setApprovalAction}
        onVersionHistory={() => setVersionHistoryOpen(true)}
        onFork={() => setForkDialogOpen(true)}
        onLineage={() => setLineageDialogOpen(true)}
        onExport={() => void handleExport()}
        onImport={() => setImportDialogOpen(true)}
        onBacktest={() => setBacktestOpen(true)}
      />

      <ResizablePanelGroup orientation="horizontal" className="min-h-0 flex-1">
        <ResizablePanel defaultSize={55} minSize={40}>
          <StudioEditorPane
            mode={mode}
            known={known}
            editorRef={editorRef}
            onOpenAiGenerate={() => setAiGenerateOpen(true)}
          />
        </ResizablePanel>
        <ResizableHandle withHandle />
        <ResizablePanel defaultSize={45} minSize={28}>
          <ResizablePanelGroup orientation="vertical">
            <ResizablePanel defaultSize={55} minSize={30}>
              <div className="flex h-full flex-col">
                <div className="flex gap-1 border-b px-2 pt-1.5 pb-1">
                  {(["preview", "scenarios"] as const).map((tab) => (
                    <Button
                      key={tab}
                      type="button"
                      variant={rightTab === tab ? "secondary" : "ghost"}
                      size="xs"
                      onClick={() => setRightTab(tab)}
                      className={cn(
                        "h-6 text-xs capitalize",
                        rightTab !== tab && "text-muted-foreground",
                      )}
                    >
                      {tab === "preview" ? "Live Preview" : "Scenarios"}
                    </Button>
                  ))}
                </div>
                <div className="min-h-0 flex-1">
                  {rightTab === "preview" ? (
                    <StudioPreviewPane preview={preview} />
                  ) : (
                    <StudioScenariosPane templateId={template?.id ?? null} preview={preview} />
                  )}
                </div>
              </div>
            </ResizablePanel>
            <ResizableHandle withHandle />
            <ResizablePanel defaultSize={45} minSize={20}>
              <StudioReferencePane known={known} onInsert={handleInsert} />
            </ResizablePanel>
          </ResizablePanelGroup>
        </ResizablePanel>
      </ResizablePanelGroup>

      {approvalAction && (
        <ApprovalActionDialog
          open={approvalAction !== null}
          onOpenChange={(open) => {
            if (!open) setApprovalAction(null);
          }}
          action={approvalAction}
          template={template}
        />
      )}

      <VersionHistoryPanel
        open={versionHistoryOpen}
        onOpenChange={setVersionHistoryOpen}
        template={template}
        onRollback={(updatedTemplate) => {
          form.reset(updatedTemplate as unknown as FormulaTemplateFormValues);
          onTemplateChanged?.(updatedTemplate);
        }}
      />

      <ForkTemplateDialog
        open={forkDialogOpen}
        onOpenChange={setForkDialogOpen}
        template={template}
        onForkSuccess={() => setForkDialogOpen(false)}
      />

      <ForkLineageDialog
        open={lineageDialogOpen}
        onOpenChange={setLineageDialogOpen}
        templateId={template?.id}
        currentTemplateId={template?.id}
      />

      <ImportTemplateDialog open={importDialogOpen} onOpenChange={setImportDialogOpen} />

      <BacktestSheet
        open={backtestOpen}
        onOpenChange={setBacktestOpen}
        form={form}
        template={template}
      />

      <AiGeneratePanel
        open={aiGenerateOpen}
        onOpenChange={setAiGenerateOpen}
        templateType={templateType ?? "FreightCharge"}
        schemaId={schemaId || "shipment"}
        onInsert={handleAiInsert}
      />
    </div>
  );
}
