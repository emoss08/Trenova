import { Icon } from "@/components/ui/icons";

import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { api } from "@/services/api";
import type { TableConfiguration } from "@/types/table-configuration";
import { faSearch } from "@fortawesome/pro-regular-svg-icons";
import { faSpinner, faTrash } from "@fortawesome/pro-solid-svg-icons";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import React, { memo, useState } from "react";
import { toast } from "sonner";

function TableConfigurationListInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1 p-1 border border-border border-dashed rounded-md">
      {children}
    </div>
  );
}

const TableConfigurationListHeader = memo(
  function TableConfigurationListHeader({
    userConfigurations,
  }: {
    userConfigurations: TableConfiguration[];
  }) {
    return (
      <div className="flex justify-center text-center items-center gap-1 text-muted-foreground text-2xs border-b border-border border-dashed pb-1">
        Saved Configurations ({userConfigurations?.length})
      </div>
    );
  },
  // * We want to memoize this ourselves as the compiler will attempt to do so, but it won't know if the props are the same
  (prevProps, nextProps) => {
    return (
      prevProps.userConfigurations?.length ===
      nextProps.userConfigurations?.length
    );
  },
);

export function TableConfigurationList({
  name,
  open,
}: {
  name: string;
  open: boolean;
}) {
  const [searchQuery, setSearchQuery] = useState<string>("");
  const { data: userConfigurations, isLoading: isLoadingUserConfigurations } =
    useQuery({
      ...queries.tableConfiguration.listUserConfigurations(name),
      enabled: open,
    });

  return (
    <TableConfigurationListInner>
      <TableConfigurationListHeader
        userConfigurations={userConfigurations?.results ?? []}
      />
      {userConfigurations?.results &&
        userConfigurations?.results?.length > 0 && (
          <Input
            icon={
              <Icon icon={faSearch} className="size-3 text-muted-foreground" />
            }
            placeholder="Search configurations..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="h-7 text-sm bg-background"
          />
        )}
      <TableConfigurationContent
        isLoadingUserConfigurations={isLoadingUserConfigurations}
        userConfigurations={userConfigurations?.results ?? []}
      />
    </TableConfigurationListInner>
  );
}

function TableConfigurationContent({
  isLoadingUserConfigurations,
  userConfigurations,
}: {
  isLoadingUserConfigurations: boolean;
  userConfigurations: TableConfiguration[];
}) {
  return (
    <ScrollArea className="h-[150px] w-full pt-2 pr-3">
      {isLoadingUserConfigurations ? (
        <div className="flex flex-col gap-1 text-center justify-center items-center p-2">
          {Array.from({ length: 5 }).map((_, index) => (
            <Skeleton key={index} className="w-full h-6" />
          ))}
        </div>
      ) : (
        <>
          {userConfigurations.length === 0 && (
            <div className="flex flex-col gap-1 text-center justify-center items-center p-2">
              <p className="text-sm">No configurations found</p>
              <p className="text-2xs text-muted-foreground">
                Table Configurations allow you to save your current column
                configuration for reuse.
              </p>
            </div>
          )}
        </>
      )}
      {userConfigurations?.map((config) => (
        <TableConfigurationListItem key={config.id} config={config} />
      ))}
    </ScrollArea>
  );
}

function TableConfigurationListItem({
  config,
}: {
  config: TableConfiguration;
}) {
  const queryClient = useQueryClient();

  const { mutate: deleteConfig, isPending: isDeletingConfig } = useMutation({
    mutationFn: api.tableConfigurations.delete,
    onSuccess: () => {
      toast.success("Configuration deleted");
      // * Invalidate on success
      queryClient.invalidateQueries({
        queryKey: queries.tableConfiguration.listUserConfigurations._def,
      });
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  return (
    <div className="group flex items-center justify-between rounded-md py-0.5 px-2 w-full hover:bg-accent">
      <div className="flex items-center gap-2">
        {config.isDefault && (
          <span
            title="Default configuration"
            aria-label="Default configuration"
            aria-describedby={`${config.name} configuration is the default configuration`}
          >
            <div className="size-2 bg-yellow-500 rounded-full" />
          </span>
        )}
        <p
          title={`${config.name} configuration`}
          aria-describedby={`${config.name} configuration`}
          className="text-xs w-[180px] truncate"
        >
          {config.name}
        </p>
      </div>
      <div className="flex items-center justify-center gap-2">
        <button
          title={`Delete ${config.name} configuration`}
          aria-label={`Delete configuration`}
          aria-describedby={`Delete ${config.name} configuration`}
          type="button"
          className="opacity-0 cursor-pointer group-hover:opacity-100 transition-opacity"
          onClick={() => deleteConfig(config.id)}
          disabled={isDeletingConfig}
        >
          {isDeletingConfig ? (
            <Icon icon={faSpinner} className="size-3 animate-spin" />
          ) : (
            <Icon icon={faTrash} className="size-3 text-red-500" />
          )}
        </button>
      </div>
    </div>
  );
}
