import { Kept } from "@/components/kept";
import type { WorkerTabProps } from "@/routes/worker/use-worker-state";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { PaySkeleton } from "./pay-skeleton";

const PayProfile = lazy(() => import("./pay-profile/worker-pay-tab"));
const Benefits = lazy(() =>
  import("./benefits/worker-benefits-section").then((module) => ({
    default: module.WorkerBenefitsSection,
  })),
);

export default function PayTab({ workerId, worker, view, open, onOpen }: WorkerTabProps) {
  const selected = view ?? "pay-profile";
  return (
    <>
      <Kept shown={selected === "pay-profile"}>
        <LazyComponent fallback={<PaySkeleton view="pay-profile" />}>
          <PayProfile workerId={workerId} worker={worker} open={open} onOpen={onOpen} />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "benefits"}>
        <LazyComponent fallback={<PaySkeleton view="benefits" />}>
          <Benefits workerId={workerId} open={open} onOpen={onOpen} />
        </LazyComponent>
      </Kept>
    </>
  );
}
