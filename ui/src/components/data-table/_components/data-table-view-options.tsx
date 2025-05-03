"use no memo";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Switch } from "@/components/ui/switch";
import { toSentenceCase, toTitleCase } from "@/lib/utils";
import { useTableStore } from "@/stores/table-store";
import {
  DataTableCreateButtonProps,
  DataTableViewOptionsProps,
} from "@/types/data-table";
import { faPlus, faSearch } from "@fortawesome/pro-regular-svg-icons";
import { faColumns } from "@fortawesome/pro-solid-svg-icons";
import { ChevronDownIcon, PlusIcon, UploadIcon } from "@radix-ui/react-icons";
import React, { memo, useCallback, useState } from "react";
import { DataTableImportModal } from "./data-table-import-modal";

export const DataTableCreateButton = memo(function DataTableCreateButton({
  name,
  isDisabled,
  onCreateClick,
  exportModelName,
  extraActions,
}: DataTableCreateButtonProps) {
  // Control popover state explicitly
  const [isPopoverOpen, setIsPopoverOpen] = useState(false);

  // Get import modal state from the store
  const [showImportModal, setShowImportModal] =
    useTableStore.use("showImportModal");

  // Memoized click handlers
  const handleCreateClick = useCallback(() => {
    setIsPopoverOpen(false);

    if (onCreateClick) {
      console.info("onCreateClick debug info", {
        onCreateClick,
      });
      onCreateClick();
    }
  }, [onCreateClick]);

  const handleImportClick = useCallback(() => {
    setIsPopoverOpen(false);
    setShowImportModal(true);
  }, [setShowImportModal]);

  return (
    <>
      <Popover open={isPopoverOpen} onOpenChange={setIsPopoverOpen}>
        <PopoverTrigger asChild>
          <Button variant="default" disabled={isDisabled}>
            <Icon icon={faPlus} className="size-4 text-background" />
            <span>New</span>
          </Button>
        </PopoverTrigger>
        <PopoverContent align="end" side="bottom" className="w-auto p-1">
          <div className="flex w-full flex-col gap-1">
            <Button
              variant="ghost"
              className="flex size-full flex-col items-start gap-0.5 text-left"
              onClick={handleCreateClick}
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
              onClick={handleImportClick}
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
            {extraActions?.map((option) => (
              <Button
                key={option.label}
                variant="ghost"
                className="flex size-full flex-col items-start gap-0.5 text-left"
                onClick={option.onClick}
              >
                <div className="flex items-center gap-2">
                  {option.icon && (
                    <Icon icon={option.icon} className="size-4" />
                  )}
                  <span>{option.label}</span>
                  {React.isValidElement(option.endContent) && option.endContent}
                </div>
                <div>
                  <p className="text-xs font-normal text-muted-foreground">
                    {option.description}
                  </p>
                </div>
              </Button>
            ))}
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
});

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
          className="h-7 border-dashed"
          aria-label="Toggle column visibility"
        >
          <Icon icon={faColumns} className="size-4" />
          <span className="hidden lg:inline">Customize Columns</span>
          <span className="lg:hidden">Columns</span>
          <span className="sr-only">Toggle column visibility options</span>
          <ChevronDownIcon />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="end"
        side="bottom"
        className="w-(--radix-popover-trigger-width) p-2"
      >
        <div className="space-y-2">
          <Input
            icon={
              <Icon icon={faSearch} className="size-3 text-muted-foreground" />
            }
            placeholder="Search columns..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="h-7 text-sm bg-background"
          />
          <div className="my-3 border-dashed border-t border-border" />
          <div className="flex flex-col gap-3">
            {filteredColumns.length > 0 ? (
              filteredColumns.map((column) => {
                const isVisible = column.getIsVisible();
                return (
                  <div
                    key={column.id}
                    className="flex items-center justify-between"
                  >
                    <Label htmlFor={column.id} className="flex-grow text-xs">
                      {toTitleCase(column.id)}
                    </Label>
                    <Switch
                      id={column.id}
                      checked={isVisible}
                      size="sm"
                      onCheckedChange={() =>
                        handleToggleVisibility(column.id, isVisible)
                      }
                      title={`Toggle ${toTitleCase(column.id)} column`}
                      aria-describedby={`Toggle ${toTitleCase(column.id)} column`}
                      aria-label={`Toggle ${toTitleCase(column.id)} column`}
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
        </div>
      </PopoverContent>
    </Popover>
  );
}
