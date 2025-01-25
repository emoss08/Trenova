import { DataTable } from "@/components/data-table/data-table";
import { type Tractor } from "@/types/tractor";
import { useMemo } from "react";
import { getColumns } from "./tractor-columns";
import { CreateTractorModal } from "./tractor-create-modal";
import { EditTractorModal } from "./tractor-edit-modal";

export default function TractorTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<Tractor>
      name="Tractor"
      link="/tractors/"
      extraSearchParams={{
        includeWorkerDetails: true,
        includeEquipmentDetails: true,
      }}
      queryKey={["tractor"]}
      exportModelName="tractor"
      TableModal={CreateTractorModal}
      TableEditModal={EditTractorModal}
      columns={columns}
    />
  );
}
