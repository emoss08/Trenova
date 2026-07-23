import { DataTable } from "@/components/data-table/data-table";
import {
  driverExpenseTableGraphQLConfig,
  type DriverExpenseRow,
} from "@/lib/graphql/driver-portal";
import { Resource } from "@/types/permission";
import { useMemo } from "react";
import { getColumns } from "./expense-columns";
import { ExpensePanel } from "./expense-panel";

export default function ExpensesTable() {
  const columns = useMemo(() => getColumns(), []);

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
