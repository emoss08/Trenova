"use no memo";
import { Button } from "@/components/ui/button";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth-store";
import { usePermission } from "@/hooks/use-permission";
import { Operation, Resource } from "@/types/permission";
import type {
  TableConfig,
  TableConfiguration,
  TableViewSource,
} from "@/types/table-configuration";
import { useSuspenseQuery } from "@tanstack/react-query";
import { BookmarkIcon, PlusIcon, UsersIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { Popover, PopoverContent, PopoverTrigger } from "../ui/popover";
import { ScrollArea } from "../ui/scroll-area";
import { DataTableConfigItem } from "./_components/data-table-config/data-table-config-item";

type DataTableConfigManagerProps = {
  resource: string;
  onApplyConfig: (config: TableConfig, source?: TableViewSource) => void;
  onSaveConfig: () => void;
  currentConfig?: TableConfig;
  activeViewId?: string | null;
  activeViewName?: string | null;
  isViewDirty?: boolean;
  onViewPersisted?: (config: TableConfiguration) => void;
  onViewDeleted?: (id: string) => void;
};

export default function DataTableConfigManager({
  resource,
  onApplyConfig,
  onSaveConfig,
  currentConfig,
  activeViewId = null,
  activeViewName = null,
  isViewDirty = false,
  onViewPersisted,
  onViewDeleted,
}: DataTableConfigManagerProps) {
  const [open, setOpen] = useState(false);
  const currentUserId = useAuthStore((s) => s.user?.id);
  const { allowed: canManageOrgDefaults } = usePermission(Resource.Organization, Operation.Update);
  const { data } = useSuspenseQuery(queries.tableConfiguration.all({ resource, limit: 50 }));

  const { myViews, teamViews } = useMemo(() => {
    const results = data?.results ?? [];
    const mine: TableConfiguration[] = [];
    const team: TableConfiguration[] = [];
    for (const config of results) {
      if (currentUserId && config.userId === currentUserId) {
        mine.push(config);
      } else {
        team.push(config);
      }
    }
    return { myViews: mine, teamViews: team };
  }, [data?.results, currentUserId]);

  const hasViews = myViews.length > 0 || teamViews.length > 0;

  const renderItem = (config: TableConfiguration, isOwn: boolean) => (
    <DataTableConfigItem
      key={config.id}
      config={config}
      isOwn={isOwn}
      isActive={config.id === activeViewId}
      isViewDirty={config.id === activeViewId && isViewDirty}
      canManageOrgDefaults={canManageOrgDefaults}
      currentConfig={currentConfig}
      onApplyConfig={onApplyConfig}
      onViewPersisted={onViewPersisted}
      onViewDeleted={onViewDeleted}
      setOpen={setOpen}
    />
  );

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            variant="ghost"
            size="sm"
            className="group flex flex-row gap-1 data-popup-open:bg-secondary"
          >
            <BookmarkIcon
              className={cn(
                "mb-0.5 size-3.5 text-muted-foreground group-hover:text-foreground group-data-popup-open:text-foreground",
                activeViewName && "text-primary group-hover:text-primary",
              )}
            />
            <span className="hidden max-w-32 truncate text-muted-foreground group-hover:text-foreground group-data-popup-open:text-foreground lg:inline">
              {activeViewName ?? "Views"}
            </span>
            {activeViewName && isViewDirty && (
              <span
                className="size-1.5 shrink-0 rounded-full bg-amber-500"
                title="This view has unsaved changes"
              />
            )}
          </Button>
        }
      />
      <PopoverContent align="end" className="dark w-72 gap-1 p-0">
        {hasViews ? (
          <ScrollArea className="flex max-h-[calc(100vh-15rem)] flex-1 flex-col">
            {myViews.length > 0 && (
              <div className="flex flex-col px-1 pb-1">
                <h3 className="flex items-center gap-1.5 px-1 py-1.5 text-xs font-medium text-muted-foreground uppercase">
                  <BookmarkIcon className="size-3" />
                  My Views
                </h3>
                {myViews.map((config) => renderItem(config, true))}
              </div>
            )}
            {teamViews.length > 0 && (
              <div
                className={cn(
                  "flex flex-col px-1 pb-1",
                  myViews.length > 0 && "border-t border-border",
                )}
              >
                <h3 className="flex items-center gap-1.5 px-1 py-1.5 text-xs font-medium text-muted-foreground uppercase">
                  <UsersIcon className="size-3" />
                  Team Views
                </h3>
                {teamViews.map((config) => renderItem(config, false))}
              </div>
            )}
          </ScrollArea>
        ) : (
          <div className="flex flex-col items-center gap-1 px-2 py-6 text-center">
            <BookmarkIcon className="size-4 text-muted-foreground" />
            <p className="text-sm font-medium">No saved views</p>
            <p className="text-xs text-muted-foreground">
              Configure filters, sorting, and columns, then save them as a reusable view.
            </p>
          </div>
        )}
        <Button
          variant="ghost"
          size="sm"
          className="rounded-t-none border-t border-border"
          onClick={() => {
            setOpen(false);
            onSaveConfig();
          }}
        >
          <PlusIcon className="size-4" />
          Save Current View
        </Button>
      </PopoverContent>
    </Popover>
  );
}
