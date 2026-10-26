import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { ediTableGraphQLConfigs } from "@/lib/graphql/edi-table";
import type { EDIPartner } from "@trenova/shared/types/edi";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { useNavigate } from "react-router";
import { getPartnerColumns } from "../../_components/edi-partner-columns";

/** The partners, each row opening that partner's project in the Lading studio. */
export default function LadingPartnerTable() {
  const t = useT();
  const navigate = useNavigate();
  const columns = useMemo(() => getPartnerColumns(t), [t]);

  return (
    <DataTable<EDIPartner>
      name="EDI Partner"
      emptyTitle={t("No EDI partners yet")}
      queryKey="edi-lading-partner-list"
      resource={Resource.EDI}
      columns={columns}
      graphql={ediTableGraphQLConfigs.partners}
      enableCreateAction={false}
      onRowClick={(row) => navigate(`/edi/lading/${row.original.id}`)}
    />
  );
}
