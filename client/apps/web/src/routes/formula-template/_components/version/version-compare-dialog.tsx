import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import type { FieldChange } from "@/types/formula-template";
import { useQuery } from "@tanstack/react-query";
import { MinusIcon, PlusIcon, RefreshCwIcon } from "lucide-react";

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
    queryKey: ["formula-template-compare", templateId, fromVersion, toVersion],
    queryFn: () =>
      apiService.formulaTemplateService.compareVersions(
        templateId,
        fromVersion,
        toVersion,
      ),
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
            {data?.changeCount ?? 0} change{data?.changeCount !== 1 ? "s" : ""}{" "}
            detected
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="h-[50vh] pr-4">
          {isLoading ? (
            <ComparisonSkeleton />
          ) : error ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <p className="text-muted-foreground">
                Failed to load comparison. Please try again.
              </p>
            </div>
          ) : data?.changeCount === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <RefreshCwIcon className="mb-4 size-12 text-muted-foreground" />
              <p className="text-muted-foreground">
                No changes between versions
              </p>
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

  return (
    <div className="overflow-hidden rounded-lg border">
      <div className="flex items-center gap-2 border-b bg-muted/50 px-3 py-2">
        {getChangeIcon()}
        <span className="font-mono text-sm font-medium">{formattedPath}</span>
        <Badge className={cn("ml-auto text-xs", getChangeBadgeVariant())}>
          {change.type}
        </Badge>
      </div>

      <div className="p-3">
        {change.type === "updated" ? (
          <div className="grid grid-cols-2 gap-4">
            <div>
              <span className="mb-1 block text-xs font-medium text-muted-foreground">
                Before
              </span>
              <pre className="overflow-x-auto rounded bg-red-50 p-2 font-mono text-xs whitespace-pre-wrap text-red-800 dark:bg-red-900/20 dark:text-red-200">
                {formatValue(change.from)}
              </pre>
            </div>
            <div>
              <span className="mb-1 block text-xs font-medium text-muted-foreground">
                After
              </span>
              <pre className="overflow-x-auto rounded bg-green-50 p-2 font-mono text-xs whitespace-pre-wrap text-green-800 dark:bg-green-900/20 dark:text-green-200">
                {formatValue(change.to)}
              </pre>
            </div>
          </div>
        ) : change.type === "created" ? (
          <div>
            <span className="mb-1 block text-xs font-medium text-muted-foreground">
              Added
            </span>
            <pre className="overflow-x-auto rounded bg-green-50 p-2 font-mono text-xs whitespace-pre-wrap text-green-800 dark:bg-green-900/20 dark:text-green-200">
              {formatValue(change.to)}
            </pre>
          </div>
        ) : (
          <div>
            <span className="mb-1 block text-xs font-medium text-muted-foreground">
              Removed
            </span>
            <pre className="overflow-x-auto rounded bg-red-50 p-2 font-mono text-xs whitespace-pre-wrap text-red-800 dark:bg-red-900/20 dark:text-red-200">
              {formatValue(change.from)}
            </pre>
          </div>
        )}
      </div>
    </div>
  );
}

function ComparisonSkeleton() {
  return (
    <div className="space-y-3">
      {[...Array(3)].map((_, i) => (
        <div key={i} className="overflow-hidden rounded-lg border">
          <div className="flex items-center gap-2 border-b bg-muted/50 px-3 py-2">
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
