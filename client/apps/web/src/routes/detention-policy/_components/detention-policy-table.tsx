import { DataTable } from "@/components/data-table/data-table";
import { detentionPolicyTableGraphQLConfig } from "@/lib/graphql/detention-policy-table";
import type { DetentionPolicy } from "@trenova/shared/types/detention";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./detention-policy-columns";
import { DetentionPolicyPanel } from "./detention-policy-panel";

export default function DetentionPolicyTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<DetentionPolicy>
      name="Detention Policy"
      queryKey="detention-policy-list"
      graphql={detentionPolicyTableGraphQLConfig}
      resource={Resource.DetentionPolicy}
      columns={columns}
      TablePanel={DetentionPolicyPanel}
    />
  );
}
