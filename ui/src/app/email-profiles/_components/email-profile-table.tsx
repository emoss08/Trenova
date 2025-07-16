import { DataTable } from "@/components/data-table/data-table";
import type { EmailProfileSchema } from "@/lib/schemas/email-profile-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./email-profile-columns";
import { CreateEmailProfileModal } from "./email-profile-create-modal";
import { EditEmailProfileModal } from "./email-profile-edit-modal";

export default function EmailProfileTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<EmailProfileSchema>
      resource={Resource.EmailProfile}
      name="Email Profile"
      link="/email-profiles/"
      queryKey="email-profile-list"
      exportModelName="email-profile"
      TableModal={CreateEmailProfileModal}
      TableEditModal={EditEmailProfileModal}
      columns={columns}
      config={{
        enableFiltering: true,
        enableSorting: true,
        enableMultiSort: true,
        maxFilters: 5,
        maxSorts: 3,
        searchDebounce: 300,
        showFilterUI: true,
        showSortUI: true,
      }}
      useEnhancedBackend={true}
    />
  );
}
