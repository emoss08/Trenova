import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import type {
  FieldChange,
  TemplateUsageResponse,
} from "@/types/formula-template";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangleIcon, ChevronDownIcon } from "lucide-react";
import { useState } from "react";

type RollbackConfirmDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  templateId: string;
  currentVersion: number;
  targetVersion: number;
  usageData?: TemplateUsageResponse | null;
  onConfirm: () => void;
  isLoading: boolean;
};

function formatUsageType(type: string): string {
  switch (type) {
    case "shipment":
      return "shipments";
    case "accessorial_charge":
      return "accessorial charges";
    default:
      return type;
  }
}

function formatFieldName(path: string): string {
  const parts = path.split(".");
  const lastPart = parts[parts.length - 1];
  return lastPart
    .replace(/([A-Z])/g, " $1")
    .replace(/^./, (str) => str.toUpperCase())
    .trim();
}

function getChangeBadgeStyle(type: string) {
  switch (type) {
    case "created":
      return "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400 uppercase";
    case "deleted":
      return "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400 uppercase";
    case "updated":
      return "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400 uppercase";
    default:
      return "bg-muted text-muted-foreground uppercase";
  }
}

type ChangeSummaryProps = {
  changes: Record<string, FieldChange>;
};

function ChangeSummary({ changes }: ChangeSummaryProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const changeEntries = Object.entries(changes);

  const categorizedChanges = {
    expression: changeEntries.filter(([k]) => k === "expression"),
    variables: changeEntries.filter(([k]) =>
      k.startsWith("variableDefinitions"),
    ),
    status: changeEntries.filter(([k]) => k === "status"),
    other: changeEntries.filter(
      ([k]) =>
        k !== "expression" &&
        k !== "status" &&
        !k.startsWith("variableDefinitions"),
    ),
  };

  const summaryParts: string[] = [];
  if (categorizedChanges.expression.length > 0) summaryParts.push("Expression");
  if (categorizedChanges.variables.length > 0)
    summaryParts.push(
      `${categorizedChanges.variables.length} Variable${categorizedChanges.variables.length > 1 ? "s" : ""}`,
    );
  if (categorizedChanges.status.length > 0) summaryParts.push("Status");
  if (categorizedChanges.other.length > 0)
    summaryParts.push(`${categorizedChanges.other.length} other`);

  return (
    <Collapsible open={isExpanded} onOpenChange={setIsExpanded}>
      <div className="rounded-md border border-border bg-muted/30 p-3">
        <CollapsibleTrigger className="flex w-full items-center justify-between text-left">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-foreground">
              {changeEntries.length} change
              {changeEntries.length !== 1 ? "s" : ""} will be applied
            </span>
          </div>
          <div className="flex items-center gap-2">
            <div className="flex flex-wrap gap-1">
              {summaryParts.slice(0, 3).map((part) => (
                <Badge
                  key={part}
                  variant="secondary"
                  className="text-[10px] font-normal"
                >
                  {part}
                </Badge>
              ))}
              {summaryParts.length > 3 && (
                <Badge variant="secondary" className="text-[10px] font-normal">
                  +{summaryParts.length - 3}
                </Badge>
              )}
            </div>
            <ChevronDownIcon
              className={cn(
                "size-4 text-muted-foreground transition-transform",
                isExpanded && "rotate-180",
              )}
            />
          </div>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div className="mt-3 max-h-48 space-y-1.5 overflow-y-auto border-t border-border pt-3">
            {changeEntries.map(([path, change]) => (
              <div
                key={path}
                className="flex items-center justify-between gap-2 rounded px-2 py-1 text-xs hover:bg-muted"
              >
                <span className="font-medium text-foreground">
                  {formatFieldName(path)}
                </span>
                <Badge
                  className={cn(
                    "text-[10px]",
                    getChangeBadgeStyle(change.type),
                  )}
                >
                  {change.type}
                </Badge>
              </div>
            ))}
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}

function ChangeSummarySkeleton() {
  return (
    <div className="rounded-md border border-border bg-muted/30 p-3">
      <div className="flex items-center gap-2">
        <Skeleton className="size-4" />
        <Skeleton className="h-4 w-32" />
        <div className="ml-auto flex gap-1">
          <Skeleton className="h-5 w-16" />
          <Skeleton className="h-5 w-12" />
        </div>
      </div>
    </div>
  );
}

export function RollbackConfirmDialog({
  open,
  onOpenChange,
  templateId,
  currentVersion,
  targetVersion,
  usageData,
  onConfirm,
  isLoading,
}: RollbackConfirmDialogProps) {
  const [confirmed, setConfirmed] = useState(false);

  const { data: diff, isLoading: isLoadingDiff } = useQuery({
    queryKey: [
      "formulaTemplate",
      "compare",
      templateId,
      targetVersion,
      currentVersion,
    ],
    queryFn: () =>
      apiService.formulaTemplateService.compareVersions(
        templateId,
        targetVersion,
        currentVersion,
      ),
    enabled: open && !!templateId && currentVersion > 0 && targetVersion > 0,
  });

  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      setConfirmed(false);
    }
    onOpenChange(newOpen);
  };

  const totalUsageCount =
    usageData?.usages.reduce((sum, u) => sum + u.count, 0) ?? 0;

  return (
    <AlertDialog open={open} onOpenChange={handleOpenChange}>
      <AlertDialogContent className="max-w-lg">
        <AlertDialogHeader>
          <AlertDialogTitle>
            Rollback to Version {targetVersion}
          </AlertDialogTitle>
          <AlertDialogDescription
            render={
              <div className="space-y-3">
                <span className="block text-sm text-muted-foreground">
                  This will restore the template to version {targetVersion},
                  creating a new version (v{currentVersion + 1}).
                </span>

                {isLoadingDiff ? (
                  <ChangeSummarySkeleton />
                ) : diff && diff.changeCount > 0 ? (
                  <ChangeSummary changes={diff.changes} />
                ) : null}

                {usageData?.inUse && (
                  <div className="flex items-start gap-2 rounded-md border border-amber-500/50 bg-amber-500/10 p-2 text-sm text-amber-600 dark:text-amber-400">
                    <AlertTriangleIcon className="mt-0.5 size-4 shrink-0" />
                    <span>
                      This template is currently used by {totalUsageCount}{" "}
                      {usageData.usages.map((u, i) => (
                        <span key={u.type}>
                          {i > 0 && ", "}
                          {u.count} {formatUsageType(u.type)}
                        </span>
                      ))}
                      . Rolling back may affect active calculations.
                    </span>
                  </div>
                )}
                <Label htmlFor="rollback-confirm">
                  <Checkbox
                    id="rollback-confirm"
                    checked={confirmed}
                    onCheckedChange={(checked) =>
                      setConfirmed(checked === true)
                    }
                  />
                  <span className="text-sm">
                    I understand this will create a new version and cannot be
                    undone
                  </span>
                </Label>
              </div>
            }
          />
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isLoading}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={!confirmed || isLoading}
            onClick={onConfirm}
          >
            {isLoading && <Spinner />}
            Rollback
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
