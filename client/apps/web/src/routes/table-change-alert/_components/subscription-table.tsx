import { DataTable } from "@/components/data-table/data-table";
import {
  tcaSubscriptionTableGraphQLConfig,
  type TCASubscriptionRow,
} from "@/lib/graphql/table-change-alert-table";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./subscription-columns";
import { SubscriptionPanel } from "./subscription-panel";

export default function SubscriptionTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<TCASubscriptionRow>
      name="Subscription"
      queryKey="tca-subscription-list"
      graphql={tcaSubscriptionTableGraphQLConfig}
      resource={Resource.TableChangeAlert}
      columns={columns}
      TablePanel={SubscriptionPanel}
    />
  );
}
