import { QueryLazyComponent } from "@/components/error-boundary";
import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { lazy, memo } from "react";

const UserTable = lazy(() => import("./_components/user-table"));

export function Users() {
  return (
    <FormSaveProvider>
      <div className="space-y-6 p-6">
        <MetaTags title="Users" description="Users" />
        <Header />
        <QueryLazyComponent queryKey={["user-list", "role-list"]}>
          <UserTable />
        </QueryLazyComponent>
      </div>
    </FormSaveProvider>
  );
}

const Header = memo(() => {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Users & Roles</h1>
        <p className="text-muted-foreground">
          Manage users & roles in your system
        </p>
      </div>
    </div>
  );
});
Header.displayName = "Header";
