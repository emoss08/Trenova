import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { formulaTemplate } from "@/lib/queries/formula-template";
import { apiService } from "@/services/api";
import type { ForkLineage, FormulaTemplate } from "@/types/formula-template";
import { useQuery } from "@tanstack/react-query";
import { AlertCircleIcon, GitBranchIcon, Loader2Icon } from "lucide-react";

type ForkLineageDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  templateId?: FormulaTemplate["id"];
  currentTemplateId?: FormulaTemplate["id"];
  onNavigateToTemplate?: (templateId: FormulaTemplate["id"]) => void;
};

type LineageNodeProps = {
  node: ForkLineage;
  currentTemplateId?: FormulaTemplate["id"];
  onNavigateToTemplate?: (templateId: FormulaTemplate["id"]) => void;
  isRoot?: boolean;
};

function LineageNode({
  node,
  currentTemplateId,
  onNavigateToTemplate,
  isRoot = false,
}: LineageNodeProps) {
  const isCurrent = node.templateId === currentTemplateId;
  const hasChildren = node.forkedTemplates && node.forkedTemplates.length > 0;

  return (
    <div className="relative">
      <div className="flex items-start gap-2">
        {!isRoot && (
          <div className="flex flex-col items-center pt-2">
            <div className="w-4 border-t border-border" />
          </div>
        )}
        <div
          className={`flex items-center gap-2 rounded-md border px-3 py-2 ${
            isCurrent
              ? "border-primary bg-primary/5"
              : "border-border hover:border-muted-foreground/50"
          } ${onNavigateToTemplate && !isCurrent ? "cursor-pointer" : ""}`}
          onClick={() => {
            if (onNavigateToTemplate && !isCurrent) {
              onNavigateToTemplate(node.templateId);
            }
          }}
        >
          <GitBranchIcon className="size-4 text-muted-foreground" />
          <div className="flex flex-col gap-0.5">
            <span className="text-sm font-medium">{node.templateName}</span>
            {node.sourceVersion && (
              <span className="text-xs text-muted-foreground">
                Forked from v{node.sourceVersion}
              </span>
            )}
          </div>
          {isCurrent && (
            <Badge variant="outline" className="ml-2 text-xs">
              Current
            </Badge>
          )}
        </div>
      </div>

      {hasChildren && (
        <div className="mt-2 ml-6 border-l border-border pl-4">
          {node.forkedTemplates!.map((child, index) => (
            <div
              key={child.templateId}
              className={index > 0 ? "mt-2" : undefined}
            >
              <LineageNode
                node={child}
                currentTemplateId={currentTemplateId}
                onNavigateToTemplate={onNavigateToTemplate}
              />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export function ForkLineageDialog({
  open,
  onOpenChange,
  templateId,
  currentTemplateId,
  onNavigateToTemplate,
}: ForkLineageDialogProps) {
  const {
    data: lineage,
    isLoading,
    error,
  } = useQuery({
    queryKey: formulaTemplate.lineage(templateId).queryKey,
    queryFn: () =>
      templateId
        ? apiService.formulaTemplateService.getLineage(templateId)
        : Promise.reject(new Error("No template ID")),
    enabled: open && !!templateId,
  });

  const handleClose = () => {
    onOpenChange(false);
  };

  const handleNavigate = (targetTemplateId: FormulaTemplate["id"]) => {
    onNavigateToTemplate?.(targetTemplateId);
    handleClose();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <GitBranchIcon className="size-4" />
            Fork Lineage
          </DialogTitle>
          <DialogDescription>
            View the fork history and ancestry of this template. Click on a
            template to navigate to it.
          </DialogDescription>
        </DialogHeader>

        <div className="max-h-[400px] overflow-y-auto py-4">
          {isLoading && (
            <div className="flex items-center justify-center py-8">
              <Loader2Icon className="size-6 animate-spin text-muted-foreground" />
            </div>
          )}

          {error && (
            <div className="flex flex-col items-center justify-center gap-2 py-8 text-muted-foreground">
              <AlertCircleIcon className="size-8" />
              <p className="text-sm">Failed to load lineage data</p>
            </div>
          )}

          {lineage && (
            <LineageNode
              node={lineage}
              currentTemplateId={currentTemplateId || templateId || undefined}
              onNavigateToTemplate={handleNavigate}
              isRoot
            />
          )}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
