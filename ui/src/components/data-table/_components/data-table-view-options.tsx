import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";

import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Switch } from "@/components/ui/switch";
import { toSentenceCase } from "@/lib/utils";
import { useTableStore } from "@/stores/table-store";
import {
  DataTableCreateButtonProps,
  DataTableViewOptionsProps,
} from "@/types/data-table";
import { faPlusCircle } from "@fortawesome/pro-regular-svg-icons";
import { faEye } from "@fortawesome/pro-solid-svg-icons";
import { PlusIcon, UploadIcon } from "@radix-ui/react-icons";
import React, { useCallback } from "react";
import { DataTableImportModal } from "./data-table-import-modal";

export function DataTableCreateButton({
  name,
  isDisabled,
  onCreateClick,
  exportModelName,
}: DataTableCreateButtonProps) {
  const [showImportModal, setShowImportModal] =
    // eslint-disable-next-line react-compiler/react-compiler
    useTableStore.use("showImportModal");

  const handleClick = useCallback(() => {
    // Handle create logic
    if (onCreateClick) {
      onCreateClick();
    } else {
      useTableStore.set("showCreateModal", true);
    }
  }, [onCreateClick]);

  return (
    <>
      <Popover>
        <PopoverTrigger asChild>
          <Button variant="default" disabled={isDisabled}>
            <Icon icon={faPlusCircle} className="size-4 text-background" />
            <span>New</span>
          </Button>
        </PopoverTrigger>
        <PopoverContent align="end" side="bottom" className="w-auto p-1">
          <div className="flex w-full flex-col gap-1">
            <Button
              variant="ghost"
              className="flex size-full flex-col items-start gap-0.5 text-left"
              onClick={handleClick}
            >
              <div className="flex items-center gap-2">
                <PlusIcon className="size-4" />
                <span>Add New {name}</span>
              </div>
              <div>
                <p className="text-xs font-normal text-muted-foreground">
                  Create a new {name} from scratch
                </p>
              </div>
            </Button>
            <Button
              variant="ghost"
              className="flex size-full flex-col items-start gap-0.5 text-left"
              onClick={() => setShowImportModal(true)}
            >
              <div className="flex items-center gap-2">
                <UploadIcon className="size-4" />
                <span>Import {name}s</span>
              </div>
              <div>
                <p className="text-xs font-normal text-muted-foreground">
                  Import existing {name}s from a file
                </p>
              </div>
            </Button>
          </div>
        </PopoverContent>
      </Popover>
      {showImportModal && (
        <DataTableImportModal
          open={showImportModal}
          onOpenChange={setShowImportModal}
          name={name}
          exportModelName={exportModelName}
        />
      )}
    </>
  );
}

export function DataTableViewOptions<TData>({
  table,
}: DataTableViewOptionsProps<TData>) {
  const [open, setOpen] = React.useState(false);
  const [searchQuery, setSearchQuery] = React.useState("");

  // Get all hideable columns
  const columns = React.useMemo(
    () =>
      table
        .getAllColumns()
        .filter(
          (column) =>
            typeof column.accessorFn !== "undefined" && column.getCanHide(),
        ),
    [table],
  );

  // Filter columns based on search query
  const filteredColumns = React.useMemo(
    () =>
      columns.filter((column) =>
        toSentenceCase(column.id)
          .toLowerCase()
          .includes(searchQuery.toLowerCase()),
      ),
    [columns, searchQuery],
  );

  // Get visible columns count from table state
  const visibleColumnsCount = table.getVisibleLeafColumns().length;

  const handleToggleVisibility = React.useCallback(
    (columnId: string, isVisible: boolean) => {
      table.getColumn(columnId)?.toggleVisibility(!isVisible);
    },
    [table],
  );

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          className="h-8 border-dashed"
          aria-label="Toggle column visibility"
        >
          <Icon icon={faEye} className="size-4" />
          View
          <Badge
            variant="default"
            withDot={false}
            className="ml-0.5 size-4 text-xs p-1 rounded-sm"
          >
            {visibleColumnsCount}
          </Badge>
          <span className="sr-only">Toggle column visibility options</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" side="bottom" className="w-[200px] p-2">
        <div className="space-y-2">
          <Input
            placeholder="Search columns..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="h-8 text-sm"
          />
          <div className="my-3 border-dashed border-t border-border" />
          <ScrollArea className="h-72">
            <div className="space-y-1.5">
              {filteredColumns.length > 0 ? (
                filteredColumns.map((column) => {
                  const isVisible = column.getIsVisible();
                  return (
                    <div
                      key={column.id}
                      className="flex items-center justify-between space-x-2 rounded-md px-2 py-1"
                    >
                      <Label
                        htmlFor={column.id}
                        className="flex-grow text-sm font-normal"
                      >
                        {toSentenceCase(column.id)}
                      </Label>
                      <Switch
                        id={column.id}
                        checked={isVisible}
                        onCheckedChange={() =>
                          handleToggleVisibility(column.id, isVisible)
                        }
                        aria-label={`Toggle ${toSentenceCase(column.id)} column`}
                      />
                    </div>
                  );
                })
              ) : (
                <p className="p-2 text-sm text-muted-foreground">
                  No columns found
                </p>
              )}
            </div>
          </ScrollArea>
        </div>
      </PopoverContent>
    </Popover>
  );
}
