import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import AccessorialChargeTable from "./_components/accessorial-charge-table";

export function AccessorialCharges() {
  return (
    <>
      <MetaTags title="Accessorial Charges" description="Accessorial Charges" />
      <div className="flex flex-col gap-y-3">
        <Header />
        <LazyComponent>
          <AccessorialChargeTable />
        </LazyComponent>
      </div>
    </>
  );
}

function Header() {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">
          Accessorial Charges
        </h1>
        <p className="text-muted-foreground">
          Manage and configure accessorial charges for your organization
        </p>
      </div>
    </div>
  );
}
