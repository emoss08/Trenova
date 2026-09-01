import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useTheme } from "@trenova/shared/components/theme-provider";
import { cn } from "@trenova/shared/lib/utils";
import { queries } from "@/lib/queries";
import type { FieldChange } from "@trenova/shared/types/formula-template";
import { useQuery } from "@tanstack/react-query";
import { MinusIcon, PlusIcon, RefreshCwIcon } from "lucide-react";
import ReactDiffViewer, { DiffMethod } from "react-diff-viewer-continued";

type VersionCompareDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  templateId: string;
  fromVersion: number;
  toVersion: number;
};

export function VersionCompareDialog({
  open,
  onOpenChange,
  templateId,
  fromVersion,
  toVersion,
}: VersionCompareDialogProps) {
  const { data, isLoading, error } = useQuery({
    ...queries.formulaTemplate.versionDiff(templateId, fromVersion, toVersion),
    enabled: open,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[80vh] max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            Compare Versions
            <Badge variant="outline" className="font-mono">
              v{fromVersion} → v{toVersion}
            </Badge>
          </DialogTitle>
          <DialogDescription>
            {data?.changeCount ?? 0} change{data?.changeCount !== 1 ? "s" : ""} detected
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="h-[50vh] pr-4">
          {isLoading ? (
            <ComparisonSkeleton />
          ) : error ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <p className="text-muted-foreground">Failed to load comparison. Please try again.</p>
            </div>
          ) : data?.changeCount === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <RefreshCwIcon className="text-muted-foreground mb-4 size-12" />
              <p className="text-muted-foreground">No changes between versions</p>
            </div>
          ) : (
            <div className="space-y-3">
              {Object.entries(data?.changes ?? {}).map(([path, change]) => (
                <ChangeItem key={path} path={path} change={change} />
              ))}
            </div>
          )}
        </ScrollArea>
      </DialogContent>
    </Dialog>
  );
}

type ChangeItemProps = {
  path: string;
  change: FieldChange;
};

function ChangeItem({ path, change }: ChangeItemProps) {
  const getChangeIcon = () => {
    switch (change.type) {
      case "created":
        return <PlusIcon className="size-4 text-green-500" />;
      case "deleted":
        return <MinusIcon className="size-4 text-red-500" />;
      case "updated":
        return <RefreshCwIcon className="size-4 text-blue-500" />;
      default:
        return null;
    }
  };

  const getChangeBadgeVariant = () => {
    switch (change.type) {
      case "created":
        return "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400";
      case "deleted":
        return "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400";
      case "updated":
        return "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400";
      default:
        return "bg-muted text-muted-foreground";
    }
  };

  const formatValue = (value: unknown): string => {
    if (value === null || value === undefined) {
      return "null";
    }
    if (typeof value === "object") {
      return JSON.stringify(value, null, 2);
    }
    if (
      typeof value === "string" ||
      typeof value === "number" ||
      typeof value === "boolean" ||
      typeof value === "bigint"
    ) {
      return String(value);
    }
    return JSON.stringify(value);
  };

  const formattedPath = path.replace(/\./g, " → ");
  const isExpressionChange = path === "expression" || path.endsWith(".expression");

  return (
    <div className="overflow-hidden rounded-lg border">
      <div className="bg-muted/50 flex items-center gap-2 border-b px-3 py-2">
        {getChangeIcon()}
        <span className="font-mono text-sm font-medium">{formattedPath}</span>
        <Badge className={cn("ml-auto text-xs", getChangeBadgeVariant())}>{change.type}</Badge>
      </div>

      <div className="p-3">
        {change.type === "updated" && isExpressionChange ? (
          <ExpressionDiff before={formatValue(change.from)} after={formatValue(change.to)} />
        ) : change.type === "updated" ? (
          <div className="grid grid-cols-2 gap-4">
            <div>
              <span className="text-muted-foreground mb-1 block text-xs font-medium">Before</span>
              <pre className="overflow-x-auto rounded bg-red-50 p-2 font-mono text-xs whitespace-pre-wrap text-red-800 dark:bg-red-900/20 dark:text-red-200">
                {formatValue(change.from)}
              </pre>
            </div>
            <div>
              <span className="text-muted-foreground mb-1 block text-xs font-medium">After</span>
              <pre className="overflow-x-auto rounded bg-green-50 p-2 font-mono text-xs whitespace-pre-wrap text-green-800 dark:bg-green-900/20 dark:text-green-200">
                {formatValue(change.to)}
              </pre>
            </div>
          </div>
        ) : change.type === "created" ? (
          <div>
            <span className="text-muted-foreground mb-1 block text-xs font-medium">Added</span>
            <pre className="overflow-x-auto rounded bg-green-50 p-2 font-mono text-xs whitespace-pre-wrap text-green-800 dark:bg-green-900/20 dark:text-green-200">
              {formatValue(change.to)}
            </pre>
          </div>
        ) : (
          <div>
            <span className="text-muted-foreground mb-1 block text-xs font-medium">Removed</span>
            <pre className="overflow-x-auto rounded bg-red-50 p-2 font-mono text-xs whitespace-pre-wrap text-red-800 dark:bg-red-900/20 dark:text-red-200">
              {formatValue(change.from)}
            </pre>
          </div>
        )}
      </div>
    </div>
  );
}

function ExpressionDiff({ before, after }: { before: string; after: string }) {
  const { theme } = useTheme();
  const isDark =
    theme === "dark" ||
    (theme === "system" && window.matchMedia("(prefers-color-scheme: dark)").matches);

  return (
    <div className="overflow-hidden rounded-md border font-mono text-xs">
      <ReactDiffViewer
        oldValue={before}
        newValue={after}
        splitView={false}
        compareMethod={DiffMethod.WORDS}
        useDarkTheme={isDark}
        hideLineNumbers
      />
    </div>
  );
}

function ComparisonSkeleton() {
  return (
    <div className="space-y-3">
      {[...Array(3)].map((_, i) => (
        <div key={i} className="overflow-hidden rounded-lg border">
          <div className="bg-muted/50 flex items-center gap-2 border-b px-3 py-2">
            <Skeleton className="size-4 rounded" />
            <Skeleton className="h-4 w-32" />
            <Skeleton className="ml-auto h-5 w-16" />
          </div>
          <div className="p-3">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Skeleton className="mb-2 h-3 w-12" />
                <Skeleton className="h-16 w-full" />
              </div>
              <div>
                <Skeleton className="mb-2 h-3 w-12" />
                <Skeleton className="h-16 w-full" />
              </div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
