import { Button } from "@/components/ui/button";
import { RefreshCwIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { usePreviewEDIDocumentMutation } from "../hooks/use-edi-document-mutations";
import { EDIDocumentProfileAutocompleteField } from "./designer-fields";
import {
  InputBlock,
  PreviewPane,
  TextareaBlock,
  parsePayload,
} from "./designer-shared";

export default function TemplatePreviewPanel() {
  const [profileId, setProfileId] = useState("");
  const [shipmentId, setShipmentId] = useState("");
  const [transferId, setTransferId] = useState("");
  const [payloadJson, setPayloadJson] = useState("");
  const previewMutation = usePreviewEDIDocumentMutation({
    onError: () => toast.error("Failed to preview EDI document"),
  });
  const canPreview = !!profileId && (!!shipmentId || !!transferId || !!payloadJson.trim());

  return (
    <div className="grid h-full grid-cols-[360px_minmax(0,1fr)]">
      <div className="space-y-3 border-r p-3">
        <EDIDocumentProfileAutocompleteField
          value={profileId}
          onValueChange={setProfileId}
        />
        <InputBlock label="Shipment ID" value={shipmentId} onChange={setShipmentId} />
        <InputBlock label="Transfer ID" value={transferId} onChange={setTransferId} />
        <TextareaBlock label="Payload JSON" value={payloadJson} onChange={setPayloadJson} />
        <Button
          type="button"
          onClick={() =>
            previewMutation.mutate({
              partnerDocumentProfileId: profileId || undefined,
              shipmentId: shipmentId || undefined,
              transferId: transferId || undefined,
              payload: parsePayload(payloadJson),
            })
          }
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
