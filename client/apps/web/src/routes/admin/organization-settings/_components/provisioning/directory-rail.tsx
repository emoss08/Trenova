import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { directoryIdParser } from "@/hooks/use-organization-setting-state";
import { formatUnixDateTimeOrDash } from "@trenova/shared/lib/date";
import { queries } from "@/lib/queries";
import { cn } from "@trenova/shared/lib/utils";
import { apiService } from "@/services/api";
import type { SCIMDirectory } from "@trenova/shared/types/iam";
import { useInfiniteQuery } from "@tanstack/react-query";
import { PlusIcon, UsersRoundIcon } from "lucide-react";
import { useQueryState } from "nuqs";
import { useEffect, useMemo, useRef } from "react";
import { EmptyState, ErrorState } from "../security-access/shared";

const directoryPageSize = 20;

type DirectoryRailProps = {
  organizationId: string;
  onAdd: () => void;
  onDirectoriesChange: (directories: SCIMDirectory[]) => void;
};

export function DirectoryRail({ organizationId, onAdd, onDirectoriesChange }: DirectoryRailProps) {
  const [selectedDirectoryId, setSelectedDirectoryId] = useQueryState(
    "directoryId",
    directoryIdParser,
  );
  const directoriesQuery = useInfiniteQuery({
    queryKey: [
      ...queries.organization.scimDirectories(organizationId).queryKey,
      "rail",
      { limit: directoryPageSize },
    ],
    queryFn: async ({ pageParam }) =>
      apiService.organizationService.listSCIMDirectories(organizationId, {
        limit: directoryPageSize,
        offset: pageParam,
      }),
    initialPageParam: 0,
    getNextPageParam: (lastPage, _, lastPageParam) => {
      if (lastPage.next || lastPage.results.length === directoryPageSize) {
        return lastPageParam + directoryPageSize;
      }
      return undefined;
    },
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
  const directories = useMemo(
    () => directoriesQuery.data?.pages.flatMap((page) => page.results) ?? [],
    [directoriesQuery.data?.pages],
  );
  const { fetchNextPage, hasNextPage, isFetchingNextPage } = directoriesQuery;
  const observerTarget = useRef<HTMLDivElement>(null);

  useEffect(() => {
    onDirectoriesChange(directories);
  }, [directories, onDirectoriesChange]);

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting && hasNextPage && !isFetchingNextPage) {
          void fetchNextPage();
        }
      },
      { threshold: 0.1 },
    );

    const currentTarget = observerTarget.current;
    if (currentTarget) {
      observer.observe(currentTarget);
    }

    return () => {
      if (currentTarget) {
        observer.unobserve(currentTarget);
      }
    };
  }, [fetchNextPage, hasNextPage, isFetchingNextPage]);

  return (
    <div className="flex h-full min-h-0 flex-col rounded-lg border bg-background">
      <div className="flex shrink-0 items-center justify-between border-b p-3">
        <div>
          <div className="text-sm font-medium">SCIM directories</div>
          <div className="text-xs text-muted-foreground">Directory sync tenants</div>
        </div>
        <Button size="sm" onClick={onAdd}>
          <PlusIcon />
          Add
        </Button>
      </div>
      {directoriesQuery.isLoading ? (
        <div className="space-y-2 p-3">
          <Skeleton className="h-14 w-full" />
          <Skeleton className="h-14 w-full" />
        </div>
      ) : directoriesQuery.isError ? (
        <ErrorState label="SCIM directories could not be loaded." compact />
      ) : directories.length > 0 ? (
        <ScrollArea className="min-h-0 flex-1" viewportClassName="min-h-0" maskHeight={18}>
          <div className="divide-y">
            {directories.map((directory) => (
              <DirectoryRailItem
                key={directory.id}
                directory={directory}
                selected={directory.id === selectedDirectoryId}
                onSelect={(directoryId) => void setSelectedDirectoryId(directoryId)}
              />
            ))}
            {directoriesQuery.isFetchingNextPage && (
              <div className="flex items-center justify-center py-3">
                <TextShimmer className="font-mono text-xs" duration={1}>
                  Loading more...
                </TextShimmer>
              </div>
            )}
            <div ref={observerTarget} className="h-px" />
          </div>
        </ScrollArea>
      ) : (
        <EmptyState
          icon={<UsersRoundIcon />}
          label="No directories"
          description="Create a SCIM directory before issuing tokens or mapping groups."
          compact
        />
      )}
    </div>
  );
}

function DirectoryRailItem({
  directory,
  selected,
  onSelect,
}: {
  directory: SCIMDirectory;
  selected: boolean;
  onSelect: (directoryId: string) => void;
}) {
  return (
    <button
      type="button"
      className={cn(
        "flex w-full items-center justify-between gap-3 px-3 py-3 text-left transition-colors hover:bg-muted/40",
        selected && "bg-muted/60",
      )}
      onClick={() => onSelect(directory.id)}
    >
      <div className="min-w-0">
        <div className="truncate text-sm font-medium">{directory.tenantSlug}</div>
        <div className="text-xs text-muted-foreground">
          Updated {formatUnixDateTimeOrDash(directory.updatedAt || directory.createdAt)}
        </div>
      </div>
      <Badge variant={directory.enabled ? "active" : "inactive"}>
        {directory.enabled ? "Enabled" : "Disabled"}
      </Badge>
    </button>
  );
}
