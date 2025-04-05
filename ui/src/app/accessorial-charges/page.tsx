import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import AccessorialChargeTable from "./_components/accessorial-charge-table";

export function AccessorialCharges() {
  return (
    <>
      <MetaTags title="Accessorial Charges" description="Accessorial Charges" />
      <LazyComponent>
        <AccessorialChargeTable />
      </LazyComponent>
    </>
  );
}
