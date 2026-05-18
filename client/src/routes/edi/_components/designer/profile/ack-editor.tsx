import { Switch } from "@/components/ui/switch";
import type { UpsertEDIPartnerDocumentProfileRequest } from "@/types/edi";
import type { Dispatch, SetStateAction } from "react";
import { InputBlock, SelectBlock } from "../components/designer-shared";

export function AckEditor({
  profile,
  onChange,
}: {
  profile: UpsertEDIPartnerDocumentProfileRequest;
  onChange: Dispatch<SetStateAction<UpsertEDIPartnerDocumentProfileRequest>>;
}) {
  return (
    <div className="space-y-2 rounded-md border bg-muted/30 p-2">
      <div className="flex items-center justify-between">
        <div className="text-xs font-medium">Acknowledgment</div>
        <Switch
          checked={profile.acknowledgment.expected}
          onCheckedChange={(expected) =>
            onChange((current) => ({
              ...current,
              acknowledgment: { ...current.acknowledgment, expected },
            }))
          }
        />
      </div>
      <div className="grid grid-cols-2 gap-2">
        <SelectBlock
          label="Type"
          value={profile.acknowledgment.type}
          onValueChange={(type) =>
            onChange((current) => ({
              ...current,
              acknowledgment: { ...current.acknowledgment, type },
            }))
          }
          options={[
            { value: "None", label: "None" },
            { value: "997", label: "997" },
            { value: "999", label: "999" },
          ]}
        />
        <InputBlock
          label="SLA Minutes"
          value={String(profile.acknowledgment.slaInMinutes)}
          onChange={(slaInMinutes) =>
            onChange((current) => ({
              ...current,
              acknowledgment: {
                ...current.acknowledgment,
                slaInMinutes: Number(slaInMinutes) || 0,
              },
            }))
          }
        />
      </div>
    </div>
  );
}

export default AckEditor;
