import { DataTable } from "@/components/data-table/data-table";
import {
  LazyImportRemittanceDialog,
  preloadImportRemittanceDialog,
} from "@/components/factoring/import-remittance-dialog-lazy";
import {
  createFactoringRemittanceTableConfig,
  type FactoringRemittanceRow,
} from "@/lib/graphql/factoring-remittance";
import { FACTORING_REMITTANCE_KEY } from "@/lib/queries/factoring-roots";
import { Upload01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
import { getFactoringRemittanceColumns } from "./factoring-remittance-columns";

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
            <Button
              size="sm"
              onPointerEnter={preloadImportRemittanceDialog}
              onFocus={preloadImportRemittanceDialog}
              onClick={() => setImporting(true)}
              className="gap-1.5"
            >
              <Upload01Icon className="size-3.5" />
              {t("Import remittance")}
            </Button>
          ),
        }}
      />
      <LazyImportRemittanceDialog open={importing} onOpenChange={setImporting} />
    </>
  );
}
