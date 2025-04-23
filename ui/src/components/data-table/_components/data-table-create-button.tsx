/* eslint-disable react-compiler/react-compiler */
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useTableStore } from "@/stores/table-store";
import { DataTableCreateButtonProps } from "@/types/data-table";
import { faPlus } from "@fortawesome/pro-regular-svg-icons";
import { PlusIcon, UploadIcon } from "@radix-ui/react-icons";
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
  const [isPopoverOpen, setIsPopoverOpen] = useState<boolean>(false);

  // Get import modal state from the store
  const [showImportModal, setShowImportModal] =
    useTableStore.use("showImportModal");

  // Memoized click handlers
  const handleCreateClick = useCallback(() => {
    setIsPopoverOpen(false);

    if (onCreateClick) {
      onCreateClick();
    } else {
      useTableStore.set("showCreateModal", true);
    }
  }, [onCreateClick]);

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
            <AddNewButton name={name} handleCreateClick={handleCreateClick} />
            <ImportButton name={name} />
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

function AddNewButton({
  name,
  handleCreateClick,
}: {
  name: string;
  handleCreateClick: () => void;
}) {
  return (
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
  );
}

function ImportButton({ name }: { name: string }) {
  return (
    <Button
      variant="ghost"
      className="flex size-full flex-col items-start gap-0.5 text-left"
      disabled
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
  );
}
