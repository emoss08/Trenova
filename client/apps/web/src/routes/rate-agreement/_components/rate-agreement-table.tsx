import { DataTable } from "@/components/data-table/data-table";
import { rateAgreementTableGraphQLConfig } from "@/lib/graphql/rate-tables";
import type { AddRecordAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import type { RateAgreement } from "@trenova/shared/types/rate";
import { FileUpIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { getColumns } from "./rate-agreement-columns";
import { ImportRateSheetDialog } from "./import-rate-sheet-dialog";
import { RateAgreementPanel } from "./rate-agreement-panel";

export default function RateAgreementTable() {
  const columns = useMemo(() => getColumns(), []);
  const [importOpen, setImportOpen] = useState(false);

  const addRecordActions = useMemo<AddRecordAction[]>(
    () => [
      {
        id: "import-rate-sheet",
        label: "Import Rate Sheet",
        description: "Upload a CSV or XLSX rate sheet into an agreement.",
        icon: FileUpIcon,
        onClick: () => setImportOpen(true),
      },
    ],
    [],
  );

  return (
    <>
      <DataTable<RateAgreement>
        name="Rate Agreement"
        queryKey="rate-agreement-list"
        graphql={rateAgreementTableGraphQLConfig}
        resource={Resource.RateAgreement}
        columns={columns}
        TablePanel={RateAgreementPanel}
        addRecordActions={addRecordActions}
      />
      <ImportRateSheetDialog open={importOpen} onOpenChange={setImportOpen} />
    </>
  );
}
