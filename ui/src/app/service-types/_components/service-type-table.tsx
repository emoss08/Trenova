/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { DataTable } from "@/components/data-table/data-table";
import type { ServiceTypeSchema } from "@/lib/schemas/service-type-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./service-type-columns";
import { CreateServiceTypeModal } from "./service-type-create-modal";
import { EditServiceTypeModal } from "./service-type-edit-modal";

export default function ServiceTypesDataTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<ServiceTypeSchema>
      resource={Resource.ServiceType}
      name="Service Type"
      link="/service-types/"
      queryKey="service-type-list"
      exportModelName="service-type"
      TableModal={CreateServiceTypeModal}
      TableEditModal={EditServiceTypeModal}
      columns={columns}
    />
  );
}
