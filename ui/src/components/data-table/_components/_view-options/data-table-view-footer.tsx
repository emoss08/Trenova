import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Resource } from "@/types/audit-entry";
import { VisibilityState } from "@tanstack/react-table";
import { useMemo, useState } from "react";
import { useDataTable } from "../../data-table-provider";
import { CreateTableConfigurationModal } from "../_configuration/table-configuration-create-modal";
import { UserTableConfigurationList } from "../_configuration/table-configuration-list";

export function DataTableViewFooter({ resource }: { resource: Resource }) {
  const [showConfigurationList, setShowConfigurationList] = useState(false);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const { table } = useDataTable();

  const hideableColumns = useMemo(
    () =>
      table
        .getAllColumns()
        .filter(
          (column) =>
            typeof column.accessorFn !== "undefined" && column.getCanHide(),
        ),
    [table],
  );
  const comprehensiveVisibilityState = useMemo(() => {
    const state: VisibilityState = {};
    hideableColumns.forEach((column) => {
      state[column.id] = column.getIsVisible();
    });
    return state;
  }, [hideableColumns]);

  return (
    <DataTableViewFooterInner>
      <Popover
        open={showConfigurationList}
        onOpenChange={setShowConfigurationList}
      >
        <PopoverTrigger asChild>
          <Button size="sm" className="w-full" variant="secondary">
            View Configuration(s)
          </Button>
        </PopoverTrigger>
        <PopoverContent
          align="end"
          alignOffset={-10}
          sideOffset={10}
          side="left"
          className="p-1 w-[250px] h-[300px]"
        >
          <UserTableConfigurationList
            resource={resource}
            open={showConfigurationList}
          />
        </PopoverContent>
      </Popover>
      <Button
        onClick={() => setShowCreateModal(!showCreateModal)}
        size="sm"
        className="w-full"
      >
        Save Configuration
      </Button>

      <CreateTableConfigurationModal
        open={showCreateModal}
        onOpenChange={setShowCreateModal}
        resource={resource}
        tableFilters={table.getState().filters}
        visiblityState={comprehensiveVisibilityState}
        columnOrder={table.getState().columnOrder}
      />
    </DataTableViewFooterInner>
  );
}

function DataTableViewFooterInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex border-t border-border border-dashed pt-2 justify-between gap-2">
      {children}
    </div>
  );
}
