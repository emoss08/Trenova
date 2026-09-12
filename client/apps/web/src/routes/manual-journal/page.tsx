import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import {
  manualJournalTableGraphQLConfig,
  type ManualJournalRow,
} from "@/lib/graphql/manual-journal-table";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getManualJournalColumns } from "./_components/manual-journal-columns";
import { ManualJournalPanel } from "./_components/manual-journal-panel";

export function ManualJournalsPage() {
  const t = useT();

  const columns = useMemo(() => getManualJournalColumns(t), [t]);

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Manual Journals"),
        description: t("Create and manage manual journal entries."),
      }}
      className="p-0"
    >
      <div className="mx-4 mt-3 mb-4">
        <DataTable<ManualJournalRow>
          name="ManualJournal"
          queryKey="manual-journal-list"
          columns={columns}
          resource={Resource.ManualJournal}
          graphql={manualJournalTableGraphQLConfig}
          TablePanel={ManualJournalPanel}
        />
      </div>
    </PageLayout>
  );
}
