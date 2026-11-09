import { usePermission } from "@/hooks/use-permission";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { useQuery } from "@tanstack/react-query";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { FactoringRemittancesSection } from "./factoring-remittances-section";
import { FactoringReserveChart } from "./factoring-reserve-chart";
import { FactoringReserveSection } from "./factoring-reserve-section";

/**
 * What the factor holds and has paid: the reserve and its movements as booked, how the reserve
 * held moved over the last days, and the funding and reserve reports the factor sent.
 */
export default function FactoringLedgerTab({ factoringCompanyId }: { factoringCompanyId: string }) {
  const { allowed: canReadReserve } = usePermission(Resource.FactoringReserve, Operation.Read);
  const overview = useQuery(factoringQueries.overview(factoringCompanyId));

  return (
    <div className="flex flex-col gap-4">
      {canReadReserve ? (
        <FactoringReserveChart
          factoringCompanyId={factoringCompanyId}
          currencyCode={overview.data?.currencyCode ?? "USD"}
        />
      ) : null}
      <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_26rem]">
        <FactoringReserveSection factoringCompanyId={factoringCompanyId} />
        <FactoringRemittancesSection factoringCompanyId={factoringCompanyId} />
      </div>
    </div>
  );
}
