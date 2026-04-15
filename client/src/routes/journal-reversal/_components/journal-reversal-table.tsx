import { DataTable } from "@/components/data-table/data-table";
import type { JournalReversal } from "@/types/journal-reversal";
import { Resource } from "@/types/permission";
import { useMemo } from "react";
import { getColumns } from "./journal-reversal-columns";
import { JournalReversalPanel } from "./journal-reversal-panel";

export default function JournalReversalTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<JournalReversal>
      name="Journal Reversal"
      link="/accounting/journal-reversals/"
      queryKey="journal-reversal-list"
      exportModelName="journal-reversal"
      resource={Resource.JournalReversal}
      columns={columns}
      TablePanel={JournalReversalPanel}
      preferDetailRowForEdit
    />
  );
}
