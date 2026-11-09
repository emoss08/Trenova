import { toast, toastDuration } from "@trenova/shared/components/ui/toast";
import { toast as ding } from "@yusufceng1z/ding";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

/** The toast on screen, as ding's store holds it. */
function shown() {
  return ding
    .getState()
    .toasts.filter((item) => item.phase === "enter" || item.phase === "open")
    .at(-1);
}

function drawn() {
  return ding.getState().toasts;
}

/** Lets a settled promise's handlers run. */
async function flush() {
  await vi.advanceTimersByTimeAsync(0);
  for (let i = 0; i < 5; i += 1) {
    await Promise.resolve();
  }
}

/** Presses the toast's button the way ding's renderer does. */
function press() {
  const item = shown();
  expect(item).toBeDefined();
  ding.act(item?.id ?? "", "action");
}

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(async () => {
  ding.dismiss(undefined, { immediate: true });
  await vi.runAllTimersAsync();
  vi.useRealTimers();
});

describe("toast", () => {
  it("raises each type with its description and button, and a plain toast as info", () => {
    toast.success("Aging report is ready", {
      description: "AR aging · 1,284 invoices",
      action: { label: "Open", onClick: vi.fn() },
    });
    expect(shown()).toMatchObject({
      type: "success",
      title: "Aging report is ready",
      description: "AR aging · 1,284 invoices",
      action: { label: "Open" },
    });

    toast("Tracking link copied");
    expect(shown()).toMatchObject({ type: "info", title: "Tracking link copied" });
    toast.error("Couldn't send");
    expect(shown()?.type).toBe("error");
    toast.warning("Approved 46 of 48");
    expect(shown()?.type).toBe("warning");
    toast.loading("Saving 0 of 4");
    expect(shown()?.type).toBe("loading");
  });

  it("changes a toast in place when called again with its id", () => {
    const id = toast.loading("Saving 0 of 4");
    toast.loading("Saving 2 of 4", { id });
    toast.success("4 cells updated", { id });

    expect(drawn()).toHaveLength(1);
    expect(shown()).toMatchObject({ type: "success", title: "4 cells updated" });
  });

  it("moves a progress bar without announcing it again, clamped to 0–1", () => {
    const id = toast.loading("Uploading 3 files", { progress: 0 });
    expect(shown()?.progress).toBe(0);
    const announced = ding.getState().status;

    toast.loading("Uploading 3 files", { id, progress: 0.5 });
    expect(shown()?.progress).toBe(0.5);
    expect(ding.getState().status).toBe(announced);

    toast.loading("Uploading 3 files", { id, progress: 4 });
    expect(shown()?.progress).toBe(1);

    toast.success("3 files uploaded", { id });
    expect(shown()).toMatchObject({ type: "success", title: "3 files uploaded" });
    expect(shown()?.progress).toBeUndefined();
  });

  it("lets a newer toast keep its place over a late plain result, but not over a failure", () => {
    const id = toast.loading("Exporting shipments");
    toast.info("Something newer");

    toast.success("Export ready", { id });
    expect(shown()).toMatchObject({ type: "info", title: "Something newer" });

    toast.error("The export failed", { id });
    expect(shown()).toMatchObject({ type: "error", title: "The export failed" });
  });

  it("counts plain news that repeats while it is up instead of replacing it", () => {
    toast.success("Copied");
    toast.success("Copied");

    expect(drawn()).toHaveLength(1);
    expect(shown()).toMatchObject({ title: "Copied", count: 2 });
  });

  it("dismisses only the caller's own toast", () => {
    const first = toast.info("First");
    const second = toast.info("Second");

    toast.dismiss(first);
    expect(shown()?.title).toBe("Second");

    toast.dismiss(second);
    expect(shown()).toBeUndefined();
  });

  it("keeps the toast an action raised in its place, and closes it otherwise", () => {
    const id = toast.success("Shipment archived", {
      action: { label: "Undo", onClick: () => toast.success("Shipment restored", { id }) },
    });
    press();
    expect(shown()).toMatchObject({ type: "success", title: "Shipment restored" });
    expect(shown()?.action).toBeUndefined();

    const open = vi.fn();
    toast.success("Report ready", { action: { label: "Open", onClick: open } });
    press();
    expect(open).toHaveBeenCalledTimes(1);
    expect(shown()).toBeUndefined();
  });
});

