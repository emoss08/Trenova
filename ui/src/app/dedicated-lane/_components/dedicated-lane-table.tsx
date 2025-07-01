import { DataTable } from "@/components/data-table/data-table";
import type { DedicatedLaneSchema } from "@/lib/schemas/dedicated-lane-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { CreateDedicatedLaneModal } from "./dedated-lane-create-modal";
import { getColumns } from "./dedicated-lane-columns";
import { EditDedicatedLaneModal } from "./dedicated-lane-edit-modal";

export default function DedicatedLaneTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<DedicatedLaneSchema>
      resource={Resource.DedicatedLane}
      name="Dedicated Lane"
      link="/dedicated-lanes/"
      extraSearchParams={{
        expandDetails: true,
      }}
      queryKey="dedicated-lane-list"
      exportModelName="dedicated-lane"
      TableModal={CreateDedicatedLaneModal}
      TableEditModal={EditDedicatedLaneModal}
      columns={columns}
    />
  );
}
