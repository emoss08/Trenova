import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { factoringCompanyTableGraphQLConfig } from "@/lib/graphql/factoring-company-table";
import type { FactoringCompany } from "@/types/factoring-company";
import { Resource } from "@trenova/shared/types/permission";
import { useNavigate } from "react-router";
import { FACTORING_COMPANIES_PATH, factoringCompanyPath } from "../use-factoring-company-state";
import { getColumns } from "./factoring-company-columns";

export default function FactoringCompanyTable() {
  const t = useT();
  const navigate = useNavigate();
  const columns = getColumns(t);

  return (
    <DataTable<FactoringCompany>
      name="Factoring Company"
      emptyTitle={t("No factoring companies yet")}
      queryKey="factoring-company-list"
      graphql={factoringCompanyTableGraphQLConfig}
      resource={Resource.FactoringCompany}
      columns={columns}
      onAddRecord={() => void navigate(`${FACTORING_COMPANIES_PATH}/new`)}
      onRowClick={(row) => {
        if (row.original.id) void navigate(factoringCompanyPath(row.original.id));
      }}
    />
  );
}
