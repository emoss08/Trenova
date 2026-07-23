import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { Badge } from "@/components/ui/badge";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import type { CatalogFilter } from "./_components/catalog";
import {
  catalog,
  parseSelectionParam,
  resolveSelection,
  serializeSelectionParam,
} from "./_components/catalog";
import { DetailPanel } from "./_components/detail-panel";
import { ListPanel } from "./_components/list-panel";
import type { CatalogSelection } from "@/types/graphql-catalog";
import { useCallback, useMemo, useState } from "react";
import { useSearchParams } from "react-router";

export function GraphQLExplorerPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [filter, setFilter] = useState<CatalogFilter>("all");

  const query = searchParams.get("q") ?? "";
  const selection = useMemo(() => parseSelectionParam(searchParams.get("sel")), [searchParams]);
  const { operation, fragment } = useMemo(() => resolveSelection(selection), [selection]);

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
    <AdminPageLayout className="flex h-[calc(100vh-3rem)] flex-col">
      <PageHeader
        title="GraphQL Explorer"
        description="Browse, search, and run every persisted GraphQL operation in the client"
        actions={
          <Badge variant="secondary" className="font-normal">
            {catalog.operationCount} operations · {catalog.fragmentCount} fragments
          </Badge>
        }
      />
      <div className="min-h-0 flex-1 p-0">
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
      </div>
    </AdminPageLayout>
  );
}
