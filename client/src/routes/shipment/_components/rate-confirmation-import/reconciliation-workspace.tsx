import { Button } from "@/components/ui/button";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import { ScrollArea } from "@/components/ui/scroll-area";
import { LoaderCircleIcon } from "lucide-react";
import { useState } from "react";
import type { Control, Path } from "react-hook-form";
import { AIActivityPanel } from "./ai-activity-panel";
import { DocumentPreviewPanel } from "./document-preview-panel";
import { FieldReconciliationList } from "./field-reconciliation-list";
import { ReconciliationHeader } from "./reconciliation-header";
import { RequiredFieldsSection } from "./required-fields-section";
import { StopReconciliationCard } from "./stop-reconciliation-card";
import type { ReconciliationCounts, ReconciliationState, RequiredFieldsForm } from "./types";

type ReconciliationWorkspaceProps = {
  documentId: string;
  fileName?: string;
  state: ReconciliationState;
  counts: ReconciliationCounts;
  issueCount: number;
  onAcceptField: (key: string) => void;
  onEditField: (key: string, value: unknown) => void;
  onResetField: (key: string) => void;
  onAcceptAllConfident: () => void;
  onEditStopField: (stopIndex: number, fieldKey: string, value: unknown) => void;
  requiredFieldsControl: Control<RequiredFieldsForm>;
  canCreateShipment: boolean;
  isCreating: boolean;
  onCreateShipment: () => void;
  hasRequiredValues: boolean;
  requiredFieldValues: {
    customerId: string;
    serviceTypeId: string;
    shipmentTypeId: string;
    formulaTemplateId: string;
  };
  onSetRequiredField: (fieldKey: string, value: string) => void;
  onSetStopLocation: (stopIndex: number, locationId: string) => void;
  onSetStopSchedule: (stopIndex: number, windowStart: string, windowEnd?: string) => void;
  onSetShipmentField: (field: string, value: string) => void;
  onShipmentCreated?: (shipmentId: string) => void;
  lastCreateError?: string | null;
  onClearCreateError?: () => void;
};

export function ReconciliationWorkspace({
  documentId,
  fileName,
  state,
  counts,
  issueCount,
  onAcceptField,
  onEditField,
  onResetField,
  onAcceptAllConfident,
  onEditStopField,
  requiredFieldsControl,
  canCreateShipment,
  isCreating,
  onCreateShipment,
  hasRequiredValues,
  requiredFieldValues,
  onSetRequiredField,
  onSetStopLocation,
  onSetStopSchedule,
  onSetShipmentField,
  onShipmentCreated,
  lastCreateError,
  onClearCreateError,
}: ReconciliationWorkspaceProps) {
  const [showIssuesOnly, setShowIssuesOnly] = useState(false);

  return (
    <div className="flex flex-1 flex-col min-h-0">
      <ResizablePanelGroup orientation="horizontal" className="flex-1 min-h-0">
        {/* PDF viewer */}
        <ResizablePanel defaultSize={35} minSize={20}>
          <DocumentPreviewPanel documentId={documentId} fileName={fileName} />
        </ResizablePanel>
        <ResizableHandle withHandle />

        {/* Reconciliation */}
        <ResizablePanel defaultSize={40} minSize={30}>
          <div className="flex h-full flex-col">
            <ReconciliationHeader
              overallConfidence={state.overallConfidence}
              counts={counts}
              issueCount={issueCount}
              onAcceptAllConfident={onAcceptAllConfident}
              onToggleFilter={() => setShowIssuesOnly(!showIssuesOnly)}
              showIssuesOnly={showIssuesOnly}
            />

            <ScrollArea className="flex-1 min-h-0">
              <div className="border-b">
                <div className="px-3 pt-2 pb-1">
                  <span className="text-2xs font-medium uppercase tracking-wider text-muted-foreground/50">
                    Required Details
                  </span>
                </div>
                <RequiredFieldsSection
                  control={requiredFieldsControl}
                  hasValues={hasRequiredValues}
                />
              </div>

              <div className="border-b">
                <div className="px-3 pt-3 pb-1">
                  <span className="text-2xs font-medium uppercase tracking-wider text-muted-foreground/50">
                    Extracted Fields
                  </span>
                </div>
                <FieldReconciliationList
                  fields={state.fields}
                  showIssuesOnly={showIssuesOnly}
                  onAccept={onAcceptField}
                  onEdit={onEditField}
                  onReset={onResetField}
                />
              </div>

              {state.stops.length > 0 && (
                <div>
                  <div className="px-3 pt-3 pb-1">
                    <span className="text-2xs font-medium uppercase tracking-wider text-muted-foreground/50">
                      Stops
                    </span>
                  </div>
                  <div className="space-y-2 px-3 py-2">
                    {state.stops.map((stop, index) => (
                      <StopReconciliationCard
                        key={`${stop.role}-${stop.sequence}`}
                        stop={stop}
                        index={index}
                        onEditField={onEditStopField}
                        formControl={requiredFieldsControl}
                        locationFieldName={`stops.${index}.locationId` as Path<RequiredFieldsForm>}
                      />
                    ))}
                  </div>
                </div>
              )}
            </ScrollArea>

            <div className="shrink-0 flex items-center justify-between border-t bg-muted/30 px-4 py-2.5">
              <div className="text-xs text-muted-foreground">
                {counts.total} fields
                {issueCount > 0 && (
                  <span className="text-amber-500 ml-1">&middot; {issueCount} need attention</span>
                )}
              </div>
              <Button
                size="sm"
                onClick={onCreateShipment}
                disabled={isCreating || !canCreateShipment}
              >
                {isCreating && <LoaderCircleIcon className="size-3.5 animate-spin" />}
                Create Shipment
              </Button>
            </div>
          </div>
        </ResizablePanel>
        <ResizableHandle withHandle />

        {/* AI Activity feed */}
        <ResizablePanel defaultSize={25} minSize={18} collapsible collapsedSize={0}>
          <AIActivityPanel
            documentId={documentId}
            state={state}
            onAcceptField={onAcceptField}
            onAcceptAllConfident={onAcceptAllConfident}
            onEditField={onEditField}
            onSetRequiredField={onSetRequiredField}
            onSetStopLocation={onSetStopLocation}
            onSetStopSchedule={onSetStopSchedule}
            onSetShipmentField={onSetShipmentField}
            onCreateShipment={onCreateShipment}
            onShipmentCreated={onShipmentCreated}
            requiredFieldValues={requiredFieldValues}
            lastCreateError={lastCreateError}
            onClearCreateError={onClearCreateError}
          />
        </ResizablePanel>
      </ResizablePanelGroup>
    </div>
  );
}
