import { DataTable } from "@/components/data-table/data-table";
import {
  customerPaymentTableGraphQLConfig,
  type CustomerPaymentRow,
} from "@/lib/graphql/customer-payment";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { CustomerPaymentPanel } from "./customer-payment-panel";
import { getColumns } from "./payment-columns";

export default function PaymentsTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<CustomerPaymentRow>
      name="Customer Payment"
      queryKey="customer-payment-list"
      graphql={customerPaymentTableGraphQLConfig}
      resource={Resource.CustomerPayment}
      columns={columns}
      TablePanel={CustomerPaymentPanel}
    />
  );
}
