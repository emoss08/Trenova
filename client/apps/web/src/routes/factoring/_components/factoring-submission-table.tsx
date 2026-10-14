import { DataTable } from "@/components/data-table/data-table";
import {
  createFactoringSubmissionTableConfig,
  FACTORING_SUBMISSION_KEY,
  type FactoringSubmissionRow,
} from "@/lib/graphql/factoring";
import { useT } from "@trenova/shared/i18n/use-t";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getFactoringSubmissionColumns } from "./factoring-submission-columns";

const ALL_STATUSES = null;

export default function FactoringSubmissionTable() {
  const t = useT();
  const columns = useMemo(() => getFactoringSubmissionColumns(t), [t]);
  const graphql = useMemo(() => createFactoringSubmissionTableConfig(ALL_STATUSES), []);

  return (
    <DataTable<FactoringSubmissionRow>
      name="Factoring schedule"
      emptyTitle={t("No schedules have been sent to a factor")}
      queryKey={FACTORING_SUBMISSION_KEY}
      graphql={graphql}
      resource={Resource.FactoringSubmission}
      columns={columns}
      enableCreateAction={false}
    />
  );
}
