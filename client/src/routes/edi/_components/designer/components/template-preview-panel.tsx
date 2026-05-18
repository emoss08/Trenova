import { Button } from "@/components/ui/button";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { RefreshCwIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { usePreviewEDIDocumentMutation } from "../hooks/use-edi-document-mutations";
import {
  InputBlock,
  PreviewPane,
  SelectBlock,
  TextareaBlock,
  parsePayload,
} from "./designer-shared";

export default function TemplatePreviewPanel() {
  const [profileId, setProfileId] = useState("");
  const [shipmentId, setShipmentId] = useState("");
  const [transferId, setTransferId] = useState("");
  const [payloadJson, setPayloadJson] = useState("");
  const profilesQuery = useQuery(
    queries.edi.documentProfiles("?limit=100&transactionSet=204&direction=Outbound"),
  );
  const previewMutation = usePreviewEDIDocumentMutation({
    onError: () => toast.error("Failed to preview EDI document"),
  });
  const canPreview = !!profileId && (!!shipmentId || !!transferId || !!payloadJson.trim());

  return (
    <div className="grid min-h-0 grid-cols-[360px_minmax(0,1fr)]">
      <div className="space-y-3 border-r p-3">
        <SelectBlock
          label="Document Profile"
          value={profileId}
          onValueChange={setProfileId}
          options={(profilesQuery.data?.results ?? []).map((profile) => ({
            value: profile.id,
            label: profile.name,
          }))}
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
