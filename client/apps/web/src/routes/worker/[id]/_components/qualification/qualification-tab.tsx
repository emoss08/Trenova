import { Kept } from "@/components/kept";
import {
  CLOSED_WORKER_OPEN_STATE as CLOSED,
  type WorkerTabProps,
} from "@/routes/worker/use-worker-state";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { QualificationSkeleton } from "./qualification-skeleton";

const Credentials = lazy(() => import("./credentials/worker-credentials-tab"));
const DQFile = lazy(() => import("./dq-file/worker-dqf-tab"));
const DrugAlcohol = lazy(() => import("./drug-alcohol/worker-testing-tab"));

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
          <Credentials
            workerId={workerId}
            open={selected === "credentials" ? open : CLOSED}
            onOpen={onOpen}
          />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "dq-file"}>
        <LazyComponent fallback={<QualificationSkeleton view="dq-file" />}>
          <DQFile
            workerId={workerId}
            open={selected === "dq-file" ? open : CLOSED}
            onOpen={onOpen}
          />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "drug-and-alcohol"}>
        <LazyComponent fallback={<QualificationSkeleton view="drug-and-alcohol" />}>
          <DrugAlcohol
            workerId={workerId}
            open={selected === "drug-and-alcohol" ? open : CLOSED}
            onOpen={onOpen}
          />
        </LazyComponent>
      </Kept>
    </>
  );
}
