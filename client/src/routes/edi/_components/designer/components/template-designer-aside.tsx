import { Input } from "@/components/ui/input";
import { createTemplateDraftSchema, type EDITemplate } from "@/types/edi";
import { FileCode2Icon, SearchIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";
import { useTemplateDesignerUrlState } from "../hooks/use-edi-designer-url-state";
import {
  useCreateEDITemplateMutation,
  useInvalidateEDITemplateQueries,
} from "../hooks/use-edi-template-mutations";
import CreateTemplateForm from "../templates/create-template-form";
import TemplateList from "../templates/template-list";
import {
  documentDirectionOptions,
  templateStatusOptions,
  transactionSetOptions,
} from "../utils/edi-designer-options";
import { ControlledSelectField } from "./designer-fields";
import { PanelHeader } from "./designer-shared";

type TemplateDesignerAsideProps = {
  templates: EDITemplate[];
  selectedTemplateId: string;
  selectedVersionId: string;
  onSelectTemplate: (templateId: string) => void;
  onTemplateCreated: (templateId: string, versionId: string) => void;
};

function createDefaultTemplateDraft() {
  return createTemplateDraftSchema.parse({
    documentTypeId: "",
    name: "",
    description: "",
    x12Version: "004010",
    functionalGroupId: "SM",
    notes: "",
  });
}

export default function TemplateDesignerAside({
  templates,
  selectedTemplateId,
  selectedVersionId,
  onSelectTemplate,
  onTemplateCreated,
}: TemplateDesignerAsideProps) {
  const [templateUrlState, setTemplateUrlState] = useTemplateDesignerUrlState();
  const { templateSearch, templateStatus, templateTransactionSet, templateDirection } =
    templateUrlState;

  const setTemplateSearch = useCallback(
    (value: string) => void setTemplateUrlState({ templateSearch: value }),
    [setTemplateUrlState],
  );
  const setTemplateStatus = useCallback(
    (value: string) => void setTemplateUrlState({ templateStatus: value }),
    [setTemplateUrlState],
  );
  const setTemplateTransactionSet = useCallback(
    (value: string) => void setTemplateUrlState({ templateTransactionSet: value }),
    [setTemplateUrlState],
  );
  const setTemplateDirection = useCallback(
    (value: string) => void setTemplateUrlState({ templateDirection: value }),
    [setTemplateUrlState],
  );
  const [newTemplate, setNewTemplate] = useState(createDefaultTemplateDraft);

  const invalidateTemplateQueries = useInvalidateEDITemplateQueries(
    selectedTemplateId,
    selectedVersionId,
  );

  const createTemplateMutation = useCreateEDITemplateMutation({
    onSuccess: async (template) => {
      toast.success("EDI template created");
      onTemplateCreated(template.id, template.versions[0]?.id ?? template.activeVersion?.id ?? "");
      setNewTemplate(createDefaultTemplateDraft());
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Failed to create EDI template"),
  });

  return (
    <aside className="flex min-h-0 flex-col rounded-md border bg-background">
      <PanelHeader icon={<FileCode2Icon />} title="Templates" />
      <div className="space-y-3 border-b p-3">
        <div className="flex items-center gap-2">
          <SearchIcon className="size-4 text-muted-foreground" />
          <Input
            value={templateSearch}
            onChange={(event) => setTemplateSearch(event.target.value)}
            placeholder="Search templates"
            className="h-8"
          />
        </div>
        <ControlledSelectField
          label="Status"
          value={templateStatus}
          onValueChange={setTemplateStatus}
          options={templateStatusOptions}
          placeholder="All statuses"
        />
        <div className="grid grid-cols-2 gap-2">
          <ControlledSelectField
            label="Set"
            value={templateTransactionSet}
            onValueChange={setTemplateTransactionSet}
            options={transactionSetOptions}
            placeholder="All sets"
          />
          <ControlledSelectField
            label="Direction"
            value={templateDirection}
            onValueChange={setTemplateDirection}
            options={documentDirectionOptions}
            placeholder="All"
          />
        </div>
      </div>
      <TemplateList
        templates={templates}
        selectedTemplateId={selectedTemplateId}
        onSelect={(templateId) => {
          onSelectTemplate(templateId);
        }}
      />
      <CreateTemplateForm
        draft={newTemplate}
        onChange={setNewTemplate}
        onCreate={() =>
          createTemplateMutation.mutate({
            documentTypeId: newTemplate.documentTypeId,
            name: newTemplate.name,
            description: newTemplate.description,
            direction: newTemplate.direction,
            standard: "X12",
            transactionSet: newTemplate.transactionSet,
            x12Version: newTemplate.x12Version,
            functionalGroupId: newTemplate.functionalGroupId,
            notes: newTemplate.notes,
          })
        }
        isLoading={createTemplateMutation.isPending}
      />
    </aside>
  );
}
