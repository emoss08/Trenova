import { DataTable } from "@/components/data-table/data-table";
import { routingGuideTableGraphQLConfig } from "@/lib/graphql/routing-guide-table";
import { Resource } from "@trenova/shared/types/permission";
import type { RoutingGuide } from "@trenova/shared/types/routing-guide";
import { useMemo } from "react";
import { getColumns } from "./routing-guide-columns";
import { RoutingGuidePanel } from "./routing-guide-panel";

export default function RoutingGuideTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<RoutingGuide>
      name="Routing Guide"
      queryKey="routing-guide-list"
      resource={Resource.RoutingGuide}
      columns={columns}
      TablePanel={RoutingGuidePanel}
      graphql={routingGuideTableGraphQLConfig}
    />
  );
}
