import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import {
  driverExpenseTableGraphQLConfig,
  type DriverExpenseRow,
} from "@trenova/shared/lib/graphql/driver-portal";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./expense-columns";
import { ExpensePanel } from "./expense-panel";

export default function ExpensesTable() {
  const t = useT();

  const columns = useMemo(() => getColumns(t), [t]);

  return (
    <DataTable<DriverExpenseRow>
      name="Driver Expense"
      queryKey="driver-expense-list"
      graphql={driverExpenseTableGraphQLConfig}
      resource={Resource.DriverExpense}
      columns={columns}
      TablePanel={ExpensePanel}
      enableCreateAction={false}
    />
  );
}
