/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { DataTable } from "@/components/data-table/data-table";
import { type AccessorialChargeSchema } from "@/lib/schemas/accessorial-charge-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./accessorial-charge-columns";
import { CreateAccessorialChargeModal } from "./accessorial-charge-create-modal";
import { EditAccessorialChargeModal } from "./accessorial-charge-edit-modal";

export default function AccessorialChargeTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<AccessorialChargeSchema>
      resource={Resource.AccessorialCharge}
      name="Accessorial Charge"
      link="/accessorial-charges/"
      exportModelName="accessorial-charge"
      queryKey="accessorial-charge-list"
      TableModal={CreateAccessorialChargeModal}
      TableEditModal={EditAccessorialChargeModal}
      columns={columns}
    />
  );
}
