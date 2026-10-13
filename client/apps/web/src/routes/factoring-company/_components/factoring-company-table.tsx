import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { factoringCompanyTableGraphQLConfig } from "@/lib/graphql/factoring-company-table";
import type { FactoringCompany } from "@/types/factoring-company";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./factoring-company-columns";
import { FactoringCompanyPanel } from "./factoring-company-panel";

export default function FactoringCompanyTable() {
  const t = useT();

  const columns = useMemo(() => getColumns(t), [t]);

  return (
    <DataTable<FactoringCompany>
      name="Factoring Company"
      emptyTitle={t("No factoring companies yet")}
      queryKey="factoring-company-list"
      graphql={factoringCompanyTableGraphQLConfig}
      resource={Resource.FactoringCompany}
      columns={columns}
      TablePanel={FactoringCompanyPanel}
    />
  );
}
