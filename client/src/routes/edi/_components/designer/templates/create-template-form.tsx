import { Button } from "@/components/ui/button";
import type { CreateTemplateDraft } from "@/types/edi";
import { PlusIcon } from "lucide-react";
import type { Dispatch, SetStateAction } from "react";
import { EDIDocumentTypeAutocompleteField } from "../components/designer-fields";
import { InputBlock } from "../components/designer-shared";
import { functionalGroupForTransactionSet } from "../utils/edi-designer-utils";

type CreateTemplateFormProps = {
  draft: CreateTemplateDraft;
  onChange: Dispatch<SetStateAction<CreateTemplateDraft>>;
  onCreate: () => void;
  isLoading: boolean;
};

export default function CreateTemplateForm({
  draft,
  onChange,
  onCreate,
  isLoading,
}: CreateTemplateFormProps) {
  return (
    <div className="space-y-2 border-t p-3">
      <div className="flex items-center gap-2 text-xs font-semibold">
        <PlusIcon className="size-4" />
        New Template
      </div>
      <EDIDocumentTypeAutocompleteField
        value={draft.documentTypeId}
        onValueChange={(documentTypeId) =>
          onChange((current) => ({
            ...current,
            documentTypeId,
          }))
        }
        onOptionChange={(documentType) => {
          if (!documentType) return;
          onChange((current) => ({
            ...current,
            documentTypeId: documentType.id,
            direction: documentType.direction,
            transactionSet: documentType.transactionSet,
            x12Version: documentType.defaultVersion || current.x12Version,
            functionalGroupId: functionalGroupForTransactionSet(documentType.transactionSet),
          }));
        }}
      />
      <InputBlock
        label="Name"
        value={draft.name}
        onChange={(name) => onChange((current) => ({ ...current, name }))}
      />
      <InputBlock
        label="Description"
        value={draft.description}
        onChange={(description) => onChange((current) => ({ ...current, description }))}
      />
      <div className="grid grid-cols-2 gap-2">
        <InputBlock
          label="X12 Version"
          value={draft.x12Version}
          onChange={(x12Version) => onChange((current) => ({ ...current, x12Version }))}
        />
        <InputBlock
          label="Group"
          value={draft.functionalGroupId}
          onChange={(functionalGroupId) =>
            onChange((current) => ({ ...current, functionalGroupId }))
          }
        />
      </div>
      <Button
        type="button"
        className="w-full"
        onClick={onCreate}
        isLoading={isLoading}
        disabled={!draft.documentTypeId || !draft.name.trim()}
      >
        <PlusIcon className="size-4" />
        Create Template
      </Button>
    </div>
  );
}
