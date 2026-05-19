import { Sheet, SheetContent } from "@/components/ui/sheet";
import { useMemo } from "react";
import { useEditorTheme } from "../components/designer-shared";
import { useEDIMessageDetailQuery } from "../hooks/use-edi-document-queries";
import InspectorHeader from "./components/inspector-header";
import InspectorState from "./components/inspector-states";
import InspectorTabs, { type InspectorTab } from "./components/inspector-tabs";
import { parseX12Document } from "./utils/x12-parser";

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
  const messageQuery = useEDIMessageDetailQuery(open ? messageId : "");
  const message = messageQuery.data;
  const editorTheme = useEditorTheme();
  const document = useMemo(
    () => parseX12Document(message?.rawX12 ?? "", message?.partnerDocumentProfile?.envelope),
    [message?.partnerDocumentProfile?.envelope, message?.rawX12],
  );

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-[min(1280px,calc(100vw-2rem))] gap-0 p-0 sm:max-w-none">
        <InspectorHeader message={message} messageId={messageId} />
        {!message ? (
          <InspectorState state={messageQuery.isLoading ? "loading" : "empty"} />
        ) : (
          <InspectorTabs
            message={message}
            document={document}
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
