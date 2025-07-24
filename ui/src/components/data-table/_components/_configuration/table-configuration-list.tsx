/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Icon } from "@/components/ui/icons";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { searchParamsParser } from "@/hooks/use-data-table-state";
import { queries } from "@/lib/queries";
import type { TableConfigurationSchema } from "@/lib/schemas/table-configuration-schema";
import { cn } from "@/lib/utils";
import { api } from "@/services/api";
import { useUser } from "@/stores/user-store";
import type { Resource } from "@/types/audit-entry";
import { faCopy, faSearch } from "@fortawesome/pro-regular-svg-icons";
import {
  faEllipsis,
  faPencil,
  faShare,
  faTableColumns,
  faTrash,
} from "@fortawesome/pro-solid-svg-icons";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import React, { memo, useState } from "react";
import { toast } from "sonner";
import { useDataTable } from "../../data-table-provider";
import { TableConfigurationEditModal } from "./table-configuration-edit-modal";
import { TableConfigurationShareModal } from "./table-configuration-share-modal";

function TableConfigurationListInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1 p-1 size-full border border-border border-dashed rounded-md">
      {children}
    </div>
  );
}

const TableConfigurationListHeader = memo(
  function TableConfigurationListHeader({
    userConfigurations,
  }: {
    userConfigurations: TableConfigurationSchema[];
  }) {
    return (
      <div className="flex justify-center text-center items-center gap-1 text-muted-foreground text-2xs border-b border-border border-dashed pb-1">
        My Configurations ({userConfigurations?.length})
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

export function UserTableConfigurationList({
  resource,
  open,
}: {
  resource: Resource;
  open: boolean;
}) {
  const user = useUser();
  const [searchQuery, setSearchQuery] = useState<string>("");
  const { data: userConfigurations, isLoading: isLoadingUserConfigurations } =
    useQuery({
      ...queries.tableConfiguration.listUserConfigurations(resource),
      enabled: open,
    });
  const {
    data: publicConfigurations,
    isLoading: isLoadingPublicConfigurations,
  } = useQuery({
    ...queries.tableConfiguration.listPublicConfigurations(resource),
    enabled: open,
  });

  // * Exclude public configurations that the user is the owner of
  const filteredPublicConfigurations =
    publicConfigurations?.results.filter(
      (config) => config.creator?.id !== user?.id,
    ) ?? [];

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
        isLoadingPublicConfigurations={isLoadingPublicConfigurations}
        publicConfigurations={filteredPublicConfigurations}
      />
    </TableConfigurationListInner>
  );
}

function TableConfigurationContent({
  isLoadingUserConfigurations,
  userConfigurations,
  isLoadingPublicConfigurations,
  publicConfigurations,
}: {
  isLoadingUserConfigurations: boolean;
  userConfigurations: TableConfigurationSchema[];
  isLoadingPublicConfigurations: boolean;
  publicConfigurations: TableConfigurationSchema[];
}) {
  return (
    <ScrollArea className="h-[220px] w-full pt-2">
      <div className="flex flex-col gap-0.5">
        {isLoadingUserConfigurations || isLoadingPublicConfigurations ? (
          <div className="flex flex-col gap-1 text-center justify-center items-center p-2">
            {Array.from({ length: 5 }).map((_, index) => (
              <Skeleton key={index} className="w-full h-6" />
            ))}
          </div>
        ) : (
          userConfigurations.length == 0 && (
            <div className="flex flex-col gap-1 text-center justify-center items-center p-2">
              <p className="text-sm">No configurations found</p>
              <p className="text-2xs text-muted-foreground">
                Table Configurations allow you to save your current column
                configuration for reuse.
              </p>
            </div>
          )
        )}
        {userConfigurations?.map((config) => (
          <TableConfigurationListItem key={config.id} config={config} />
        ))}

        {publicConfigurations?.length > 0 && (
          <>
            <div className="flex flex-col gap-1 text-left p-2 border-t border-border border-dashed pt-2">
              <p className="text-xs text-muted-foreground">
                Public Configurations ({publicConfigurations?.length})
              </p>
            </div>
            {publicConfigurations?.map((config) => (
              <PublicTableConfigurationListItem
                key={config.id}
                config={config}
              />
            ))}
          </>
        )}
      </div>
    </ScrollArea>
  );
}

function TableConfigurationListItem({
  config,
}: {
  config: TableConfigurationSchema;
}) {
  const { table } = useDataTable();
  const queryClient = useQueryClient();
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [shareModalOpen, setShareModalOpen] = useState(false);
  const [, setSearchParams] = useQueryStates(searchParamsParser);

  const applyConfig = () => {
    if (!table) return;

    table.setColumnVisibility(config.tableConfig.columnVisibility);

    // * Set column order if available
    if (config.tableConfig.columnOrder) {
      table.setColumnOrder(config.tableConfig.columnOrder);
    }

    // * Set the search params to the configuration filters and sort
    setSearchParams({
      filters: JSON.stringify(config.tableConfig.filters) as string,
      sort: JSON.stringify(config.tableConfig.sort) as string,
    });
  };

  const { mutate: deleteConfig, isPending: isDeletingConfig } = useMutation({
    mutationFn: api.tableConfigurations.delete,
    onSuccess: () => {
      toast.success("Configuration deleted");
      queryClient.invalidateQueries({
        queryKey: queries.tableConfiguration.listUserConfigurations._def,
      });
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  return (
    <>
      <div className="group flex text-left items-center justify-between rounded-md py-0.5 px-2 w-full hover:bg-accent">
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
            className="text-xs w-[170px] truncate"
          >
            {config.name}
          </p>
        </div>
        <div className="flex items-center justify-center gap-2">
          <DropdownMenu open={dropdownOpen} onOpenChange={setDropdownOpen}>
            <DropdownMenuTrigger asChild>
              <button
                title={`${config.name} configuration options`}
                aria-label={`${config.name} configuration options`}
                aria-describedby={`${config.name} configuration options`}
                type="button"
                className={cn(
                  "opacity-0 cursor-pointer group-hover:opacity-100 transition-opacity",
                  dropdownOpen && "opacity-100",
                )}
              >
                <Icon
                  icon={faEllipsis}
                  className="size-3 text-muted-foreground"
                />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuGroup>
                <DropdownMenuLabel>Actions</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  title="Apply"
                  description="Apply the configuration to the table"
                  onClick={applyConfig}
                  startContent={
                    <Icon icon={faTableColumns} className="size-3" />
                  }
                />
                <DropdownMenuItem
                  title="Edit"
                  description="Edit the configuration options"
                  startContent={<Icon icon={faPencil} className="size-3" />}
                  onClick={() => setEditModalOpen(!editModalOpen)}
                />
                <DropdownMenuItem
                  title="Share"
                  description="Share the configuration with another user"
                  onClick={() => setShareModalOpen(!shareModalOpen)}
                  startContent={<Icon icon={faShare} className="size-3" />}
                />
                <DropdownMenuItem
                  title="Delete"
                  color="danger"
                  description="Delete the configuration"
                  onClick={() => deleteConfig(config.id ?? "")}
                  disabled={isDeletingConfig}
                  startContent={<Icon icon={faTrash} className="size-3" />}
                />
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
      <TableConfigurationEditModal
        config={config}
        open={editModalOpen}
        onOpenChange={setEditModalOpen}
      />
      {shareModalOpen && (
        <TableConfigurationShareModal
          configId={config.id}
          open={shareModalOpen}
          onOpenChange={setShareModalOpen}
        />
      )}
    </>
  );
}

function PublicTableConfigurationListItem({
  config,
}: {
  config: TableConfigurationSchema;
}) {
  const { table } = useDataTable();
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const [, setSearchParams] = useQueryStates(searchParamsParser);
  const queryClient = useQueryClient();

  const applyConfig = () => {
    if (!table) return;

    table.setColumnVisibility(config.tableConfig.columnVisibility);

    // * Set column order if available
    if (config.tableConfig.columnOrder) {
      table.setColumnOrder(config.tableConfig.columnOrder);
    }

    // * Set the search params to the configuration filters and sort
    setSearchParams({
      filters: JSON.stringify(config.tableConfig.filters) as string,
      sort: JSON.stringify(config.tableConfig.sort) as string,
    });
  };

  const { mutate: copyConfig, isPending: isCopyingConfig } = useMutation({
    mutationFn: (configID: string) =>
      api.tableConfigurations.copy({ configID }),
    onSuccess: () => {
      toast.success("Configuration copied");
      queryClient.invalidateQueries({
        queryKey: queries.tableConfiguration.listUserConfigurations._def,
      });
    },
    onError: (error) => {
      toast.error(error.message);
    },
  });

  return (
    <>
      <div className="group flex text-left items-center justify-between rounded-md py-0.5 px-2 w-full hover:bg-accent">
        <div className="flex flex-col items-start gap-0.5">
          <p className="text-xs w-[170px] truncate">{config.name}</p>
          <p className="text-2xs text-muted-foreground">
            Created by {config.creator?.name}
          </p>
        </div>
        <div className="flex items-center justify-center gap-2">
          <DropdownMenu open={dropdownOpen} onOpenChange={setDropdownOpen}>
            <DropdownMenuTrigger asChild>
              <button
                title={`${config.name} configuration options`}
                aria-label={`${config.name} configuration options`}
                aria-describedby={`${config.name} configuration options`}
                type="button"
                className={cn(
                  "opacity-0 cursor-pointer group-hover:opacity-100 transition-opacity",
                  dropdownOpen && "opacity-100",
                )}
              >
                <Icon
                  icon={faEllipsis}
                  className="size-3 text-muted-foreground"
                />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuGroup>
                <DropdownMenuLabel>Actions</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  title="Apply"
                  description="Apply the configuration to the table"
                  onClick={applyConfig}
                  startContent={
                    <Icon icon={faTableColumns} className="size-3" />
                  }
                />
                <DropdownMenuItem
                  title="Copy"
                  description="Copy the configuration to your own"
                  onClick={() => copyConfig(config.id ?? "")}
                  disabled={isCopyingConfig}
                  startContent={<Icon icon={faCopy} className="size-3" />}
                />
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </>
  );
}
