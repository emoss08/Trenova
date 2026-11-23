import { Kept } from "@/components/kept";
import { lazyComponent } from "@/lib/lazy-component";
import {
  CLOSED_WORKER_OPEN_STATE as CLOSED,
  type WorkerTabProps,
} from "@/routes/worker/use-worker-state";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { WORKER_VIEW_MODULES } from "../worker-view-modules";
import { PaySkeleton } from "./pay-skeleton";

const PayProfile = lazyComponent(WORKER_VIEW_MODULES["pay-profile"], (module) => module.default);
const Benefits = lazyComponent(
  WORKER_VIEW_MODULES.benefits,
  (module) => module.WorkerBenefitsSection,
);

export default function PayTab({ workerId, worker, view, open, onOpen }: WorkerTabProps) {
  const selected = view ?? "pay-profile";
  return (
    <>
      <Kept shown={selected === "pay-profile"}>
        <LazyComponent fallback={<PaySkeleton view="pay-profile" />}>
          <PayProfile.Component
            workerId={workerId}
            worker={worker}
            open={selected === "pay-profile" ? open : CLOSED}
            onOpen={onOpen}
          />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "benefits"}>
        <LazyComponent fallback={<PaySkeleton view="benefits" />}>
          <Benefits.Component
            workerId={workerId}
            open={selected === "benefits" ? open : CLOSED}
            onOpen={onOpen}
          />
        </LazyComponent>
      </Kept>
    </>
  );
}