describe("toastDuration", () => {
  const plain = { title: "Saved", description: "", hasAction: false };

  it("keeps errors and loading toasts until they change or are closed", () => {
    expect(toastDuration("error", undefined, plain)).toBe(0);
    expect(toastDuration("loading", undefined, plain)).toBe(0);
  });

  it("reads the words, with eight seconds at least for a button and sixteen at most", () => {
    expect(toastDuration("success", undefined, plain)).toBe(5000);
    expect(toastDuration("success", undefined, { ...plain, hasAction: true })).toBe(8000);
    expect(
      toastDuration("info", undefined, {
        title: "Twelve invoices were emailed",
        description: "Three customers have no billing email on file and were skipped",
        hasAction: false,
      }),
    ).toBe(1500 + 15 * 260);
    expect(toastDuration("info", undefined, { ...plain, description: "word ".repeat(100) })).toBe(
      16000,
    );
  });

  it("takes the caller's duration, with Infinity keeping the toast", () => {
    expect(toastDuration("success", 4200, plain)).toBe(4200);
    expect(toastDuration("success", Infinity, plain)).toBe(0);
  });
});

describe("toast.promise", () => {
  it("shows the loading toast, then turns that same toast into the result, then runs finally", async () => {
    const onFinally = vi.fn();
    let resolve!: (value: { proNumber: string }) => void;
    const work = new Promise<{ proNumber: string }>((done) => {
      resolve = done;
    });

    toast.promise(work, {
      loading: "Generating shipment…",
      success: (result) => `Shipment ${result.proNumber} generated`,
      error: "Failed to generate shipment",
      finally: onFinally,
    });
    expect(shown()).toMatchObject({ type: "loading", title: "Generating shipment…" });
    expect(onFinally).not.toHaveBeenCalled();

    resolve({ proNumber: "S-104882" });
    await flush();

    expect(drawn()).toHaveLength(1);
    expect(shown()).toMatchObject({ type: "success", title: "Shipment S-104882 generated" });
    expect(onFinally).toHaveBeenCalledTimes(1);
  });

  it("takes a whole toast, with a description and a button, on either side", async () => {
    toast.promise(Promise.resolve(12), {
      loading: { title: "Dry run: posting invoices…", description: "Nothing is posted yet." },
      success: (count) => ({
        title: `${count} invoices would post`,
        description: "$48,210.40 in total",
        action: { label: "View list", onClick: vi.fn() },
      }),
    });
    expect(shown()?.description).toBe("Nothing is posted yet.");
    await flush();

    expect(shown()).toMatchObject({
      type: "success",
      title: "12 invoices would post",
      description: "$48,210.40 in total",
      action: { label: "View list" },
    });
  });

  it("shows the failure when the work fails, and still runs finally", async () => {
    const onFinally = vi.fn();
    toast.promise(Promise.reject(new Error("timeout")), {
      loading: "Testing the connection…",
      success: "Connected",
      error: (reason) => `QuickBooks refused: ${(reason as Error).message}`,
      finally: onFinally,
    });
    await flush();

    expect(shown()).toMatchObject({ type: "error", title: "QuickBooks refused: timeout" });
    expect(onFinally).toHaveBeenCalledTimes(1);
  });

  it("treats a work function that throws as a failure", async () => {
    toast.promise(
      () => {
        throw new Error("boom");
      },
      { loading: "Duplicating…", error: "Couldn't duplicate the template" },
    );
    await flush();

    expect(shown()).toMatchObject({ type: "error", title: "Couldn't duplicate the template" });
  });

  it("closes the loading toast when the outcome has no message or returns null", async () => {
    toast.promise(Promise.resolve(), { loading: "Saving…" });
    await flush();
    expect(shown()).toBeUndefined();

    toast.promise(Promise.reject(new Error("cancelled")), {
      loading: "Removing…",
      error: () => null,
    });
    await flush();
    expect(shown()).toBeUndefined();
  });

  it("starts the result's clock, since a loading toast never runs out", async () => {
    toast.promise(Promise.resolve(), { loading: "Saving…", success: "Saved" });
    expect(shown()?.duration).toBe(0);

    await flush();

    expect(shown()?.duration).toBe(5000);
  });
});
