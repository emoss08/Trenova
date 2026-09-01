import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import { formulaTemplateStatusChoices } from "@/lib/choices";
import { ColorOptionValue } from "@/components/fields/select-components";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@trenova/shared/components/ui/hover-card";
import type { FormulaTemplate } from "@trenova/shared/types/formula-template";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeftIcon,
  CheckIcon,
  ClockIcon,
  DownloadIcon,
  FileUpIcon,
  GitBranchIcon,
  GitForkIcon,
  HistoryIcon,
  MoreVerticalIcon,
  NetworkIcon,
  SendIcon,
  UsersIcon,
  XIcon,
} from "lucide-react";
import { useNavigate } from "react-router";
import type { ApprovalAction } from "../approval-action-dialog";

const USAGE_LABELS: Record<string, string> = {
  shipment: "shipments",
  accessorial_charge: "accessorial charges",
  rate_agreement_rule: "rate agreement rules",
  rate_agreement_accessorial: "rate agreement accessorials",
};

function UsageChip({ templateId }: { templateId: string }) {
  const { data } = useQuery({
    ...queries.formulaTemplate.usage(templateId),
    staleTime: 60_000,
  });

  if (!data) return null;

  const total = data.usages.reduce((sum, usage) => sum + usage.count, 0);

  return (
    <HoverCard>
      <HoverCardTrigger
        render={
          <Badge variant={data.inUse ? "info" : "outline"} className="gap-1 text-xs">
            <UsersIcon className="size-3" />
            {data.inUse ? `In use (${total})` : "Not in use"}
          </Badge>
        }
      />
      <HoverCardContent side="bottom" className="w-64 space-y-1.5">
        <p className="text-sm font-semibold">Where this template is used</p>
        {data.usages.length === 0 ? (
          <p className="text-muted-foreground text-xs">
            Nothing references this template yet. Editing it is safe.
          </p>
        ) : (
          <div className="space-y-1">
            {data.usages.map((usage) => (
              <div key={usage.type} className="flex items-center justify-between text-xs">
                <span className="capitalize">{USAGE_LABELS[usage.type] ?? usage.type}</span>
                <span className="font-mono font-medium">{usage.count}</span>
              </div>
            ))}
            <p className="text-muted-foreground text-2xs pt-1">
              Changes take effect the next time these are rated.
            </p>
          </div>
        )}
      </HoverCardContent>
    </HoverCard>
  );
}

type StudioHeaderProps = {
  mode: "create" | "edit";
  template: FormulaTemplate | null;
  templateName: string;
  isSubmitting: boolean;
  isDirty: boolean;
  onSave: () => void;
  onApprovalAction: (action: ApprovalAction) => void;
  onVersionHistory: () => void;
  onFork: () => void;
  onLineage: () => void;
  onExport: () => void;
  onImport: () => void;
  onBacktest: () => void;
};

export function StudioHeader({
  mode,
  template,
  templateName,
  isSubmitting,
  isDirty,
  onSave,
  onApprovalAction,
  onVersionHistory,
  onFork,
  onLineage,
  onExport,
  onImport,
  onBacktest,
}: StudioHeaderProps) {
  const navigate = useNavigate();
  const { allowed: canSubmit } = usePermission(Resource.FormulaTemplate, Operation.Submit);
  const { allowed: canApprove } = usePermission(Resource.FormulaTemplate, Operation.Approve);
  const { allowed: canReject } = usePermission(Resource.FormulaTemplate, Operation.Reject);

  const statusChoice = template
    ? formulaTemplateStatusChoices.find((choice) => choice.value === template.status)
    : null;

  return (
    <div className="flex items-center justify-between gap-3 border-b px-4 py-2.5">
      <div className="flex min-w-0 items-center gap-3">
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          onClick={() => void navigate("/billing/configuration-files/formula-templates")}
        >
          <ArrowLeftIcon className="size-4" />
        </Button>
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <h1 className="truncate text-sm font-semibold">
              {mode === "create" ? "New Formula Template" : templateName || "Formula Template"}
            </h1>
            {statusChoice && (
              <ColorOptionValue color={statusChoice.color} value={statusChoice.label} />
            )}
            {template?.currentVersionNumber != null && (
              <Badge variant="outline" className="font-mono text-xs">
                v{template.currentVersionNumber}
              </Badge>
            )}
            {template?.sourceTemplateId && (
              <button
                type="button"
                onClick={onLineage}
                className="flex items-center gap-1 rounded-sm border border-amber-500/50 bg-amber-500/15 px-1.5 py-0.5 text-xs text-amber-600 dark:text-amber-400"
              >
                <GitBranchIcon className="size-3" />
                Forked from v{template.sourceVersionNumber}
              </button>
            )}
          </div>
        </div>
        {mode === "edit" && template?.id && <UsageChip templateId={template.id} />}
      </div>

      <div className="flex shrink-0 items-center gap-1.5">
        {mode === "edit" &&
          (template?.status === "Draft" || template?.status === "Inactive") &&
          canSubmit && (
            <Button
              type="button"
              variant="outline"
              size="xs"
              className="gap-1.5"
              onClick={() => onApprovalAction("submit")}
            >
              <SendIcon className="size-3" />
              {template?.status === "Inactive" ? "Reactivate via Review" : "Submit for Review"}
            </Button>
          )}
        {mode === "edit" && template?.status === "InReview" && (
          <>
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
                className="text-destructive gap-1.5"
                onClick={() => onApprovalAction("reject")}
              >
                <XIcon className="size-3" />
                Reject
              </Button>
            )}
          </>
        )}

        {mode === "edit" && (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button type="button" variant="ghost" size="icon-sm">
                  <MoreVerticalIcon className="size-4" />
                </Button>
              }
            />
            <DropdownMenuContent align="end" className="min-w-[200px]">
              <DropdownMenuGroup>
                <DropdownMenuItem
                  title="Version History"
                  startContent={<ClockIcon className="size-4" />}
                  onClick={onVersionHistory}
                />
                <DropdownMenuItem
                  title="Backtest"
                  description="Re-rate recent shipments"
                  startContent={<HistoryIcon className="size-4" />}
                  onClick={onBacktest}
                />
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
              <DropdownMenuGroup>
                <DropdownMenuItem
                  title="Fork Template"
                  startContent={<GitForkIcon className="size-4" />}
                  onClick={onFork}
                />
                <DropdownMenuItem
                  title="View Lineage"
                  startContent={<NetworkIcon className="size-4" />}
                  onClick={onLineage}
                />
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
              <DropdownMenuGroup>
                <DropdownMenuItem
                  title="Export JSON"
                  startContent={<DownloadIcon className="size-4" />}
                  onClick={onExport}
                />
                <DropdownMenuItem
                  title="Import Templates"
                  startContent={<FileUpIcon className="size-4" />}
                  onClick={onImport}
                />
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        )}

        <Button
          type="button"
          size="sm"
          onClick={onSave}
          isLoading={isSubmitting}
          loadingText="Saving..."
          disabled={mode === "edit" && !isDirty}
        >
          {mode === "create" ? "Create Template" : "Save Changes"}
        </Button>
      </div>
    </div>
  );
}
