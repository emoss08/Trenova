import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";

export function BillingQueue() {
  return (
    <>
      <MetaTags title="Billing Queue" description="Billing Queue" />
      <LazyComponent>
        <BulkTransfer />
      </LazyComponent>
    </>
  );
}

function BulkTransfer() {
  return (
    <div>
      <h1>Bulk Transfer</h1>
    </div>
  );
}
