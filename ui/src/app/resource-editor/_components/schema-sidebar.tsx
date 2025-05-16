import { TreeView, type TreeDataItem } from "@/components/tree-view";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { HttpClientResponse } from "@/lib/http-client";
import { resourceEditorSearchParamsParser } from "@/lib/search-params/resource-editor";
import type {
  ColumnDetails,
  ConstraintDetails,
  IndexDetails,
  SchemaInformation,
} from "@/types/resource-editor";
import { Folder, List, Table2 } from "lucide-react";
import { useQueryStates } from "nuqs";

export function SchemaSidebar({
  results,
}: {
  results?: HttpClientResponse<SchemaInformation>;
}) {
  const [searchParams, setSearchParams] = useQueryStates(
    resourceEditorSearchParamsParser,
  );

  const schemaInfo = results?.data;

  const constructTreeData = (): TreeDataItem[] => {
    if (!schemaInfo) return [];

    return [
      {
        id: schemaInfo.schemaName,
        name: schemaInfo.schemaName,
        icon: Folder,
        children: schemaInfo.tables.map((table) => ({
          id: `${schemaInfo.schemaName}-${table.tableName}`,
          name: table.tableName,
          icon: Table2,
          onClick: () => setSearchParams({ selectedTable: table.tableName }),
          children: [
            {
              id: `${table.tableName}-columns-category`,
              name: "Columns",
              icon: List,
              children: table.columns.map((col: ColumnDetails) => ({
                id: `${table.tableName}-col-${col.columnName}`,
                name: col.columnName,
                // TODO(wolfred): Potentially add onClick to highlight column in main view later
              })),
            },
            {
              id: `${table.tableName}-indexes-category`,
              name: "Indexes",
              icon: List,
              children: table.indexes.map((idx: IndexDetails) => ({
                id: `${table.tableName}-idx-${idx.indexName}`,
                name: idx.indexName,
              })),
            },
            {
              id: `${table.tableName}-constraints-category`,
              name: "Constraints",
              icon: List,
              children: table.constraints.map((con: ConstraintDetails) => ({
                id: `${table.tableName}-con-${con.constraintName}`,
                name: con.constraintName,
              })),
            },
          ],
        })),
      },
    ];
  };

  const treeData = constructTreeData();

  return (
    <div className="w-1/4 flex flex-col h-full rounded-lg border bg-sidebar">
      <div className="p-4 border-b">
        <h2 className="text-xl font-semibold text-card-foreground">
          Entities ({schemaInfo?.tables?.length ?? 0})
        </h2>
      </div>
      <ScrollArea className="flex h-[calc(100%-60px)] text-card-foreground px-2">
        {treeData.length > 0 ? (
          <TreeView
            data={treeData}
            initialSelectedItemId={
              searchParams.selectedTable
                ? `${schemaInfo?.schemaName}-${searchParams.selectedTable}`
                : undefined
            }
            onSelectChange={(item) => {
              if (item && item.icon === Table2) {
                const tableName = item.id.split("-").slice(1).join("-");
                const tableDetail = schemaInfo?.tables.find(
                  (t) => t.tableName === tableName,
                );
                setSearchParams({
                  selectedTable: tableDetail?.tableName ?? null,
                });
              } else if (!item) {
                setSearchParams({ selectedTable: null });
              }
            }}
          />
        ) : (
          <p className="text-sm text-muted-foreground p-2">
            No tables found in schema: {schemaInfo?.schemaName ?? "N/A"}
          </p>
        )}
      </ScrollArea>
    </div>
  );
}
