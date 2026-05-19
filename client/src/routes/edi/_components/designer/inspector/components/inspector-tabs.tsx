import { Alert, AlertDescription } from "@/components/ui/alert";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { EDIMessage } from "@/types/edi";
import type { useEditorTheme } from "../../components/designer-shared";
import ControlNumbersTab from "./control-numbers-tab";
import DiagnosticsTab from "./diagnostics-tab";
import FormattedViewTab from "./formatted-view-tab";
import OverviewTab from "./overview-tab";
import PayloadTab from "./payload-tab";
import ProvenanceTab from "./provenance-tab";
import RawViewTab from "./raw-view-tab";
import SegmentTreeTab from "./segment-tree-tab";
import type { ParsedX12Document } from "../utils/x12-parser";

export const inspectorTabs = [
  "overview",
  "controls",
  "raw",
  "formatted",
  "segments",
  "diagnostics",
  "payload",
  "provenance",
] as const;

export type InspectorTab = (typeof inspectorTabs)[number];

export default function InspectorTabs({
  message,
  document,
  selectedTab,
  selectedSegmentIndex,
  editorTheme,
  onTabChange,
  onSelectSegment,
}: {
  message: EDIMessage;
  document: ParsedX12Document;
  selectedTab: InspectorTab;
  selectedSegmentIndex: number;
  editorTheme: ReturnType<typeof useEditorTheme>;
  onTabChange: (tab: InspectorTab) => void;
  onSelectSegment: (segmentIndex: number) => void;
}) {
  const countComparison = document.metadata.seSegmentCount;

  return (
    <Tabs
      value={selectedTab}
      onValueChange={(value) => onTabChange(value as InspectorTab)}
      className="grid min-h-0 flex-1 grid-rows-[auto_minmax(0,1fr)] gap-0"
    >
      <div className="overflow-x-auto p-3 pb-0">
        <TabsList className="grid w-max grid-cols-8">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="controls">Controls</TabsTrigger>
          <TabsTrigger value="raw">Raw</TabsTrigger>
          <TabsTrigger value="formatted">Formatted</TabsTrigger>
          <TabsTrigger value="segments">Segments</TabsTrigger>
          <TabsTrigger value="diagnostics">Diagnostics</TabsTrigger>
          <TabsTrigger value="payload">Payload</TabsTrigger>
          <TabsTrigger value="provenance">Provenance</TabsTrigger>
        </TabsList>
      </div>
      <div className="min-h-0 overflow-auto p-3">
        {countComparison && !countComparison.matches ? (
          <Alert variant="warning" className="mb-3">
            <AlertDescription>
              SE01 reports {countComparison.expected} segments, but the transaction contains{" "}
              {countComparison.actual}.
            </AlertDescription>
          </Alert>
        ) : null}
        <TabsContent value="overview" className="m-0 min-h-0">
          <OverviewTab message={message} />
        </TabsContent>
        <TabsContent value="controls" className="m-0 min-h-0">
          <ControlNumbersTab message={message} />
        </TabsContent>
        <TabsContent value="raw" className="m-0 h-[calc(100vh-14rem)] min-h-0">
          <RawViewTab
            message={message}
            document={document}
            selectedSegmentIndex={selectedSegmentIndex}
            editorTheme={editorTheme}
          />
        </TabsContent>
        <TabsContent value="formatted" className="m-0 h-[calc(100vh-14rem)] min-h-0">
          <FormattedViewTab
            document={document}
            diagnostics={message.validationErrors}
            onSelectSegment={onSelectSegment}
          />
        </TabsContent>
        <TabsContent value="segments" className="m-0 h-[calc(100vh-14rem)] min-h-0">
          <SegmentTreeTab
            document={document}
            diagnostics={message.validationErrors}
            selectedSegmentIndex={selectedSegmentIndex}
            onSelectSegment={onSelectSegment}
          />
        </TabsContent>
        <TabsContent value="diagnostics" className="m-0 min-h-0">
          <DiagnosticsTab
            diagnostics={message.validationErrors}
            document={document}
            onSelectSegment={onSelectSegment}
          />
        </TabsContent>
        <TabsContent value="payload" className="m-0 h-[calc(100vh-14rem)] min-h-0">
          <PayloadTab message={message} editorTheme={editorTheme} />
        </TabsContent>
        <TabsContent value="provenance" className="m-0 min-h-0">
          <ProvenanceTab message={message} />
        </TabsContent>
      </div>
    </Tabs>
  );
}
