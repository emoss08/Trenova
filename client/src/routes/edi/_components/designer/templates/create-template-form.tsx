import { Button } from "@/components/ui/button";
import type { CreateTemplateDraft, EDIDocumentType } from "@/types/edi";
import { PlusIcon } from "lucide-react";
import type { Dispatch, SetStateAction } from "react";
import { InputBlock, SelectBlock } from "../components/designer-shared";

type CreateTemplateFormProps = {
  documentTypes: EDIDocumentType[];
  draft: CreateTemplateDraft;
  onChange: Dispatch<SetStateAction<CreateTemplateDraft>>;
  onCreate: () => void;
  isLoading: boolean;
};

export default function CreateTemplateForm({
  documentTypes,
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
      <SelectBlock
        label="Document Type"
        value={draft.documentTypeId}
        onValueChange={(documentTypeId) => {
          const documentType = documentTypes.find((item) => item.id === documentTypeId);
          onChange((current) => ({
            ...current,
            documentTypeId,
            x12Version: documentType?.defaultVersion ?? current.x12Version,
          }));
        }}
        options={documentTypes.map((documentType) => ({
          value: documentType.id,
          label: `${documentType.code} - ${documentType.name}`,
        }))}
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
