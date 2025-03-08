import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import AuditLogTable from "./_components/audit-log-table";

export function AuditLogs() {
  return (
    <div className="flex flex-col space-y-6">
      <MetaTags
        title="Audit Entries"
        description="View and manage audit entries"
      />
      <Header />

      <SuspenseLoader>
        <AuditLogTable />
      </SuspenseLoader>
    </div>
  );
}

function Header() {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Audit Entries</h1>
        <p className="text-muted-foreground">
          Monitor, track, and analyze system activities and user actions
        </p>
      </div>
    </div>
  );
}
