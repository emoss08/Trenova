import { DataTable } from "@/components/data-table/data-table";
import { recordPath } from "@/config/record-links";
import { usePermission } from "@/hooks/use-permission";
import { useJournalReviewActions } from "@/hooks/use-journal-review-actions";
import { useJournalReviewLabels } from "@/hooks/use-journal-review-labels";
import {
  JOURNAL_REVIEW_TABLE_KEY,
  journalReviewTableGraphQLConfig,
  type JournalReviewRow,
} from "@/lib/graphql/journal-review";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DockAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { BookCheckIcon, CheckIcon } from "lucide-react";
import { useMemo } from "react";
import { toast } from "sonner";
import { useNavigate } from "react-router";
import { getJournalReviewColumns } from "./review-columns";

function idsWithStatus(rows: JournalReviewRow[], status: string): string[] {
  return rows.filter((row) => row.status === status).map((row) => row.id);
}

export default function JournalReviewTable() {
  const t = useT();
  const navigate = useNavigate();
  const labels = useJournalReviewLabels();
  const actions = useJournalReviewActions();
  const { allowed: canApprove } = usePermission(Resource.JournalEntry, Operation.Approve);
  const columns = useMemo(() => getJournalReviewColumns(t, labels), [t, labels]);

  const dockActions = useMemo<DockAction<JournalReviewRow>[]>(() => {
    if (!canApprove) {
      return [];
    }
    return [
      {
        id: "approve",
        label: t("Approve"),
        loadingLabel: t("Approving..."),
        icon: CheckIcon,
        clearSelectionOnSuccess: true,
        onClick: async (rows) => {
          const ids = idsWithStatus(rows, "Pending");
          if (ids.length === 0) {
            toast.info(t("None of the selected entries is awaiting approval."));
            return;
          }
          await actions.approve.mutateAsync(ids);
        },
      },
      {
        id: "post",
        label: t("Post to ledger"),
        loadingLabel: t("Posting..."),
        icon: BookCheckIcon,
        clearSelectionOnSuccess: true,
        onClick: async (rows) => {
          const ids = idsWithStatus(rows, "Approved");
          if (ids.length === 0) {
            toast.info(t("None of the selected entries is approved yet."));
            return;
          }
          await actions.post.mutateAsync(ids);
        },
      },
    ];
  }, [actions.approve, actions.post, canApprove, t]);

  return (
    <DataTable<JournalReviewRow>
      name="Journal entry"
      queryKey={JOURNAL_REVIEW_TABLE_KEY}
      graphql={journalReviewTableGraphQLConfig}
      resource={Resource.JournalEntry}
      columns={columns}
      dockActions={dockActions}
      enableRowSelection={canApprove}
      enableCreateAction={false}
      onRowClick={(row) => void navigate(recordPath("journal_entry", row.original.id))}
    />
  );
}
