import {
  belongsToToaster,
  createToastStore,
  resolveDuration,
  type ToastRecord,
} from "@trenova/shared/components/ui/toast-store";
import { describe, expect, it, vi } from "vitest";

function only(store: ReturnType<typeof createToastStore>): ToastRecord {
  const { toasts } = store.getSnapshot();
  expect(toasts).toHaveLength(1);
  return toasts[0];
}

describe("toast store", () => {
  it("puts the newest toast first", () => {
    const store = createToastStore();
    store.show({ type: "success", title: "Aging report is ready" });
    store.show({ type: "error", title: "Couldn't reach PC*MILER" });

    expect(store.getSnapshot().toasts.map((toast) => toast.title)).toEqual([
      "Couldn't reach PC*MILER",
      "Aging report is ready",
    ]);
  });

  it("folds a repeat of plain news into the toast already up, counting it and moving it to the front", () => {
    const store = createToastStore();
    const first = store.show({
      type: "error",
      title: "Couldn't reach PC*MILER",
      description: "Mileage fills in when it answers.",
    });
    store.show({ type: "info", title: "Sync started" });
    const again = store.show({
      type: "error",
      title: "Couldn't reach PC*MILER",
      description: "Mileage fills in when it answers.",
    });

    const { toasts } = store.getSnapshot();
    expect(again).toBe(first);
    expect(toasts.map((toast) => [toast.title, toast.count])).toEqual([
      ["Couldn't reach PC*MILER", 2],
      ["Sync started", 1],
    ]);
    expect(toasts[0].revision).toBe(0);
  });

  it("keeps toasts that differ in type, description or Toaster apart", () => {
    const store = createToastStore();
    store.show({ type: "error", title: "Couldn't save" });
    store.show({ type: "warning", title: "Couldn't save" });
    store.show({ type: "error", title: "Couldn't save", description: "The period is closed." });
    store.show({ type: "error", title: "Couldn't save", toasterId: "user-settings" });

    expect(store.getSnapshot().toasts.map((toast) => toast.count)).toEqual([1, 1, 1, 1]);
  });

  it("never folds toasts that carry an action, since each Undo undoes something different", () => {
    const store = createToastStore();
    const undoFirst = vi.fn();
    const undoSecond = vi.fn();
    store.show({
      type: "default",
      title: "Shipment archived",
      action: { label: "Undo", onClick: undoFirst },
    });
    store.show({
      type: "default",
      title: "Shipment archived",
      action: { label: "Undo", onClick: undoSecond },
    });

    const { toasts } = store.getSnapshot();
    expect(toasts).toHaveLength(2);
    expect(toasts.map((toast) => toast.action?.onClick)).toEqual([undoSecond, undoFirst]);
  });

  it("does not fold toasts whose content is markup it cannot compare", () => {
    const store = createToastStore();
    const description = { type: "span", props: {}, key: null };
    store.show({ type: "success", title: "Paid", description: description as never });
    store.show({ type: "success", title: "Paid", description: description as never });

    expect(store.getSnapshot().toasts).toHaveLength(2);
  });

  it("changes a toast in place when shown again with its id, keeping fields the call leaves out", () => {
    const store = createToastStore();
    const id = store.show({ type: "loading", title: "Saving 0 of 4", description: "Rates" });
    store.show({ id, type: "success", title: "4 cells updated" });

    const toast = only(store);
    expect(toast).toMatchObject({
      id,
      type: "success",
      title: "4 cells updated",
      description: "Rates",
      revision: 1,
      count: 1,
    });
    expect(toast.duration).toBe(5000);
  });

  it("clears a field when the update passes it as undefined", () => {
    const store = createToastStore();
    const id = store.show({
      type: "loading",
      title: "Saving",
      action: { label: "Cancel", onClick: vi.fn() },
    });
    store.show({ id, type: "error", title: "No cells were saved", action: undefined });

    expect(only(store).action).toBeUndefined();
  });

  it("creates the toast with the caller's id when none is up yet", () => {
    const store = createToastStore();
    store.show({ id: "user-settings-saved", type: "success", title: "Saved" });

    expect(only(store).id).toBe("user-settings-saved");
  });

  it("calls onDismiss for a toast the person closes and onAutoClose for one that runs out", () => {
    const store = createToastStore();
    const onDismiss = vi.fn();
    const onAutoClose = vi.fn();
    const closed = store.show({ type: "info", title: "Closed", onDismiss, onAutoClose });
    const expired = store.show({ type: "info", title: "Expired", onDismiss, onAutoClose });

    store.dismiss(closed);
    store.expire(expired);

    expect(onDismiss).toHaveBeenCalledTimes(1);
    expect(onDismiss.mock.calls[0][0]).toMatchObject({ id: closed, title: "Closed" });
    expect(onAutoClose).toHaveBeenCalledTimes(1);
    expect(onAutoClose.mock.calls[0][0]).toMatchObject({ id: expired, title: "Expired" });
    expect(store.getSnapshot().toasts).toEqual([]);
  });

  it("dismisses every toast when no id is given", () => {
    const store = createToastStore();
    store.show({ type: "info", title: "One" });
    store.show({ type: "info", title: "Two", toasterId: "user-settings" });

    store.dismiss();

    expect(store.getSnapshot().toasts).toEqual([]);
  });

  it("does nothing, and tells no listener, when the toast is already gone", () => {
    const store = createToastStore();
    const listener = vi.fn();
    store.subscribe(listener);

    store.dismiss("missing");

    expect(listener).not.toHaveBeenCalled();
  });

  it("keeps at most fifty toasts, dropping the oldest", () => {
    const store = createToastStore();
    const onDismiss = vi.fn();
    for (let i = 0; i < 51; i += 1) {
      store.show({ type: "info", title: `Toast ${i}`, onDismiss });
    }

    const { toasts } = store.getSnapshot();
    expect(toasts).toHaveLength(50);
    expect(toasts.at(-1)?.title).toBe("Toast 1");
    expect(onDismiss).toHaveBeenCalledTimes(1);
    expect(onDismiss.mock.calls[0][0]).toMatchObject({ title: "Toast 0" });
  });

  it("tracks which named Toasters are mounted, counting each mount", () => {
    const store = createToastStore();
    const first = store.registerToaster("user-settings");
    const second = store.registerToaster("user-settings");

    first();
    expect(store.getSnapshot().toasters.has("user-settings")).toBe(true);
    second();
    expect(store.getSnapshot().toasters.has("user-settings")).toBe(false);
  });
});

