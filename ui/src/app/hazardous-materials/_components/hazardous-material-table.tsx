import { DataTable } from "@/components/data-table/data-table";
import { type HazardousMaterialSchema } from "@/lib/schemas/hazardous-material-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./hazardous-material-columns";
import { CreateHazardousMaterialModal } from "./hazardous-material-create-modal";
import { EditHazardousMaterialModal } from "./hazardous-material-edit-modal";

export default function HazardousMaterialTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<HazardousMaterialSchema>
      resource={Resource.HazardousMaterial}
      name="Hazardous Material"
      link="/hazardous-materials/"
      exportModelName="hazardous-material"
      queryKey="hazardous-material-list"
      TableModal={CreateHazardousMaterialModal}
      TableEditModal={EditHazardousMaterialModal}
      columns={columns}
    />
  );
}
