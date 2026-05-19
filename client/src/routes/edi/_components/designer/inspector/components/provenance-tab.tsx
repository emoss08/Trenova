import type { EDIMessage } from "@/types/edi";
import InspectorGrid from "./inspector-grid";
import { scriptLibraryLabel, versionLabel } from "./overview-tab";

export default function ProvenanceTab({ message }: { message: EDIMessage }) {
  return (
    <InspectorGrid
      rows={[
        ["Profile ID", message.partnerDocumentProfileId],
        ["Profile Name", message.partnerDocumentProfile?.name ?? "-"],
        ["Template ID", message.templateId],
        [
          "Template Name",
          message.template?.name ?? message.partnerDocumentProfile?.template?.name ?? "-",
        ],
        ["Template Version ID", message.templateVersionId],
        ["Template Version", versionLabel(message)],
        ["Template Version Status", message.templateVersion?.status ?? "-"],
        ["Script Libraries", scriptLibraryLabel(message)],
        ["Source X12 Version", message.templateVersion?.x12Version ?? message.x12Version],
        ["Validation Mode", message.validationMode],
      ]}
    />
  );
}
