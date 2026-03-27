import { DataTable } from "@/components/data-table/data-table";
import { Resource } from "@/types/permission";
import type { TCASubscription } from "@/types/table-change-alert";
import { useMemo } from "react";
import { getColumns } from "./subscription-columns";
import { SubscriptionPanel } from "./subscription-panel";

export default function SubscriptionTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<TCASubscription>
      name="Subscription"
      link="/tca/subscriptions/"
      queryKey="tca-subscription-list"
      exportModelName="tca-subscription"
      resource={Resource.TableChangeAlert}
      columns={columns}
      TablePanel={SubscriptionPanel}
    />
  );
}
