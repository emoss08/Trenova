import { Sheet, SheetContent } from "@trenova/shared/components/ui/sheet";
import type { EDIDocumentPreview } from "@trenova/shared/types/edi";
import type { Extension } from "@codemirror/state";
import { useEffect, useMemo } from "react";
import { useInspectX12Mutation } from "../hooks/use-edi-document-mutations";
import InspectorHeader from "./components/inspector-header";
import InspectorState from "./components/inspector-states";
import InspectorTabs, { type InspectorTab } from "./components/inspector-tabs";
import { buildPreviewInspectRequest, buildPreviewInspectorContext } from "./inspector-context";

export default function PreviewInspectorSheet({
  preview,
  open,
  selectedTab,
  selectedSegmentIndex,
  editorTheme,
  onOpenChange,
  onTabChange,
  onSelectSegment,
}: {
  preview?: EDIDocumentPreview;
  open: boolean;
  selectedTab: InspectorTab;
  selectedSegmentIndex: number;
  editorTheme: Extension;
  onOpenChange: (open: boolean) => void;
  onTabChange: (tab: InspectorTab) => void;
  onSelectSegment: (segmentIndex: number) => void;
}) {
  const inspectMutation = useInspectX12Mutation();
  const { mutate } = inspectMutation;
  const context = useMemo(
    () => (preview ? buildPreviewInspectorContext(preview) : undefined),
    [preview],
  );
  const request = useMemo(
    () => (preview ? buildPreviewInspectRequest(preview) : undefined),
    [preview],
  );

  useEffect(() => {
    if (!open || !request) return;
    mutate(request);
  }, [mutate, open, request]);

  const inspection = inspectMutation.data;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-[min(1280px,calc(100vw-2rem))] gap-0 p-0 sm:max-w-none">
        <InspectorHeader context={context} fallbackTitle="Preview inspection" />
        {!preview || !context ? (
          <InspectorState state="empty" message="Preview output is unavailable." />
        ) : inspectMutation.isPending ? (
          <InspectorState state="loading" message="Inspecting preview output." />
        ) : inspectMutation.isError ? (
          <InspectorState state="error" message={inspectionErrorMessage(inspectMutation.error)} />
        ) : !inspection ? (
          <InspectorState state="empty" message="Inspection output is unavailable." />
        ) : (
          <InspectorTabs
            context={context}
            inspection={inspection}
            selectedTab={selectedTab}
            selectedSegmentIndex={selectedSegmentIndex}
            editorTheme={editorTheme}
            onTabChange={onTabChange}
            onSelectSegment={onSelectSegment}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}

function inspectionErrorMessage(error: unknown) {
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return "Unable to inspect preview output.";
}
