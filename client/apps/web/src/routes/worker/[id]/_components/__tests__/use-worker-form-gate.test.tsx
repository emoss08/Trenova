import { permissionManifest } from "@/test/permissions";
import type { WorkerForm } from "@/routes/worker/use-worker-state";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  useWorkerFormGate,
  WorkerFormDropContext,
  type WorkerFormDecision,
} from "../use-worker-form-gate";

function gate(
  form: WorkerForm | null,
  decisions: Partial<Record<WorkerForm, WorkerFormDecision>>,
) {
  const drop = vi.fn();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <WorkerFormDropContext value={drop}>{children}</WorkerFormDropContext>
  );
  const hook = renderHook(
    ({ named, said }: { named: WorkerForm | null; said: typeof decisions }) =>
      useWorkerFormGate(named, said),
    { wrapper, initialProps: { named: form, said: decisions } },
  );
  return { drop, ...hook };
}

function signIn() {
  usePermissionStore.setState({
    manifest: permissionManifest({}),
    lastFetched: Date.now(),
    isLoading: false,
  });
}

afterEach(() => {
  usePermissionStore.setState({ manifest: null });
});

describe("useWorkerFormGate", () => {
  it("opens the named form when its tab says it can, and only that one", () => {
    signIn();
    const { result, drop } = gate("answer-query", { "answer-query": true, "log-query": true });
    expect(result.current("answer-query")).toBe(true);
    expect(result.current("log-query")).toBe(false);
    expect(drop).not.toHaveBeenCalled();
  });

  it("takes a form its tab refuses out of the address", () => {
    signIn();
    const { result, drop } = gate("edit-review", { "edit-review": false });
    expect(result.current("edit-review")).toBe(false);
    expect(drop).toHaveBeenCalledOnce();
  });

  it("keeps the form while its reads are out, and drops it once they refuse it", () => {
    signIn();
    const { result, drop, rerender } = gate("record-training-result", {
      "record-training-result": "waiting",
    });
    expect(result.current("record-training-result")).toBe(false);
    expect(drop).not.toHaveBeenCalled();

    rerender({ named: "record-training-result", said: { "record-training-result": false } });
    expect(drop).toHaveBeenCalledOnce();
  });

  it("refuses nothing before the reader's permissions are known", () => {
    const { drop } = gate("assign-pay-profile", { "assign-pay-profile": false });
    expect(drop).not.toHaveBeenCalled();
  });

  it("leaves a form another tab owns alone", () => {
    signIn();
    const { drop } = gate("upload-document", { "assign-pay-profile": false });
    expect(drop).not.toHaveBeenCalled();
  });
});
