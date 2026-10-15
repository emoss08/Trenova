import { DataTable } from "@/components/data-table/data-table";
import { FACTORING_REMITTANCE_KEY } from "@/lib/graphql/factoring";
import {
  createFactoringRemittanceTableConfig,
  type FactoringRemittanceRow,
} from "@/lib/graphql/factoring-remittance";
import { Upload01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
import { getFactoringRemittanceColumns } from "./factoring-remittance-columns";
import { ImportRemittanceDialog } from "./import-remittance-dialog";

const ALL_FACTORS = null;

export default function FactoringRemittanceTable() {
  const t = useT();
  const [importing, setImporting] = useState(false);
  const columns = useMemo(() => getFactoringRemittanceColumns(t), [t]);
  const graphql = useMemo(() => createFactoringRemittanceTableConfig(ALL_FACTORS), []);

  return (
    <>
      <DataTable<FactoringRemittanceRow>
        name="Factor remittance"
        emptyTitle={t("No remittances have been imported")}
        queryKey={FACTORING_REMITTANCE_KEY}
        graphql={graphql}
        resource={Resource.FactoringRemittance}
        columns={columns}
        enableCreateAction={false}
        toolbar={{
          end: (
            <Button size="sm" onClick={() => setImporting(true)} className="gap-1.5">
              <Upload01Icon className="size-3.5" />
              {t("Import remittance")}
            </Button>
          ),
        }}
      />
      <ImportRemittanceDialog open={importing} onOpenChange={setImporting} />
    </>
  );
}
