import { FormCreateModal } from "@/components/form-create-modal";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  useCurrentTemplateInvalidation,
  useTemplateDesignerUrlActions,
} from "@/hooks/use-template-designer-state";
import { useTemplateDesignerStore } from "@/stores/template-designer-store";
import type { EDITemplate } from "@/types/edi";
import { zodResolver } from "@hookform/resolvers/zod";
import { FileCode2Icon, FilterIcon, PlusIcon, SearchIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { useForm } from "react-hook-form";
import { useTemplateDesignerUrlState } from "../hooks/use-edi-designer-url-state";
import { CreateTemplateForm, createTemplateFormSchema } from "../templates/create-template-form";
import TemplateList from "../templates/template-list";
import {
  documentDirectionOptions,
  templateStatusOptions,
  transactionSetOptions,
} from "../utils/edi-designer-options";
import { ControlledSelectField } from "./designer-fields";
import { PanelHeader } from "./designer-shared";

export default function TemplateDesignerAside() {
  const [templateUrlState, setTemplateUrlState] = useTemplateDesignerUrlState();
  const { templateSearch, templateStatus, templateTransactionSet, templateDirection } =
    templateUrlState;
  const { patchTemplateUrlState } = useTemplateDesignerUrlActions();
  const resetDraftState = useTemplateDesignerStore((state) => state.resetDraftState);

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
  const resetTemplateFilters = useCallback(
    () =>
      void setTemplateUrlState({
        templateStatus: "",
        templateTransactionSet: "",
        templateDirection: "",
      }),
    [setTemplateUrlState],
  );
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const activeFilterCount = [templateStatus, templateTransactionSet, templateDirection].filter(
    Boolean,
  ).length;

  const invalidateTemplateQueries = useCurrentTemplateInvalidation();
  const createTemplateForm = useForm({
    resolver: zodResolver(createTemplateFormSchema),
    defaultValues: {
      documentTypeId: "",
      name: "",
      description: "",
      direction: "Outbound",
      standard: "X12",
      transactionSet: "204",
      x12Version: "004010",
      functionalGroupId: "SM",
      notes: "",
    },
  });

  const handleCreateDialogOpenChange = (open: boolean) => setIsCreateDialogOpen(open);

  const handleTemplateCreated = async (template: EDITemplate) => {
    const versionId = template.versions[0]?.id ?? template.activeVersion?.id ?? "";
    resetDraftState();
    patchTemplateUrlState({
      templateId: template.id,
      versionId,
      segmentId: "",
      elementPosition: 0,
    });
    await invalidateTemplateQueries();
  };

  return (
    <aside className="flex h-full min-h-0 flex-col overflow-hidden rounded-md border bg-background">
      <PanelHeader
        icon={<FileCode2Icon />}
        title="Templates"
        actions={
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => setIsCreateDialogOpen(true)}
          >
            <PlusIcon className="size-3.5" />
            New
          </Button>
        }
      />
      <div className="border-b p-3">
        <div className="flex w-full items-center gap-2">
          <Input
            value={templateSearch}
            onChange={(event) => setTemplateSearch(event.target.value)}
            placeholder="Search templates"
            inputContainerClassName="w-full"
            leftElement={<SearchIcon className="size-3 text-muted-foreground" />}
          />
          <TemplateFilterPopover
            activeFilterCount={activeFilterCount}
            templateStatus={templateStatus}
            templateTransactionSet={templateTransactionSet}
            templateDirection={templateDirection}
            onStatusChange={setTemplateStatus}
            onTransactionSetChange={setTemplateTransactionSet}
            onDirectionChange={setTemplateDirection}
            onReset={resetTemplateFilters}
          />
        </div>
      </div>
      <ScrollArea className="min-h-0 flex-1" viewportClassName="min-h-0">
        <TemplateList />
      </ScrollArea>
      <FormCreateModal
        open={isCreateDialogOpen}
        onOpenChange={handleCreateDialogOpenChange}
        title="EDI Template"
        description="Choose a document type and name the EDI template before editing its version details."
        url="/edi/templates/"
        queryKey="templates"
        form={createTemplateForm}
        formComponent={<CreateTemplateForm />}
        className="sm:max-w-120"
        submitText="Create Template"
        loadingText="Creating..."
        onSuccess={handleTemplateCreated}
      />
    </aside>
  );
}

function TemplateFilterPopover({
  activeFilterCount,
  templateStatus,
  templateTransactionSet,
  templateDirection,
  onStatusChange,
  onTransactionSetChange,
  onDirectionChange,
  onReset,
}: {
  activeFilterCount: number;
  templateStatus: string;
  templateTransactionSet: string;
  templateDirection: string;
  onStatusChange: (value: string) => void;
  onTransactionSetChange: (value: string) => void;
  onDirectionChange: (value: string) => void;
  onReset: () => void;
}) {
  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button type="button" variant="outline" className="shrink-0">
            <FilterIcon className="size-4" />
            <span className="text-xs">Filter</span>
            {activeFilterCount > 0 ? (
              <Badge variant="active" className="ml-0.5 px-1.5 py-0 text-[10px]">
                {activeFilterCount}
              </Badge>
            ) : null}
          </Button>
        }
      />
      <PopoverContent align="end" className="w-72 p-0">
        <div className="border-b px-3 py-2">
          <div className="text-sm font-semibold">Template Filters</div>
          <div className="text-xs text-muted-foreground">Narrow the template list.</div>
        </div>
        <div className="space-y-3 p-3">
          <ControlledSelectField
            label="Status"
            value={templateStatus}
            onValueChange={onStatusChange}
            options={templateStatusOptions}
            placeholder="All statuses"
          />
          <ControlledSelectField
            label="Set"
            value={templateTransactionSet}
            onValueChange={onTransactionSetChange}
            options={transactionSetOptions}
            placeholder="All sets"
          />
          <ControlledSelectField
            label="Direction"
            value={templateDirection}
            onValueChange={onDirectionChange}
            options={documentDirectionOptions}
            placeholder="All"
          />
        </div>
        <div className="flex justify-end border-t p-2">
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={onReset}
            disabled={activeFilterCount === 0}
          >
            Reset
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}
