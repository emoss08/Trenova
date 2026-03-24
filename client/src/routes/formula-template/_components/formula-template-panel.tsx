import { FormCreatePanel } from "@/components/form-create-panel";
import { TabbedFormEditPanel } from "@/components/tabbed-form-edit-panel";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import type { DataTablePanelProps } from "@/types/data-table";
import {
  formulaTemplateSchema,
  type FormulaTemplate,
  type FormulaTemplateFormValues,
} from "@/types/formula-template";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  BookOpenIcon,
  ClockIcon,
  FlaskConicalIcon,
  GitBranchIcon,
  GitForkIcon,
  NetworkIcon,
} from "lucide-react";
import { lazy, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { ForkLineageDialog } from "./fork-lineage-dialog";
import { ForkTemplateDialog } from "./fork-template-dialog";
import { FormulaTemplateForm } from "./formula-template-form";
import { VersionHistoryPanel } from "./version/version-history-panel";

const FormulaTemplateTestTab = lazy(() => import("./formula-template-test-tab"));
const FormulaTemplateReferenceTab = lazy(() => import("./formula-template-reference-tab"));

type FormulaTemplateHeaderActionsProps = {
  template: FormulaTemplate;
  onVersionHistoryClick: () => void;
  onForkClick: () => void;
  onViewLineageClick: () => void;
};

function FormulaTemplateHeaderActions({
  template,
  onVersionHistoryClick,
  onForkClick,
  onViewLineageClick,
}: FormulaTemplateHeaderActionsProps) {
  return (
    <TooltipProvider>
      <div className="flex items-center gap-1">
        {template.currentVersionNumber && (
          <Badge variant="outline" className="mr-1 font-mono text-xs">
            v{template.currentVersionNumber}
          </Badge>
        )}
        <Tooltip>
          <TooltipTrigger
            render={
              <Button variant="ghost" size="icon-sm" onClick={onVersionHistoryClick}>
                <ClockIcon className="size-4" />
              </Button>
            }
          />
          <TooltipContent>Version History</TooltipContent>
        </Tooltip>
        <DropdownMenu>
          <Tooltip>
            <TooltipTrigger
              render={
                <DropdownMenuTrigger
                  render={
                    <Button variant="ghost" size="icon-sm">
                      <GitBranchIcon className="size-4" />
                    </Button>
                  }
                />
              }
            />
            <TooltipContent>Fork Options</TooltipContent>
          </Tooltip>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              title="Fork Template"
              startContent={<GitForkIcon className="size-4" />}
              onClick={onForkClick}
            />
            <DropdownMenuItem
              title="View Lineage"
              startContent={<NetworkIcon className="size-4" />}
              onClick={onViewLineageClick}
            />
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </TooltipProvider>
  );
}

export function FormulaTemplatePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<FormulaTemplate>) {
  const [versionHistoryOpen, setVersionHistoryOpen] = useState(false);
  const [forkDialogOpen, setForkDialogOpen] = useState(false);
  const [lineageDialogOpen, setLineageDialogOpen] = useState(false);

  const form = useForm<FormulaTemplateFormValues>({
    resolver: zodResolver(formulaTemplateSchema),
    defaultValues: {
      name: "",
      description: "",
      type: "FreightCharge",
      expression: "",
      status: "Draft",
      schemaId: "shipment",
      variableDefinitions: [],
    },
  });

  const handleForkSuccess = () => {
    setForkDialogOpen(false);
    onOpenChange(false);
  };

  const extraTabs = useMemo(
    () => [
      {
        value: "testing",
        label: "Testing",
        icon: FlaskConicalIcon,
        content: FormulaTemplateTestTab,
        contentProps: { form },
      },
      {
        value: "reference",
        label: "Reference",
        icon: BookOpenIcon,
        content: FormulaTemplateReferenceTab,
      },
    ],
    [form],
  );

  if (mode === "edit") {
    return (
      <>
        <TabbedFormEditPanel<FormulaTemplateFormValues, FormulaTemplate>
          open={open}
          onOpenChange={onOpenChange}
          row={row}
          form={form}
          url="/formula-templates/"
          queryKey="formula-template-list"
          title="Formula Template"
          fieldKey="name"
          formComponent={<FormulaTemplateForm />}
          tabs={extraTabs}
          size="xl"
          headerActions={
            row ? (
              <FormulaTemplateHeaderActions
                template={row}
                onVersionHistoryClick={() => setVersionHistoryOpen(true)}
                onForkClick={() => setForkDialogOpen(true)}
                onViewLineageClick={() => setLineageDialogOpen(true)}
              />
            ) : null
          }
        />

        <VersionHistoryPanel
          open={versionHistoryOpen}
          onOpenChange={setVersionHistoryOpen}
          template={row ?? null}
          onRollback={(updatedTemplate) => {
            form.reset(updatedTemplate as unknown as FormulaTemplateFormValues);
          }}
        />

        <ForkTemplateDialog
          open={forkDialogOpen}
          onOpenChange={setForkDialogOpen}
          template={row ?? null}
          onForkSuccess={handleForkSuccess}
        />

        <ForkLineageDialog
          open={lineageDialogOpen}
          onOpenChange={setLineageDialogOpen}
          templateId={row?.id}
          currentTemplateId={row?.id}
        />
      </>
    );
  }

  return (
    <FormCreatePanel<FormulaTemplateFormValues, FormulaTemplate>
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/formula-templates/"
      queryKey="formula-template-list"
      title="Formula Template"
      formComponent={<FormulaTemplateForm />}
      size="lg"
    />
  );
}
