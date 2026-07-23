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
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { usePermission } from "@/hooks/use-permission";
import type { DataTablePanelProps } from "@/types/data-table";
import {
  formulaTemplateSchema,
  type FormulaTemplate,
  type FormulaTemplateFormValues,
} from "@/types/formula-template";
import { Operation, Resource } from "@/types/permission";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  BookOpenIcon,
  CheckIcon,
  ClockIcon,
  FlaskConicalIcon,
  GitBranchIcon,
  GitForkIcon,
  HistoryIcon,
  NetworkIcon,
  SendIcon,
  XIcon,
} from "lucide-react";
import { lazy, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { ApprovalActionDialog, type ApprovalAction } from "./approval-action-dialog";
import { ForkLineageDialog } from "./fork-lineage-dialog";
import { ForkTemplateDialog } from "./fork-template-dialog";
import { FormulaTemplateForm } from "./formula-template-form";
import { VersionHistoryPanel } from "./version/version-history-panel";

const FormulaTemplateTestTab = lazy(() => import("./formula-template-test-tab"));
const FormulaTemplateReferenceTab = lazy(() => import("./formula-template-reference-tab"));
const FormulaTemplateBacktestTab = lazy(() => import("./formula-template-backtest-tab"));

type FormulaTemplateHeaderActionsProps = {
  template: FormulaTemplate;
  onVersionHistoryClick: () => void;
  onForkClick: () => void;
  onViewLineageClick: () => void;
  onApprovalAction: (action: ApprovalAction) => void;
};

function FormulaTemplateHeaderActions({
  template,
  onVersionHistoryClick,
  onForkClick,
  onViewLineageClick,
  onApprovalAction,
}: FormulaTemplateHeaderActionsProps) {
  const { allowed: canSubmit } = usePermission(Resource.FormulaTemplate, Operation.Submit);
  const { allowed: canApprove } = usePermission(Resource.FormulaTemplate, Operation.Approve);
  const { allowed: canReject } = usePermission(Resource.FormulaTemplate, Operation.Reject);

  return (
    <div className="flex items-center gap-1">
      {template.status === "Draft" && canSubmit && (
        <Button
          type="button"
          variant="outline"
          size="xs"
          className="mr-1 gap-1.5"
          onClick={() => onApprovalAction("submit")}
        >
          <SendIcon className="size-3" />
          Submit for Review
        </Button>
      )}
      {template.status === "InReview" && (
        <div className="mr-1 flex items-center gap-1">
          {canApprove && (
            <Button
              type="button"
              variant="outline"
              size="xs"
              className="gap-1.5 text-emerald-600 dark:text-emerald-400"
              onClick={() => onApprovalAction("approve")}
            >
              <CheckIcon className="size-3" />
              Approve
            </Button>
          )}
          {canReject && (
            <Button
              type="button"
              variant="outline"
              size="xs"
              className="gap-1.5 text-destructive"
              onClick={() => onApprovalAction("reject")}
            >
              <XIcon className="size-3" />
              Reject
            </Button>
          )}
        </div>
      )}
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
  const [approvalAction, setApprovalAction] = useState<ApprovalAction | null>(null);

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
      breakdownDefinitions: [],
      minCharge: null,
      maxCharge: null,
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
        value: "backtest",
        label: "Backtest",
        icon: HistoryIcon,
        content: FormulaTemplateBacktestTab,
        contentProps: { form, template: row ?? null },
      },
      {
        value: "reference",
        label: "Reference",
        icon: BookOpenIcon,
        content: FormulaTemplateReferenceTab,
      },
    ],
    [form, row],
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
                onApprovalAction={setApprovalAction}
              />
            ) : null
          }
        />

        {approvalAction && (
          <ApprovalActionDialog
            open={approvalAction !== null}
            onOpenChange={(open) => {
              if (!open) setApprovalAction(null);
            }}
            action={approvalAction}
            template={row ?? null}
          />
        )}

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
