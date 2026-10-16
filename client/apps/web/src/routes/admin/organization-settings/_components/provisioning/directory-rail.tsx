import { useT } from "@trenova/shared/i18n/use-t";
import { useSecurityAccess } from "../security-access/use-security-access";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { directoryIdParser } from "@/hooks/use-organization-setting-state";
import { formatUnixDateTimeOrDash } from "@trenova/shared/lib/date";
import { organization } from "@/lib/queries/organization";
import { cn } from "@trenova/shared/lib/utils";
import { listSCIMDirectoriesGraphQL } from "@/lib/graphql/scim-directory";
import type { SCIMDirectory } from "@trenova/shared/types/iam";
import { useInfiniteQuery } from "@tanstack/react-query";
import { PlusIcon, Users01Icon } from "@trenova/shared/components/icons";
import { useQueryState } from "nuqs";
import { useEffect, useMemo, useRef } from "react";
import { EmptyState, ErrorState, RowSkeleton, SecuritySection } from "../security-access/shared";

const directoryPageSize = 20;

type DirectoryRailProps = {
  organizationId: string;
  onAdd: () => void;
  onDirectoriesChange: (directories: SCIMDirectory[]) => void;
};

export function DirectoryRail({ organizationId, onAdd, onDirectoriesChange }: DirectoryRailProps) {
  const t = useT();
  const access = useSecurityAccess();

  const [selectedDirectoryId, setSelectedDirectoryId] = useQueryState(
    "directoryId",
    directoryIdParser,
  );
  const directoriesQuery = useInfiniteQuery({
    queryKey: [
      ...organization.scimDirectories(organizationId).queryKey,
      "rail",
      { first: directoryPageSize },
    ],
    queryFn: async ({ pageParam, signal }) =>
      listSCIMDirectoriesGraphQL({ first: directoryPageSize, after: pageParam }, { signal }),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => (lastPage.hasNextPage ? lastPage.endCursor : undefined),
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
  const directories = useMemo(
    () => directoriesQuery.data?.pages.flatMap((page) => page.directories) ?? [],
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
    <SecuritySection
      title={t("SCIM directories")}
      icon={<Users01Icon />}
      count={directories.length}
      className="h-full min-h-0"
      action={
        access.scimDirectories.create ? (
          <Button size="sm" onClick={onAdd}>
            <PlusIcon />
            {t("New directory")}
          </Button>
        ) : null
      }
    >
      {directoriesQuery.isLoading ? (
        <RowSkeleton rows={2} label={t("Loading SCIM directories")} />
      ) : directoriesQuery.isError ? (
        <ErrorState label={t("SCIM directories could not be loaded.")} />
      ) : directories.length > 0 ? (
        <ScrollArea className="min-h-0 flex-1" viewportClassName="min-h-0" maskHeight={18}>
          <div className="flex flex-col gap-px py-1">
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
                  {t("Loading more...")}
                </TextShimmer>
              </div>
            )}
            <div ref={observerTarget} className="h-px" />
          </div>
        </ScrollArea>
      ) : (
        <EmptyState
          icon={<Users01Icon />}
          label={t("No directories")}
          description={t("Create a SCIM directory before issuing tokens or mapping groups.")}
        />
      )}
    </SecuritySection>
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
  const t = useT();

  return (
    <button
      type="button"
      aria-pressed={selected}
      className={cn(
        "ui-focus-ring hover:bg-surface-hover flex w-full items-center justify-between gap-3 rounded-md px-2 py-2 text-left transition-colors",
        selected && "bg-surface-selected hover:bg-surface-selected",
      )}
      onClick={() => onSelect(directory.id)}
    >
      <div className="min-w-0">
        <div className="truncate text-sm font-medium">{directory.tenantSlug}</div>
        <div className="text-muted-foreground text-xs">
          {t("Updated {0}", formatUnixDateTimeOrDash(directory.updatedAt || directory.createdAt))}
        </div>
      </div>
      <Badge variant={directory.enabled ? "success" : "neutral"}>
        {directory.enabled ? t("Enabled") : t("Disabled")}
      </Badge>
    </button>
  );
}
