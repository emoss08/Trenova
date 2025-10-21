import { QueryLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { queries } from "@/lib/queries";
import { lazy, memo } from "react";
import { BackupAlert } from "./_components/backup-alert";

const BackupList = lazy(() => import("./_components/backup-list"));
const DataRetentionPolicies = lazy(
  () => import("./_components/data-retention-policies"),
);

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
      <QueryLazyComponent queryKey={queries.organization.getDataRetention._def}>
        <DataRetentionPolicies />
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
