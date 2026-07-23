import { Sheet, SheetContent } from "@trenova/shared/components/ui/sheet";
import { useEditorTheme } from "../components/designer-shared";
import { useEDIMessageInspectionQuery } from "../hooks/use-edi-document-queries";
import InspectorHeader from "./components/inspector-header";
import InspectorState from "./components/inspector-states";
import InspectorTabs, { type InspectorTab } from "./components/inspector-tabs";
import { buildMessageInspectorContext } from "./inspector-context";

export default function MessageInspectorSheet({
  messageId,
  open,
  selectedTab,
  selectedSegmentIndex,
  onOpenChange,
  onTabChange,
  onSelectSegment,
}: {
  messageId: string;
  open: boolean;
  selectedTab: InspectorTab;
  selectedSegmentIndex: number;
  onOpenChange: (open: boolean) => void;
  onTabChange: (tab: InspectorTab) => void;
  onSelectSegment: (segmentIndex: number) => void;
}) {
  const inspectionQuery = useEDIMessageInspectionQuery(open ? messageId : "");
  const context = inspectionQuery.data
    ? buildMessageInspectorContext(inspectionQuery.data)
    : undefined;
  const inspection = inspectionQuery.data?.inspection;
  const editorTheme = useEditorTheme();

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-[min(1280px,calc(100vw-2rem))] gap-0 p-0 sm:max-w-none">
        <InspectorHeader context={context} fallbackTitle={`Message ${messageId}`} />
        {!context || !inspection ? (
          <InspectorState state={inspectionQuery.isLoading ? "loading" : "empty"} />
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
