import { DataTable } from "@/components/data-table/data-table";
import { type FleetCodeSchema } from "@/lib/schemas/fleet-code-schema";
import { useMemo } from "react";
import { getColumns } from "./fleet-code-columns";
import { CreateFleetCodeModal } from "./fleet-code-create-modal";
import { EditFleetCodeModal } from "./fleet-code-edit-modal";

export default function FleetCodesDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<FleetCodeSchema>
      name="Fleet Code"
      link="/fleet-codes/"
      queryKey={["fleet-code"]}
      exportModelName="fleet-code"
      TableModal={CreateFleetCodeModal}
      TableEditModal={EditFleetCodeModal}
      columns={columns}
    />
  );
}
