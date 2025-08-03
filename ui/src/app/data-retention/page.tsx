/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { QueryLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { queries } from "@/lib/queries";
import { api } from "@/services/api";
import {
  faDownload,
  faExclamationTriangle,
} from "@fortawesome/pro-regular-svg-icons";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { lazy, memo } from "react";
import { toast } from "sonner";

const BackupList = lazy(() => import("./_components/backup-list"));

export function DataRetention() {
  return (
    <div className="flex flex-col space-y-6">
      <MetaTags
        title="Data Retention"
        description="Database Backup & Data Retention"
      />
      <Header />
      <BackupAlert />
      <QueryLazyComponent
        queryKey={queries.organization.getDatabaseBackups._def}
      >
        <BackupList />
      </QueryLazyComponent>
    </div>
  );
}

const Header = memo(() => {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Data Retention</h1>
        <p className="text-muted-foreground">
          Manage database backups, configure retention policies, and restore
          data when needed
        </p>
      </div>
    </div>
  );
});
Header.displayName = "Header";

const BackupAlert = memo(() => {
  const queryClient = useQueryClient();

  const createBackup = useMutation({
    mutationFn: async () => {
      // API call to delete the backup
      return await api.databaseBackups.create();
    },
    onSuccess: () => {
      toast.success("Backup created successfully");
      queryClient.invalidateQueries({
        queryKey: queries.organization.getDatabaseBackups._def,
      });
    },
    onError: (error) => {
      toast.error(
        `Failed to delete backup: ${error instanceof Error ? error.message : "Unknown error"}`,
      );
    },
  });

  return (
    <div className="flex bg-amber-500/20 border border-amber-600/50 p-4 rounded-md justify-between items-center mb-4 w-full">
      <div className="flex items-center gap-2 w-full text-amber-600">
        <Icon icon={faExclamationTriangle} className="size-4" />
        <div className="flex flex-col">
          <p className="text-sm font-medium">Automatic Backups</p>
          <p className="text-xs dark:text-amber-100">
            Backups run automatically based on your configured schedule. You can
            also create manual backups as needed.
          </p>
        </div>
      </div>
      <div className="flex gap-2">
        <Button
          variant="outline"
          className="flex items-center gap-2 text-amber-600 border-amber-600 hover:bg-amber-400/10 hover:text-amber-600 bg-amber-400/10"
          onClick={() => createBackup.mutate()}
          isLoading={createBackup.isPending}
          disabled={createBackup.isPending}
          loadingText="Creating backup..."
        >
          <Icon icon={faDownload} className="size-4 mb-0.5" />
          <span>Create Backup</span>
        </Button>
      </div>
    </div>
  );
});
BackupAlert.displayName = "BackupAlert";
