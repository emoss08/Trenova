import { DataTable } from "@/components/data-table/data-table";
import { authEventTableGraphQLConfig, type AuthEventRow } from "@/lib/graphql/security-activity-table";
import { useT } from "@trenova/shared/i18n/use-t";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { authEventColumns } from "./activity-columns";

const INITIAL_VISIBILITY = { userAgent: false };

export default function AuthEventsTable() {
  const t = useT();
  const columns = useMemo(() => authEventColumns(t), [t]);

  return (
    <DataTable<AuthEventRow>
      name="Auth Event"
      emptyTitle={t("No sign-ins recorded yet")}
      queryKey="security-auth-events"
      graphql={authEventTableGraphQLConfig}
      resource={Resource.AuthEvent}
      columns={columns}
      initialColumnVisibility={INITIAL_VISIBILITY}
      enableCreateAction={false}
    />
  );
}
