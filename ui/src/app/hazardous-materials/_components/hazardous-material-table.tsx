/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
