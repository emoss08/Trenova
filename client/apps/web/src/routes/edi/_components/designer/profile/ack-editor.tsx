import { Switch } from "@/components/ui/switch";
import type { UpsertEDIPartnerDocumentProfileRequest } from "@/types/edi";
import type { Dispatch, SetStateAction } from "react";
import { ControlledSelectField } from "../components/designer-fields";
import { InputBlock } from "../components/designer-shared";
import { acknowledgmentTypeOptions } from "../utils/edi-designer-options";

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
        <ControlledSelectField
          label="Type"
          value={profile.acknowledgment.type}
          onValueChange={(type) =>
            onChange((current) => ({
              ...current,
              acknowledgment: { ...current.acknowledgment, type },
            }))
          }
          options={acknowledgmentTypeOptions}
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
