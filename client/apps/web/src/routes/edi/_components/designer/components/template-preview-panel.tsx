import { Button } from "@trenova/shared/components/ui/button";
import { DocumentSourceControls } from "@/components/edi/document-source-controls";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { ControlledEDIDocumentProfileAutocompleteField } from "@/components/autocomplete-fields";
import {
  buildEDIDocumentResolutionRequest,
  hasEDIDocumentSourceValue,
  pruneEDIDocumentSourceValues,
  type EDIDocumentSourceField,
  type EDIDocumentSourceValues,
} from "@/lib/edi/document-source";
import type { EDIPartnerDocumentProfile } from "@trenova/shared/types/edi";
import { RefreshCwIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { usePreviewEDIDocumentMutation } from "../hooks/use-edi-document-mutations";
import { PreviewPane, parsePayload } from "./designer-shared";

export default function TemplatePreviewPanel() {
  const [profileId, setProfileId] = useState("");
  const [selectedProfile, setSelectedProfile] = useState<EDIPartnerDocumentProfile | null>(null);
  const [sourceValues, setSourceValues] = useState<EDIDocumentSourceValues>({});
  const previewMutation = usePreviewEDIDocumentMutation({
    onError: () => toast.error("Failed to preview EDI document"),
  });
  const transactionSet = selectedProfile?.transactionSet;
  const direction = selectedProfile?.direction;
  const canPreview = !!profileId && hasEDIDocumentSourceValue(sourceValues, transactionSet);

  useEffect(() => {
    setSourceValues((current) => pruneEDIDocumentSourceValues(current, transactionSet));
  }, [transactionSet]);

  const setSourceValue = (field: EDIDocumentSourceField, value: string) => {
    setSourceValues((current) => ({ ...current, [field]: value }));
  };

  return (
    <div className="grid h-full min-h-0 grid-cols-[360px_minmax(0,1fr)] overflow-hidden">
      <ScrollArea className="min-h-0 border-r" viewportClassName="min-h-0">
        <div className="space-y-3 p-3">
          <ControlledEDIDocumentProfileAutocompleteField
            value={profileId}
            onValueChange={(nextProfileId) => {
              setProfileId(nextProfileId);
              setSelectedProfile((current) => (current?.id === nextProfileId ? current : null));
            }}
            onOptionChange={setSelectedProfile}
          />
          <DocumentSourceControls
            transactionSet={transactionSet}
            values={sourceValues}
            onChange={setSourceValue}
          />
          <Button
            type="button"
            onClick={() => {
              const payloadResult = parsePayload(sourceValues.payload ?? "");
              if (!payloadResult.ok) return;
              previewMutation.mutate(
                buildEDIDocumentResolutionRequest({
                  partnerDocumentProfileId: profileId || undefined,
                  sourceValues,
                  transactionSet,
                  direction,
                  payload: payloadResult.payload,
                }),
              );
            }}
            isLoading={previewMutation.isPending}
            disabled={!canPreview}
          >
            <RefreshCwIcon className="size-4" />
            Preview
          </Button>
        </div>
      </ScrollArea>
      <PreviewPane preview={previewMutation.data} isLoading={previewMutation.isPending} />
    </div>
  );
}
