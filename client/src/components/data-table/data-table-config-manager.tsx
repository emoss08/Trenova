"use no memo";
import { Button } from "@/components/ui/button";
import { queries } from "@/lib/queries";
import type { TableConfig } from "@/types/table-configuration";
import { useSuspenseQuery } from "@tanstack/react-query";
import { BookmarkIcon, PlusIcon } from "lucide-react";
import { useState } from "react";
import { Popover, PopoverContent, PopoverTrigger } from "../ui/popover";
import { ScrollArea } from "../ui/scroll-area";
import { DataTableConfigItem } from "./_components/data-table-config/data-table-config-item";

type DataTableConfigManagerProps = {
  resource: string;
  onApplyConfig: (config: TableConfig) => void;
  onSaveConfig: () => void;
};

export default function DataTableConfigManager({
  resource,
  onApplyConfig,
  onSaveConfig,
}: DataTableConfigManagerProps) {
  const [open, setOpen] = useState(false);
  const { data, isLoading } = useSuspenseQuery(
    queries.tableConfiguration.all({ resource, limit: 50 }),
  );

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button variant="outline" size="sm">
            <BookmarkIcon className="size-4" />
            <span className="hidden pt-0.5 lg:inline">Views</span>
          </Button>
        }
      />
      <PopoverContent align="end" className="dark w-64 gap-1 p-0">
        {isLoading ? (
          <div className="px-2 py-4 text-center text-sm text-muted-foreground">
            Loading views...
          </div>
        ) : data?.results?.length === 0 ? (
          <div className="px-2 py-4 text-center text-sm text-muted-foreground">
            No saved views yet
          </div>
        ) : (
          <div className="flex flex-col">
            <h3 className="border-b border-border p-1 text-sm font-medium">
              My Views
            </h3>
            <ScrollArea className="flex max-h-[calc(100vh-15rem)] flex-1 flex-col px-1 pt-1">
              {data?.results?.map((config) => (
                <DataTableConfigItem
                  key={config.id}
                  config={config}
                  onApplyConfig={onApplyConfig}
                  setOpen={setOpen}
                />
              ))}
            </ScrollArea>
          </div>
        )}
        <Button
          variant="ghost"
          size="sm"
          className="rounded-t-none border-t border-border"
          onClick={onSaveConfig}
        >
          <PlusIcon className="size-4" />
          Save Current View
        </Button>
      </PopoverContent>
    </Popover>
  );
}
