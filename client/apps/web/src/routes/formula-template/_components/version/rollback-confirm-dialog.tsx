import { useT } from "@trenova/shared/i18n/use-t";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { Label } from "@trenova/shared/components/ui/label";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { cn } from "@trenova/shared/lib/utils";
import { queries } from "@/lib/queries";
import type {
  FormulaTemplate,
  FieldChange,
  TemplateUsageResponse,
} from "@trenova/shared/types/formula-template";
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
  templateStatus?: FormulaTemplate["status"];
  onConfirm: () => void;
  isLoading: boolean;
};

function formatUsageType(type: string): string {
  switch (type) {
    case "shipment":
      return "shipments";
    case "rate_matrix":
      return "rate matrices";
    case "rate_agreement_rule":
      return "rate agreement rules";
    case "rate_agreement_accessorial":
      return "rate agreement accessorials";
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

function getChangeBadgeVariant(type: string): BadgeVariant {
  switch (type) {
    case "created":
      return "success";
    case "deleted":
      return "danger";
    case "updated":
      return "info";
    default:
      return "neutral";
  }
}

type ChangeSummaryProps = {
  changes: Record<string, FieldChange>;
};

function ChangeSummary({ changes }: ChangeSummaryProps) {
  const t = useT();

  const [isExpanded, setIsExpanded] = useState(false);
  const changeEntries = Object.entries(changes);

  const categorizedChanges = {
    expression: changeEntries.filter(([k]) => k === "expression"),
    variables: changeEntries.filter(([k]) => k.startsWith("variableDefinitions")),
    status: changeEntries.filter(([k]) => k === "status"),
    other: changeEntries.filter(
      ([k]) => k !== "expression" && k !== "status" && !k.startsWith("variableDefinitions"),
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
      <div className="border-border bg-muted/30 rounded-md border p-3">
        <CollapsibleTrigger className="flex w-full items-center justify-between text-left">
          <div className="flex items-center gap-2">
            <span className="text-foreground text-sm font-medium">
              {t(
                "{0, plural, one {# change} other {# changes}} will be applied",
                changeEntries.length,
              )}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <div className="flex flex-wrap gap-1">
              {summaryParts.slice(0, 3).map((part) => (
                <Badge key={part} variant="neutral" className="text-2xs font-normal">
                  {part}
                </Badge>
              ))}
              {summaryParts.length > 3 && (
                <Badge variant="neutral" className="text-2xs font-normal">
                  +{summaryParts.length - 3}
                </Badge>
              )}
            </div>
            <ChevronDownIcon
              className={cn(
                "text-muted-foreground size-4 transition-transform",
                isExpanded && "rotate-180",
              )}
            />
          </div>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div className="border-border mt-3 max-h-48 space-y-1.5 overflow-y-auto border-t pt-3">
            {changeEntries.map(([path, change]) => (
              <div
                key={path}
                className="hover:bg-muted flex items-center justify-between gap-2 rounded-md px-2 py-1 text-xs"
              >
                <span className="text-foreground font-medium">{formatFieldName(path)}</span>
                <Badge variant={getChangeBadgeVariant(change.type)} className="text-2xs">
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
    <div className="border-border bg-muted/30 rounded-md border p-3">
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
  templateStatus,
  onConfirm,
  isLoading,
}: RollbackConfirmDialogProps) {
  const t = useT();

  const [confirmed, setConfirmed] = useState(false);

  const { data: diff, isLoading: isLoadingDiff } = useQuery({
    ...queries.formulaTemplate.versionDiff(templateId, targetVersion, currentVersion),
    enabled: open && !!templateId && currentVersion > 0 && targetVersion > 0,
  });

  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      setConfirmed(false);
    }
    onOpenChange(newOpen);
  };

  const totalUsageCount = usageData?.usages.reduce((sum, u) => sum + u.count, 0) ?? 0;

  return (
    <AlertDialog open={open} onOpenChange={handleOpenChange}>
      <AlertDialogContent className="max-w-lg">
        <AlertDialogHeader>
          <AlertDialogTitle>{t("Rollback to Version {0}", targetVersion)}</AlertDialogTitle>
          <AlertDialogDescription
            render={
              <div className="space-y-3">
                <span className="text-muted-foreground block text-sm">
                  {t(
                    "This will restore the template to version {0}, creating a new version (v{1}).",
                    targetVersion,
                    currentVersion + 1,
                  )}
                </span>

                {isLoadingDiff ? (
                  <ChangeSummarySkeleton />
                ) : diff && diff.changeCount > 0 ? (
                  <ChangeSummary changes={diff.changes} />
                ) : null}

                {usageData?.inUse && (
                  <Alert variant="warning" size="sm">
                    <AlertTriangleIcon />
                    <AlertDescription>
                      <p>
                        {t("This template is currently used by")} {totalUsageCount}{" "}
                        {usageData.usages.map((u, i) => (
                          <span key={u.type}>
                            {i > 0 && ", "}
                            {u.count} {formatUsageType(u.type)}
                          </span>
                        ))}
                        .{" "}
                        {templateStatus === "Active" || templateStatus === "InReview"
                          ? t(
                              "Rolling back to different content returns the template to Draft, and nothing rates with it until it is approved again.",
                            )
                          : t("Rolling back changes the content the next approval will review.")}
                      </p>
                    </AlertDescription>
                  </Alert>
                )}
                <Label htmlFor="rollback-confirm">
                  <Checkbox
                    id="rollback-confirm"
                    checked={confirmed}
                    onCheckedChange={(checked) => setConfirmed(checked === true)}
                  />
                  <span className="text-sm">
                    {t("I understand this will create a new version and cannot be undone")}
                  </span>
                </Label>
              </div>
            }
          />
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isLoading}>{t("Cancel")}</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={!confirmed || isLoading}
            onClick={onConfirm}
          >
            {isLoading && <Spinner />}
            {t("Rollback")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
