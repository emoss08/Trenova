import { DataTable } from "@/components/data-table/data-table";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import type { ManualJournal } from "@/types/manual-journal";
import { Resource } from "@/types/permission";
import { useMemo } from "react";
import { getManualJournalColumns } from "./_components/manual-journal-columns";
import { ManualJournalPanel } from "./_components/manual-journal-panel";

export function ManualJournalsPage() {
  const columns = useMemo(() => getManualJournalColumns(), []);

  return (
    <PageLayout
      pageHeaderProps={{
        title: "Manual Journals",
        description: "Create and manage manual journal entries.",
      }}
    >
      <div className="mx-4 mt-3 mb-4">
        <DataTable<ManualJournal>
          exportModelName="ManualJournal"
          name="ManualJournal"
          link="/accounting/manual-journals/"
          queryKey="manual-journal-list"
          columns={columns}
          resource={Resource.ManualJournal}
          TablePanel={ManualJournalPanel}
        />
      </div>
    </PageLayout>
  );
}
