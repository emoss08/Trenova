import { Alert, AlertDescription } from "@/components/ui/alert";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { EDIX12Inspection } from "@/types/edi";
import type { useEditorTheme } from "../../components/designer-shared";
import type { InspectorContext } from "../inspector-context";
import ControlNumbersTab from "./control-numbers-tab";
import DiagnosticsTab from "./diagnostics-tab";
import FormattedViewTab from "./formatted-view-tab";
import OverviewTab from "./overview-tab";
import PayloadTab from "./payload-tab";
import ProvenanceTab from "./provenance-tab";
import RawViewTab from "./raw-view-tab";
import SegmentTreeTab from "./segment-tree-tab";

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
  context,
  inspection,
  selectedTab,
  selectedSegmentIndex,
  editorTheme,
  onTabChange,
  onSelectSegment,
}: {
  context: InspectorContext;
  inspection: EDIX12Inspection;
  selectedTab: InspectorTab;
  selectedSegmentIndex: number;
  editorTheme: ReturnType<typeof useEditorTheme>;
  onTabChange: (tab: InspectorTab) => void;
  onSelectSegment: (segmentIndex: number) => void;
}) {
  const countComparison = inspection.transactions.find(
    (transaction) =>
      transaction.expectedSegments > 0 &&
      transaction.actualSegments > 0 &&
      transaction.expectedSegments !== transaction.actualSegments,
  );
  const tabs = inspectorTabs.filter((tab) => {
    if (tab === "payload") return !!context.payload;
    if (tab === "provenance") return !!context.provenanceRows;
    return true;
  });
  const tabGridClass = tabs.length === 8 ? "grid-cols-8" : "grid-cols-6";
  const activeTab = tabs.includes(selectedTab) ? selectedTab : "overview";

  return (
    <Tabs
      value={activeTab}
      onValueChange={(value) => onTabChange(value as InspectorTab)}
      className="grid min-h-0 flex-1 grid-rows-[auto_minmax(0,1fr)] gap-0"
    >
      <div className="overflow-x-auto p-3 pb-0">
        <TabsList
          variant="underline"
          className={`grid w-max ${tabGridClass} border-b border-border`}
        >
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="controls">Controls</TabsTrigger>
          <TabsTrigger value="raw">Raw</TabsTrigger>
          <TabsTrigger value="formatted">Formatted</TabsTrigger>
          <TabsTrigger value="segments">Segments</TabsTrigger>
          <TabsTrigger value="diagnostics">Diagnostics</TabsTrigger>
          {context.payload ? <TabsTrigger value="payload">Payload</TabsTrigger> : null}
          {context.provenanceRows ? <TabsTrigger value="provenance">Provenance</TabsTrigger> : null}
        </TabsList>
      </div>
      <div className="min-h-0 overflow-auto p-3">
        {countComparison ? (
          <Alert variant="warning" className="mb-3">
            <AlertDescription>
              SE01 reports {countComparison.expectedSegments} segments, but the transaction contains{" "}
              {countComparison.actualSegments}.
            </AlertDescription>
          </Alert>
        ) : null}
        <TabsContent value="overview" className="m-0 min-h-0">
          <OverviewTab context={context} inspection={inspection} />
        </TabsContent>
        <TabsContent value="controls" className="m-0 min-h-0">
          <ControlNumbersTab context={context} />
        </TabsContent>
        <TabsContent value="raw" className="m-0 h-[calc(100vh-14rem)] min-h-0">
          <RawViewTab
            context={context}
            inspection={inspection}
            selectedSegmentIndex={selectedSegmentIndex}
            editorTheme={editorTheme}
          />
        </TabsContent>
        <TabsContent value="formatted" className="m-0 h-[calc(100vh-14rem)] min-h-0">
          <FormattedViewTab
            inspection={inspection}
            diagnostics={inspection.diagnostics}
            onSelectSegment={onSelectSegment}
          />
        </TabsContent>
        <TabsContent value="segments" className="m-0 h-[calc(100vh-14rem)] min-h-0">
          <SegmentTreeTab
            inspection={inspection}
            diagnostics={inspection.diagnostics}
            selectedSegmentIndex={selectedSegmentIndex}
            onSelectSegment={onSelectSegment}
          />
        </TabsContent>
        <TabsContent value="diagnostics" className="m-0 min-h-0">
          <DiagnosticsTab
            diagnostics={inspection.diagnostics}
            inspection={inspection}
            onSelectSegment={onSelectSegment}
          />
        </TabsContent>
        {context.payload ? (
          <TabsContent value="payload" className="m-0 h-[calc(100vh-14rem)] min-h-0">
            <PayloadTab context={context} editorTheme={editorTheme} />
          </TabsContent>
        ) : null}
        {context.provenanceRows ? (
          <TabsContent value="provenance" className="m-0 min-h-0">
            <ProvenanceTab context={context} />
          </TabsContent>
        ) : null}
      </div>
    </Tabs>
  );
}
