import { Kept } from "@/components/kept";
import {
  CLOSED_WORKER_OPEN_STATE as CLOSED,
  type WorkerTabProps,
} from "@/routes/worker/use-worker-state";
import { lazyComponent } from "@/lib/lazy-component";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { WORKER_VIEW_MODULES } from "../worker-view-modules";
import { QualificationSkeleton } from "./qualification-skeleton";

const Credentials = lazyComponent(WORKER_VIEW_MODULES.credentials, (module) => module.default);
const DQFile = lazyComponent(WORKER_VIEW_MODULES["dq-file"], (module) => module.default);
const DrugAlcohol = lazyComponent(
  WORKER_VIEW_MODULES["drug-and-alcohol"],
  (module) => module.default,
);

/**
 * The Qualification tab: the shell draws its section strip; this draws the selected view, each
 * kept mounted after it is first shown, and hands each the address's open pane.
 */
export default function QualificationTab({ workerId, view, open, onOpen }: WorkerTabProps) {
  const selected = view ?? "credentials";
  return (
    <>
      <Kept shown={selected === "credentials"}>
        <LazyComponent fallback={<QualificationSkeleton view="credentials" />}>
          <Credentials.Component
            workerId={workerId}
            open={selected === "credentials" ? open : CLOSED}
            onOpen={onOpen}
          />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "dq-file"}>
        <LazyComponent fallback={<QualificationSkeleton view="dq-file" />}>
          <DQFile.Component
            workerId={workerId}
            open={selected === "dq-file" ? open : CLOSED}
            onOpen={onOpen}
          />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "drug-and-alcohol"}>
        <LazyComponent fallback={<QualificationSkeleton view="drug-and-alcohol" />}>
          <DrugAlcohol.Component
            workerId={workerId}
            open={selected === "drug-and-alcohol" ? open : CLOSED}
            onOpen={onOpen}
          />
        </LazyComponent>
      </Kept>
    </>
  );
}
