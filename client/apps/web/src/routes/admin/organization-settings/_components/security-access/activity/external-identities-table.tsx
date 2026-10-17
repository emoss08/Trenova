import { DataTable } from "@/components/data-table/data-table";
import { externalIdentityTableGraphQLConfig, type ExternalIdentityRow } from "@/lib/graphql/security-activity-table";
import { useT } from "@trenova/shared/i18n/use-t";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { externalIdentityColumns } from "./activity-columns";

export default function ExternalIdentitiesTable() {
  const t = useT();
  const columns = useMemo(() => externalIdentityColumns(t), [t]);

  return (
    <DataTable<ExternalIdentityRow>
      name="External Identity"
      emptyTitle={t("No identities linked yet")}
      queryKey="security-external-identities"
      graphql={externalIdentityTableGraphQLConfig}
      resource={Resource.ExternalIdentity}
      columns={columns}
      enableCreateAction={false}
    />
  );
}
