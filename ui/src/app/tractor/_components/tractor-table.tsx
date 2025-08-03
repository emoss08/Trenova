/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { DataTable } from "@/components/data-table/data-table";
import type { TractorSchema } from "@/lib/schemas/tractor-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./tractor-columns";
import { CreateTractorModal } from "./tractor-create-modal";
import { EditTractorModal } from "./tractor-edit-modal";

export default function TractorTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<TractorSchema>
      resource={Resource.Tractor}
      name="Tractor"
      link="/tractors/"
      extraSearchParams={{
        includeWorkerDetails: true,
        includeEquipmentDetails: true,
        includeFleetDetails: true,
      }}
      queryKey="tractor-list"
      exportModelName="tractor"
      TableModal={CreateTractorModal}
      TableEditModal={EditTractorModal}
      columns={columns}
    />
  );
}
