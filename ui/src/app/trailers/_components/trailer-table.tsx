import { DataTable } from "@/components/data-table/data-table";
import { type TrailerSchema } from "@/lib/schemas/trailer-schema";
import { useMemo } from "react";
import { getColumns } from "./trailer-columns";
import { CreateTrailerModal } from "./trailer-create-modal";
import { EditTrailerModal } from "./trailer-edit-modal";

export default function TrailerTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<TrailerSchema>
      name="Trailer"
      link="/trailers/"
      extraSearchParams={{
        includeEquipmentDetails: true,
      }}
      queryKey="trailer-list"
      exportModelName="trailer"
      TableModal={CreateTrailerModal}
      TableEditModal={EditTrailerModal}
      columns={columns}
    />
  );
}
