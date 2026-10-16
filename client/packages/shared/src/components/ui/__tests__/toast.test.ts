import { toast } from "@trenova/shared/components/ui/toast";
import { toastStore } from "@trenova/shared/components/ui/toast-store";
import { afterEach, describe, expect, it, vi } from "vitest";

function current() {
  return toastStore.getSnapshot().toasts;
}

function find(id: string | number) {
  return current().find((entry) => entry.id === id);
}

async function flush() {
  for (let i = 0; i < 5; i += 1) {
    await Promise.resolve();
  }
}

afterEach(() => {
  toast.dismiss();
});

describe("toast", () => {
  it("raises each type with sonner's call shape and returns the toast's id", () => {
    const action = { label: "Open", onClick: vi.fn() };
    const id = toast.success("Aging report is ready", {
      description: "AR aging · 1,284 invoices",
      action,
    });

    expect(find(id)).toMatchObject({
      type: "success",
      title: "Aging report is ready",
      description: "AR aging · 1,284 invoices",
      action,
    });
    expect(find(toast("Tracking link copied"))?.type).toBe("default");
    expect(find(toast.error("Couldn't send"))?.type).toBe("error");
    expect(find(toast.warning("Approved 46 of 48"))?.type).toBe("warning");
    expect(find(toast.info("Dry run finished"))?.type).toBe("info");
    expect(find(toast.loading("Saving 0 of 4"))?.type).toBe("loading");
  });

  it("changes a loading toast into its result when called again with its id", () => {
    const id = toast.loading("Saving 0 of 4");
    toast.loading("Saving 2 of 4", { id });
    toast.success("4 cells updated", { id });

    expect(current()).toHaveLength(1);
    expect(find(id)).toMatchObject({ type: "success", title: "4 cells updated", revision: 2 });
  });

  it("draws a custom toast from the caller's render function", () => {
    const render = vi.fn(() => null);
    const id = toast.custom(render, { toasterId: "user-settings-preview", duration: 4600 });

    expect(find(id)).toMatchObject({
      render,
      toasterId: "user-settings-preview",
      duration: 4600,
    });
  });

  it("dismisses one toast by id", () => {
    const keep = toast("Keep");
    const drop = toast("Drop");
    toast.dismiss(drop);

    expect(current().map((entry) => entry.id)).toEqual([keep]);
  });
});

describe("toast.promise", () => {
  it("shows the loading toast, then turns that same toast into the result, then runs finally", async () => {
    const onFinally = vi.fn();
    let resolve!: (value: { proNumber: string }) => void;
    const work = new Promise<{ proNumber: string }>((done) => {
      resolve = done;
    });

    const id = toast.promise(work, {
      loading: "Generating shipment…",
      success: (result) => `Shipment ${result.proNumber} generated`,
      error: "Failed to generate shipment",
      finally: onFinally,
    });

    expect(find(id)).toMatchObject({ type: "loading", title: "Generating shipment…" });
    expect(onFinally).not.toHaveBeenCalled();

    resolve({ proNumber: "S-104882" });
    await flush();

    expect(current()).toHaveLength(1);
    expect(find(id)).toMatchObject({ type: "success", title: "Shipment S-104882 generated" });
    expect(onFinally).toHaveBeenCalledTimes(1);
  });

  it("takes a whole toast as the outcome, with a description and an action", async () => {
    const open = vi.fn();
    const id = toast.promise(Promise.resolve(12), {
      loading: "Dry run: posting invoices…",
      success: (count) => ({
        title: `${count} invoices would post`,
        description: "$48,210.40 in total",
        action: { label: "View list", onClick: open },
      }),
    });
    await flush();

    expect(find(id)).toMatchObject({
      type: "success",
      title: "12 invoices would post",
      description: "$48,210.40 in total",
      action: { label: "View list", onClick: open },
    });
  });

  it("shows the error message when the work fails, and still runs finally", async () => {
    const onFinally = vi.fn();
    const id = toast.promise(Promise.reject(new Error("timeout")), {
      loading: "Testing the connection…",
      success: "Connected",
      error: (reason) => `QuickBooks refused: ${(reason as Error).message}`,
      finally: onFinally,
    });
    await flush();

    expect(find(id)).toMatchObject({ type: "error", title: "QuickBooks refused: timeout" });
    expect(onFinally).toHaveBeenCalledTimes(1);
  });

  it("treats a work function that throws as a failure", async () => {
    const id = toast.promise(
      () => {
        throw new Error("boom");
      },
      { loading: "Duplicating…", error: "Couldn't duplicate the template" },
    );
    await flush();

    expect(find(id)).toMatchObject({ type: "error", title: "Couldn't duplicate the template" });
  });

  it("removes the loading toast when there is no message for the outcome", async () => {
    const id = toast.promise(Promise.resolve(), { loading: "Saving…" });
    expect(find(id)).toBeDefined();

    await flush();

    expect(find(id)).toBeUndefined();
  });

  it("starts the result's clock, since a loading toast never runs out", async () => {
    const id = toast.promise(Promise.resolve(), { loading: "Saving…", success: "Saved" });
    expect(find(id)?.duration).toBe(Infinity);

    await flush();

    expect(find(id)?.duration).toBe(5000);
  });
});
