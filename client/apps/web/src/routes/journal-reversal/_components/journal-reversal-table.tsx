import { DataTable } from "@/components/data-table/data-table";
import {
  journalReversalTableGraphQLConfig,
  type JournalReversalRow,
} from "@/lib/graphql/journal-reversal-table";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./journal-reversal-columns";
import { JournalReversalPanel } from "./journal-reversal-panel";

export default function JournalReversalTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<JournalReversalRow>
      name="Journal Reversal"
      queryKey="journal-reversal-list"
      graphql={journalReversalTableGraphQLConfig}
      resource={Resource.JournalReversal}
      columns={columns}
      TablePanel={JournalReversalPanel}
    />
  );
}