describe("resolveDuration", () => {
  const plain = { title: "Saved", description: undefined, hasAction: false };

  it("uses the caller's duration whatever the content", () => {
    expect(resolveDuration("error", 1200, plain)).toBe(1200);
    expect(resolveDuration("success", Infinity, plain)).toBe(Infinity);
  });

  it("keeps a loading toast until it changes", () => {
    expect(resolveDuration("loading", undefined, plain)).toBe(Infinity);
  });

  it("gives each type its own floor for short news", () => {
    expect(resolveDuration("success", undefined, plain)).toBe(5000);
    expect(resolveDuration("info", undefined, plain)).toBe(6000);
    expect(resolveDuration("warning", undefined, plain)).toBe(7000);
    expect(resolveDuration("error", undefined, plain)).toBe(8000);
  });

  it("stays long enough to press what it offers", () => {
    expect(resolveDuration("default", undefined, { ...plain, hasAction: true })).toBe(8000);
  });

  it("stays long enough to read a long description, up to sixteen seconds", () => {
    const description = Array.from({ length: 30 }, () => "word").join(" ");
    expect(resolveDuration("success", undefined, { ...plain, description })).toBe(1500 + 31 * 260);

    const essay = Array.from({ length: 200 }, () => "word").join(" ");
    expect(resolveDuration("success", undefined, { ...plain, description: essay })).toBe(16000);
  });
});

describe("belongsToToaster", () => {
  const toast = (toasterId: string | undefined) => ({ toasterId }) as ToastRecord;

  it("shows a named Toaster only its own toasts", () => {
    const mounted = new Set(["user-settings"]);
    expect(belongsToToaster(toast("user-settings"), "user-settings", mounted)).toBe(true);
    expect(belongsToToaster(toast(undefined), "user-settings", mounted)).toBe(false);
  });

  it("shows the app's Toaster toasts addressed to a Toaster that is not on screen", () => {
    expect(belongsToToaster(toast(undefined), undefined, new Set())).toBe(true);
    expect(belongsToToaster(toast("user-settings"), undefined, new Set())).toBe(true);
    expect(belongsToToaster(toast("user-settings"), undefined, new Set(["user-settings"]))).toBe(
      false,
    );
  });
});
