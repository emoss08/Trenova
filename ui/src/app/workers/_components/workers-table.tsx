import { DataTable } from "@/components/data-table/data-table";
import { type WorkerSchema } from "@/lib/schemas/worker-schema";
import { useMemo } from "react";
import { CreateWorkerModal } from "./workers-create-modal";
import { EditWorkerModal } from "./workers-edit-modal";
import { getColumns } from "./workers-table-columns";

export default function WorkersDataTable() {
  const columns = useMemo(() => getColumns(), []);

  /**
   * Advanced filter fields for the data table.
   * These fields provide more complex filtering options compared to the regular filterFields.
   *
   * Key differences from regular filterFields:
   * 1. More field types: Includes 'text', 'multi-select', 'date', and 'boolean'.
   * 2. Enhanced flexibility: Allows for more precise and varied filtering options.
   * 3. Used with DataTableAdvancedToolbar: Enables a more sophisticated filtering UI.
   * 4. Date and boolean types: Adds support for filtering by date ranges and boolean values.
   */
  // const advancedFilterFields: DataTableAdvancedFilterField<WorkerSchema>[] = [
  //   {
  //     id: "status",
  //     label: "Status",
  //     type: "multi-select",
  //     options: Object.values(Status).map((status) => ({
  //       label: toTitleCase(status),
  //       value: status,
  //       // icon: getStatusColor(status),
  //       // count: statusCounts[status],
  //     })),
  //   },
  //   {
  //     id: "profile.endorsement" as keyof WorkerSchema,
  //     label: "Endorsement",
  //     type: "multi-select",
  //     options: Object.values(Endorsement).map((endorsement) => ({
  //       label: mapToEndorsement(endorsement),
  //       value: endorsement,
  //     })),
  //   },
  //   {
  //     id: "type",
  //     label: "Type",
  //     type: "multi-select",
  //     options: Object.values(WorkerType).map((type) => ({
  //       label: toTitleCase(type),
  //       value: type,
  //       // icon: getPriorityIcon(priority),
  //       // count: priorityCounts[priority],
  //     })),
  //   },
  //   {
  //     id: "profile.licenseExpiry" as keyof WorkerSchema,
  //     label: "License Expiry",
  //     type: "date",
  //   },
  //   {
  //     id: "createdAt",
  //     label: "Created at",
  //     type: "date",
  //   },
  // ];

  return (
    <DataTable<WorkerSchema>
      extraSearchParams={{
        includeProfile: "true",
        includePTO: "true",
      }}
      TableModal={CreateWorkerModal}
      TableEditModal={EditWorkerModal}
      queryKey="worker-list"
      name="Worker"
      link="/workers/"
      columns={columns}
    />
  );
}
