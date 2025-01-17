import { DataTable } from "@/components/data-table/data-table";
import { type HazardousMaterialSchema } from "@/lib/schemas/hazardous-material-schema";
import { useMemo } from "react";
import { getColumns } from "./hazardous-material-columns";
import { CreateHazardousMaterialModal } from "./hazardous-material-create-modal";
import { EditHazardousMaterialModal } from "./hazardous-material-edit-modal";

export default function HazardousMaterialTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<HazardousMaterialSchema>
      name="Hazardous Material"
      link="/hazardous-materials/"
      queryKey="hazardous-material-list"
      TableModal={CreateHazardousMaterialModal}
      TableEditModal={EditHazardousMaterialModal}
      columns={columns}
    />
  );
}
