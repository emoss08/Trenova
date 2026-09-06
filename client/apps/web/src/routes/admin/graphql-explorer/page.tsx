import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@trenova/shared/components/ui/resizable";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import type { CatalogFilter, CatalogIndex } from "./_components/catalog";
import {
  parseSelectionParam,
  resolveSelection,
  serializeSelectionParam,
} from "./_components/catalog";
import { DetailPanel } from "./_components/detail-panel";
import { ExplorerSkeleton } from "./_components/explorer-skeleton";
import { ListPanel } from "./_components/list-panel";
import { CatalogProvider, useCatalogQuery } from "./_components/use-catalog";
import type { CatalogSelection } from "@/types/graphql-catalog";
import { CircleAlertIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useSearchParams } from "react-router";

export function GraphQLExplorerPage() {
  const { data: index, isError, error, isFetching, refetch } = useCatalogQuery();

  return (
    <AdminPageLayout className="flex h-[calc(100vh-3rem)] flex-col">
      <PageHeader
        title="GraphQL Explorer"
        description="Browse, search, and run every persisted GraphQL operation in the client"
        actions={
          index ? (
            <Badge variant="secondary" className="font-normal">
              {index.catalog.operationCount} operations · {index.catalog.fragmentCount} fragments
            </Badge>
          ) : (
            <Skeleton className="h-5 w-44 rounded-md" />
          )
        }
      />
      <div className="min-h-0 flex-1 p-0">
        {isError ? (
          <CatalogLoadError
            message={error instanceof Error ? error.message : "The catalog could not be loaded."}
            isRetrying={isFetching}
            onRetry={() => void refetch()}
          />
        ) : index ? (
          <CatalogExplorer index={index} />
        ) : (
          <ExplorerSkeleton />
        )}
      </div>
    </AdminPageLayout>
  );
}

function CatalogLoadError({
  message,
  isRetrying,
  onRetry,
}: {
  message: string;
  isRetrying: boolean;
  onRetry: () => void;
}) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3">
      <CircleAlertIcon className="text-destructive/60 size-5" />
      <div className="text-center">
        <p className="text-foreground text-sm font-medium">Operation catalog unavailable</p>
        <p className="text-muted-foreground mt-0.5 max-w-80 text-xs">{message}</p>
      </div>
      <Button type="button" variant="outline" size="xs" onClick={onRetry} disabled={isRetrying}>
        {isRetrying ? "Retrying…" : "Try again"}
      </Button>
    </div>
  );
}

function CatalogExplorer({ index }: { index: CatalogIndex }) {
  const [searchParams, setSearchParams] = useSearchParams();
  const [filter, setFilter] = useState<CatalogFilter>("all");

  const query = searchParams.get("q") ?? "";
  const selection = useMemo(
    () => parseSelectionParam(index, searchParams.get("sel")),
    [index, searchParams],
  );
  const { operation, fragment } = useMemo(
    () => resolveSelection(index, selection),
    [index, selection],
  );

  const setQuery = useCallback(
    (value: string) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (value) {
            next.set("q", value);
          } else {
            next.delete("q");
          }
          return next;
        },
        { replace: true },
      );
    },
    [setSearchParams],
  );

  const setSelection = useCallback(
    (value: CatalogSelection) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("sel", serializeSelectionParam(value));
        return next;
      });
    },
    [setSearchParams],
  );

  return (
    <CatalogProvider value={index}>
      <ResizablePanelGroup orientation="horizontal" className="min-h-0 flex-1">
        <ResizablePanel defaultSize="340px" minSize="280px" maxSize="480px">
          <ListPanel
            query={query}
            filter={filter}
            selection={selection}
            onQueryChange={setQuery}
            onFilterChange={setFilter}
            onSelect={setSelection}
          />
        </ResizablePanel>
        <ResizableHandle withHandle />
        <ResizablePanel minSize="40%">
          <div className="flex h-full min-h-0 flex-col overflow-hidden p-4">
            <DetailPanel operation={operation} fragment={fragment} onSelect={setSelection} />
          </div>
        </ResizablePanel>
      </ResizablePanelGroup>
    </CatalogProvider>
  );
}
