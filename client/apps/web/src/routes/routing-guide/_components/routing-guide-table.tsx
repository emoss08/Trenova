import { DataTable } from "@/components/data-table/data-table";
import {
  routingGuideTableGraphQLConfig,
  type RoutingGuideRow,
} from "@/lib/graphql/routing-guide-table";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getColumns } from "./routing-guide-columns";
import { RoutingGuidePanel } from "./routing-guide-panel";

export default function RoutingGuideTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<RoutingGuideRow>
      name="Routing Guide"
      queryKey="routing-guide-list"
      resource={Resource.RoutingGuide}
      columns={columns}
      TablePanel={RoutingGuidePanel}
      graphql={routingGuideTableGraphQLConfig}
    />
  );
}
