import { Button } from "@/components/ui/button";
import { DocumentSourceControls } from "@/components/edi/document-source-controls";
import {
  buildEDIDocumentResolutionRequest,
  hasEDIDocumentSourceValue,
  pruneEDIDocumentSourceValues,
  type EDIDocumentSourceField,
  type EDIDocumentSourceValues,
} from "@/lib/edi/document-source";
import type { EDIPartnerDocumentProfile } from "@/types/edi";
import { RefreshCwIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { usePreviewEDIDocumentMutation } from "../hooks/use-edi-document-mutations";
import { EDIDocumentProfileAutocompleteField } from "./designer-fields";
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
    <div className="grid h-full grid-cols-[360px_minmax(0,1fr)]">
      <div className="space-y-3 border-r p-3">
        <EDIDocumentProfileAutocompleteField
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
      <PreviewPane preview={previewMutation.data} isLoading={previewMutation.isPending} />
    </div>
  );
}
