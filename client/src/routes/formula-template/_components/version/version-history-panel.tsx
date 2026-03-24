import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { UserHoverCard } from "@/components/user-hover-card";
import { formatToUserTimezone } from "@/lib/date";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import {
  VERSION_TAG_OPTIONS,
  type FieldChange,
  type FormulaTemplate,
  type FormulaTemplateVersion,
  type TemplateUsageResponse,
  type VersionTag,
} from "@/types/formula-template";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { formatDistanceToNow } from "date-fns";
import {
  AlertCircleIcon,
  CheckIcon,
  ClockIcon,
  DotIcon,
  DownloadIcon,
  GitBranchIcon,
  GitCompare,
  GitCompareArrowsIcon,
  MoreVertical,
  RotateCcw,
  TagIcon,
  XIcon,
} from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";
import { RollbackConfirmDialog } from "./rollback-confirm-dialog";
import { VersionCompareDialog } from "./version-compare-dialog";

type VersionHistoryPanelProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  template: FormulaTemplate | null;
  onRollback?: (template: FormulaTemplate) => void;
};

export function VersionHistoryPanel({
  open,
  onOpenChange,
  template,
  onRollback,
}: VersionHistoryPanelProps) {
  const queryClient = useQueryClient();
  const [compareDialogOpen, setCompareDialogOpen] = useState(false);
  const [compareVersions, setCompareVersions] = useState<{
    from: number;
    to: number;
  } | null>(null);
  const [rollingBackVersion, setRollingBackVersion] = useState<number | null>(
    null,
  );
  const [rollbackDialogOpen, setRollbackDialogOpen] = useState(false);
  const [pendingRollbackVersion, setPendingRollbackVersion] =
    useState<FormulaTemplateVersion | null>(null);
  const [usageData, setUsageData] = useState<TemplateUsageResponse | null>(
    null,
  );
  const [isLoadingUsage, setIsLoadingUsage] = useState(false);
  const [compareMode, setCompareMode] = useState(false);
  const [selectedForCompare, setSelectedForCompare] = useState<number | null>(
    null,
  );

  const { data, isLoading, error } = useQuery({
    ...queries.formulaTemplate.versions(template?.id),
    enabled: open && !!template?.id,
  });

  const versions = data?.results ?? [];

  const handleCompare = (fromVersion: number, toVersion: number) => {
    const [from, to] =
      fromVersion < toVersion
        ? [fromVersion, toVersion]
        : [toVersion, fromVersion];
    setCompareVersions({ from, to });
    setCompareDialogOpen(true);
    setCompareMode(false);
    setSelectedForCompare(null);
  };

  const handleSelectForCompare = (versionNumber: number) => {
    setCompareMode(true);
    setSelectedForCompare(versionNumber);
  };

  const handleCancelCompareMode = () => {
    setCompareMode(false);
    setSelectedForCompare(null);
  };

  const handleExportVersion = useCallback(
    (version: FormulaTemplateVersion) => {
      const exportData = {
        exportedAt: new Date().toISOString(),
        templateId: template?.id,
        templateName: template?.name,
        version: {
          versionNumber: version.versionNumber,
          name: version.name,
          description: version.description,
          type: version.type,
          expression: version.expression,
          status: version.status,
          schemaId: version.schemaId,
          variableDefinitions: version.variableDefinitions,
          metadata: version.metadata,
          changeMessage: version.changeMessage,
          createdAt: version.createdAt,
        },
      };

      const blob = new Blob([JSON.stringify(exportData, null, 2)], {
        type: "application/json",
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${template?.name?.replace(/\s+/g, "-").toLowerCase()}-v${version.versionNumber}.json`;
      a.click();
      URL.revokeObjectURL(url);
      toast.success("Version exported", {
        description: `Downloaded v${version.versionNumber} as JSON`,
      });
    },
    [template],
  );

  const handleRollbackClick = async (version: FormulaTemplateVersion) => {
    if (!template?.id) return;

    setPendingRollbackVersion(version);
    setIsLoadingUsage(true);
    setRollbackDialogOpen(true);

    await apiService.formulaTemplateService
      .getUsage(template.id)
      .then((usage) => {
        setUsageData(usage);
      })
      .catch(() => {
        setUsageData(null);
      })
      .finally(() => {
        setIsLoadingUsage(false);
      });
  };

  const handleRollbackConfirm = async () => {
    if (!template?.id || !pendingRollbackVersion) return;

    setRollingBackVersion(pendingRollbackVersion.versionNumber);
    await apiService.formulaTemplateService
      .rollback(template.id, {
        targetVersion: pendingRollbackVersion.versionNumber,
        changeMessage: `Rolled back to version ${pendingRollbackVersion.versionNumber}`,
      })
      .then((updatedTemplate) => {
        toast.success("Rollback successful", {
          description: `Restored to version ${pendingRollbackVersion.versionNumber}`,
        });

        void queryClient.invalidateQueries({ queryKey: ["formulaTemplate"] });
        setRollbackDialogOpen(false);
        setPendingRollbackVersion(null);
        setUsageData(null);
        onRollback?.(updatedTemplate);
      })
      .catch((err) => {
        console.error("Rollback failed:", err);
        toast.error("Rollback failed", {
          description: "Could not restore to the selected version",
        });
      })
      .finally(() => {
        setRollingBackVersion(null);
      });
  };

  return (
    <>
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent side="right" className="w-[500px] gap-0 sm:max-w-[500px]">
          <SheetHeader className="border-b border-border pb-2">
            <SheetTitle className="flex items-center gap-2">
              <ClockIcon className="size-5" />
              Version History
            </SheetTitle>
            <SheetDescription>
              {template?.name} - {versions.length} version(s)
            </SheetDescription>
          </SheetHeader>

          <div className="flex h-[calc(100%-80px)] flex-col">
            {compareMode && selectedForCompare !== null && (
              <div className="m-2 flex items-center justify-between gap-2 rounded-md border border-primary/50 bg-primary/10 px-3 py-2 text-sm">
                <div className="flex items-center gap-2">
                  <GitCompareArrowsIcon className="size-4 text-primary" />
                  <span className="text-foreground">
                    Select another version to compare with{" "}
                    <span className="font-mono font-semibold">
                      v{selectedForCompare}
                    </span>
                  </span>
                </div>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  onClick={handleCancelCompareMode}
                >
                  <XIcon className="size-4" />
                </Button>
              </div>
            )}

            {template?.sourceTemplateId && (
              <div className="m-2 flex items-center gap-2 rounded-sm border border-amber-500 bg-amber-500/20 p-1 text-sm text-amber-500">
                <GitBranchIcon className="size-4" />
                <p>Forked from version {template.sourceVersionNumber}</p>
              </div>
            )}

            <ScrollArea className="flex max-h-[calc(100vh-5rem)] flex-col p-2 transition-all hover:pr-3">
              {isLoading ? (
                <VersionListSkeleton />
              ) : error ? (
                <div className="flex flex-col items-center justify-center py-12 text-center">
                  <AlertCircleIcon className="mb-4 size-12 text-destructive" />
                  <p className="text-muted-foreground">
                    Failed to load version history
                  </p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    Please try again later
                  </p>
                </div>
              ) : versions.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-12 text-center">
                  <ClockIcon className="mb-4 size-12 text-muted-foreground" />
                  <p className="text-muted-foreground">
                    No version history yet
                  </p>
                </div>
              ) : (
                <div className="space-y-2">
                  {versions.map((version, index) => (
                    <VersionItem
                      key={version.id}
                      templateId={template?.id ?? ""}
                      version={version}
                      isCurrent={
                        version.versionNumber === template?.currentVersionNumber
                      }
                      onComparePrevious={
                        index < versions.length - 1
                          ? () =>
                              handleCompare(
                                versions[index + 1].versionNumber,
                                version.versionNumber,
                              )
                          : undefined
                      }
                      onRollback={
                        index !== 0
                          ? () => void handleRollbackClick(version)
                          : undefined
                      }
                      isRollingBack={
                        rollingBackVersion === version.versionNumber
                      }
                      onExport={() => handleExportVersion(version)}
                      compareMode={compareMode}
                      selectedForCompare={selectedForCompare}
                      onSelectForCompare={() =>
                        handleSelectForCompare(version.versionNumber)
                      }
                      onCompareWith={() => {
                        if (selectedForCompare !== null) {
                          handleCompare(
                            selectedForCompare,
                            version.versionNumber,
                          );
                        }
                      }}
                    />
                  ))}
                </div>
              )}
            </ScrollArea>
          </div>
        </SheetContent>
      </Sheet>

      {template?.id && compareVersions && (
        <VersionCompareDialog
          open={compareDialogOpen}
          onOpenChange={setCompareDialogOpen}
          templateId={template.id}
          fromVersion={compareVersions.from}
          toVersion={compareVersions.to}
        />
      )}

      {template && pendingRollbackVersion && (
        <RollbackConfirmDialog
          open={rollbackDialogOpen}
          onOpenChange={(open) => {
            setRollbackDialogOpen(open);
            if (!open) {
              setPendingRollbackVersion(null);
              setUsageData(null);
            }
          }}
          templateId={template.id ?? ""}
          currentVersion={template.currentVersionNumber ?? 0}
          targetVersion={pendingRollbackVersion.versionNumber}
          usageData={usageData}
          onConfirm={handleRollbackConfirm}
          isLoading={rollingBackVersion !== null || isLoadingUsage}
        />
      )}
    </>
  );
}

type VersionItemProps = {
  templateId: string;
  version: FormulaTemplateVersion;
  isCurrent: boolean;
  onComparePrevious?: () => void;
  onRollback?: () => void;
  isRollingBack: boolean;
  onExport: () => void;
  compareMode: boolean;
  selectedForCompare: number | null;
  onSelectForCompare: () => void;
  onCompareWith: () => void;
};

type ChangeBadgeInfo = {
  label: string;
  color: string;
  tooltip: string;
};

function getChangeBadges(
  changeSummary?: Record<string, FieldChange> | null,
): ChangeBadgeInfo[] | null {
  if (!changeSummary || Object.keys(changeSummary).length === 0) return null;

  const changes = Object.entries(changeSummary);
  const badges: ChangeBadgeInfo[] = [];

  const hasExpression = changes.some(([k]) => k === "expression");
  const variableChanges = changes.filter(([k]) =>
    k.startsWith("variableDefinitions"),
  );
  const metadataChanges = changes.filter(([k]) => k.startsWith("metadata"));
  const hasStatus = changes.some(([k]) => k === "status");
  const hasName = changes.some(([k]) => k === "name");
  const hasDescription = changes.some(([k]) => k === "description");
  const hasType = changes.some(([k]) => k === "type");

  if (hasExpression) {
    badges.push({
      label: "Expr",
      color: "bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300",
      tooltip: "Expression changed",
    });
  }

  if (variableChanges.length > 0) {
    badges.push({
      label: `Vars${variableChanges.length > 1 ? ` (${variableChanges.length})` : ""}`,
      color:
        "bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300",
      tooltip: `${variableChanges.length} variable change${variableChanges.length > 1 ? "s" : ""}`,
    });
  }

  if (hasStatus) {
    badges.push({
      label: "Status",
      color:
        "bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300",
      tooltip: "Status changed",
    });
  }

  const otherCount =
    (hasName ? 1 : 0) +
    (hasDescription ? 1 : 0) +
    (hasType ? 1 : 0) +
    metadataChanges.length;

  if (otherCount > 0) {
    const otherLabels: string[] = [];
    if (hasName) otherLabels.push("name");
    if (hasDescription) otherLabels.push("description");
    if (hasType) otherLabels.push("type");
    if (metadataChanges.length > 0)
      otherLabels.push(`${metadataChanges.length} metadata`);

    badges.push({
      label: `+${otherCount}`,
      color: "bg-muted text-muted-foreground",
      tooltip: otherLabels.join(", "),
    });
  }

  return badges.length > 0 ? badges : null;
}

function VersionItem({
  templateId,
  version,
  isCurrent,
  onComparePrevious,
  onRollback,
  isRollingBack,
  onExport,
  compareMode,
  selectedForCompare,
  onSelectForCompare,
  onCompareWith,
}: VersionItemProps) {
  const queryClient = useQueryClient();
  const [tagsDialogOpen, setTagsDialogOpen] = useState(false);
  const [selectedTags, setSelectedTags] = useState<VersionTag[]>(
    version.tags ?? [],
  );

  const updateTagsMutation = useMutation({
    mutationFn: (tags: VersionTag[]) =>
      apiService.formulaTemplateService.updateVersionTags(
        templateId,
        version.versionNumber,
        tags,
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["formulaTemplate"] });
      toast.success("Tags updated");
      setTagsDialogOpen(false);
    },
    onError: () => {
      toast.error("Failed to update tags");
    },
  });

  const handleTagToggle = (tag: VersionTag) => {
    setSelectedTags((prev) =>
      prev.includes(tag) ? prev.filter((t) => t !== tag) : [...prev, tag],
    );
  };

  const handleSaveTags = () => {
    updateTagsMutation.mutate(selectedTags);
  };

  const handleOpenTagsDialog = () => {
    setSelectedTags(version.tags ?? []);
    setTagsDialogOpen(true);
  };

  const isSelectedForCompare = selectedForCompare === version.versionNumber;
  const canCompareWith =
    compareMode &&
    selectedForCompare !== null &&
    selectedForCompare !== version.versionNumber;
  const changeBadges = getChangeBadges(version.changeSummary);
  const currentTags = version.tags ?? [];

  return (
    <>
      <div
        className={cn(
          "group relative rounded-lg border bg-card p-3 transition-all hover:bg-sidebar",
          isCurrent && "border-primary/50 bg-primary/5",
          isSelectedForCompare && "ring-2 ring-primary ring-offset-2",
          canCompareWith && "cursor-pointer hover:border-primary/50",
        )}
        onClick={canCompareWith ? onCompareWith : undefined}
      >
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0 flex-1">
            <div className="mb-1 flex flex-wrap items-center gap-2">
              <span className="font-mono text-sm font-medium">
                v{version.versionNumber}
              </span>
              {isCurrent && (
                <Badge variant="active" className="text-xs">
                  Current
                </Badge>
              )}
              {isSelectedForCompare && (
                <Badge
                  variant="outline"
                  className="border-primary text-xs text-primary"
                >
                  Selected
                </Badge>
              )}
              {currentTags.length > 0 && (
                <div className="flex flex-wrap gap-1">
                  {currentTags.map((tag) => {
                    const tagOption = VERSION_TAG_OPTIONS.find(
                      (t) => t.value === tag,
                    );
                    return (
                      <Tooltip key={tag}>
                        <TooltipTrigger
                          render={
                            <span
                              className={cn(
                                "inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-medium",
                                tagOption?.color ??
                                  "bg-muted text-muted-foreground",
                              )}
                            >
                              <TagIcon className="size-2.5" />
                              {tag}
                            </span>
                          }
                        />
                        <TooltipContent side="top" className="text-xs">
                          {tagOption?.description ?? tag}
                        </TooltipContent>
                      </Tooltip>
                    );
                  })}
                </div>
              )}
            </div>

            {changeBadges && changeBadges.length > 0 && (
              <div className="mb-2 flex flex-wrap gap-1">
                {changeBadges.map((badge) => (
                  <Tooltip key={badge.label}>
                    <TooltipTrigger
                      render={
                        <span
                          className={cn(
                            "inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-medium",
                            badge.color,
                          )}
                        >
                          {badge.label}
                        </span>
                      }
                    />
                    <TooltipContent side="top" className="text-xs">
                      {badge.tooltip}
                    </TooltipContent>
                  </Tooltip>
                ))}
              </div>
            )}

            {version.changeMessage && (
              <p className="mb-2 line-clamp-2 text-sm text-muted-foreground">
                {version.changeMessage}
              </p>
            )}

            <div className="flex items-center text-xs text-muted-foreground">
              <span
                className="cursor-help"
                title={formatToUserTimezone(version.createdAt, {
                  showSeconds: true,
                })}
              >
                {formatDistanceToNow(version.createdAt * 1000, {
                  addSuffix: true,
                })}
              </span>
              <DotIcon className="size-3" />
              <span className="flex items-center gap-1">
                by{" "}
                <UserHoverCard
                  userId={version.createdBy?.id}
                  username={version.createdBy?.username || ""}
                />
              </span>
            </div>
          </div>

          <div
            className={cn(
              "flex items-center gap-1 transition-opacity",
              canCompareWith
                ? "opacity-100"
                : "opacity-0 group-hover:opacity-100",
            )}
          >
            {canCompareWith ? (
              <Button variant="outline" size="xs" onClick={onCompareWith}>
                <GitCompare className="mr-1 size-3" />
                Compare
              </Button>
            ) : (
              <DropdownMenu>
                <DropdownMenuTrigger
                  render={
                    <Button variant="ghost" size="icon-xs">
                      <MoreVertical className="size-4" />
                    </Button>
                  }
                />
                <DropdownMenuContent align="end" className="min-w-[180px]">
                  <DropdownMenuGroup>
                    <DropdownMenuItem
                      startContent={<TagIcon className="size-4" />}
                      title="Manage Tags"
                      description="Add or remove version labels"
                      onClick={handleOpenTagsDialog}
                    />
                  </DropdownMenuGroup>
                  <DropdownMenuSeparator />
                  <DropdownMenuGroup>
                    {compareMode && isSelectedForCompare ? null : (
                      <DropdownMenuItem
                        startContent={
                          <GitCompareArrowsIcon className="size-4" />
                        }
                        title="Select for Compare"
                        description="Compare with any version"
                        onClick={(e) => {
                          e.stopPropagation();
                          onSelectForCompare();
                        }}
                      />
                    )}
                    {onComparePrevious && (
                      <DropdownMenuItem
                        startContent={<GitCompare className="size-4" />}
                        title="Compare Previous"
                        description="Compare with previous version"
                        onClick={onComparePrevious}
                      />
                    )}
                  </DropdownMenuGroup>
                  <DropdownMenuSeparator />
                  <DropdownMenuGroup>
                    <DropdownMenuItem
                      startContent={<DownloadIcon className="size-4" />}
                      title="Export JSON"
                      description="Download version snapshot"
                      onClick={onExport}
                    />
                  </DropdownMenuGroup>
                  <DropdownMenuSeparator />
                  <DropdownMenuGroup>
                    <DropdownMenuItem
                      startContent={<RotateCcw className="size-4" />}
                      title="Rollback"
                      description={
                        isCurrent
                          ? "Already on this version"
                          : "Rollback to this version"
                      }
                      color="danger"
                      onClick={onRollback}
                      disabled={isRollingBack || isCurrent}
                    />
                  </DropdownMenuGroup>
                </DropdownMenuContent>
              </DropdownMenu>
            )}
          </div>
        </div>
      </div>

      <Dialog open={tagsDialogOpen} onOpenChange={setTagsDialogOpen}>
        <DialogContent className="sm:max-w-[360px]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <TagIcon className="size-4" />
              Manage Tags
            </DialogTitle>
            <DialogDescription>
              Select tags for version {version.versionNumber}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-1 py-2">
            {VERSION_TAG_OPTIONS.map((option) => (
              <div
                key={option.value}
                className="flex items-center gap-3 rounded-md p-2 hover:bg-muted"
              >
                <Checkbox
                  id={`tag-dialog-${version.id}-${option.value}`}
                  checked={selectedTags.includes(option.value)}
                  onCheckedChange={() => handleTagToggle(option.value)}
                />
                <label
                  htmlFor={`tag-dialog-${version.id}-${option.value}`}
                  className="flex flex-1 cursor-pointer flex-col"
                >
                  <span
                    className={cn(
                      "inline-flex w-fit items-center gap-1 rounded px-1.5 py-0.5 text-xs font-medium",
                      option.color,
                    )}
                  >
                    {option.label}
                  </span>
                  <span className="mt-0.5 text-xs text-muted-foreground">
                    {option.description}
                  </span>
                </label>
              </div>
            ))}
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setTagsDialogOpen(false)}
            >
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={handleSaveTags}
              disabled={updateTagsMutation.isPending}
            >
              {updateTagsMutation.isPending ? (
                "Saving..."
              ) : (
                <>
                  <CheckIcon className="mr-1 size-3" />
                  Save Tags
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function VersionListSkeleton() {
  return (
    <div className="space-y-2 pr-4">
      {[...Array(5)].map((_, i) => (
        <div key={i} className="rounded-lg border p-3">
          <div className="mb-2 flex items-center gap-2">
            <Skeleton className="h-5 w-12" />
            <Skeleton className="h-5 w-16" />
          </div>
          <Skeleton className="mb-2 h-4 w-3/4" />
          <Skeleton className="h-3 w-24" />
        </div>
      ))}
    </div>
  );
}
