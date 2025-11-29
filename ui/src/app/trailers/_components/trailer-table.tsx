import { DataTable } from "@/components/data-table/data-table";
import { type TrailerSchema } from "@/lib/schemas/trailer-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./trailer-columns";
import { CreateTrailerModal } from "./trailer-create-modal";
import { EditTrailerModal } from "./trailer-edit-modal";

export default function TrailerTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<TrailerSchema>
      resource={Resource.Trailer}
      name="Trailer"
      link="/trailers/"
      extraSearchParams={{
        includeEquipmentDetails: true,
        includeFleetDetails: true,
      }}
      queryKey="trailer-list"
      exportModelName="trailer"
      TableModal={CreateTrailerModal}
      TableEditModal={EditTrailerModal}
      columns={columns}
    />
  );
}
