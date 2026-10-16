import { isValidElement, type ReactNode } from "react";
import {
  toastStore,
  type ToastId,
  type ToastOptions,
  type ToastType,
} from "@trenova/shared/components/ui/toast-store";

export type {
  ToastAction,
  ToastDetail,
  ToastId,
  ToastOptions,
} from "@trenova/shared/components/ui/toast-store";

/** What a settled promise becomes: a title, or a title with the rest of a toast. */
export type ToastOutcome = { title: ReactNode } & Omit<ToastOptions, "id" | "toasterId">;

type OutcomeMessage<T> = ReactNode | ToastOutcome | ((value: T) => ReactNode | ToastOutcome);

export type ToastPromiseOptions<T> = ToastOptions & {
  loading?: ReactNode;
  success?: OutcomeMessage<T>;
  error?: OutcomeMessage<unknown>;
  /** Runs once the promise settles, either way, after the toast has changed. */
  finally?: () => void | Promise<void>;
};

function isOutcome(value: unknown): value is ToastOutcome {
  return (
    typeof value === "object" &&
    value !== null &&
    !Array.isArray(value) &&
    !isValidElement(value) &&
    "title" in value
  );
}

function show(type: ToastType, title: ReactNode, options?: ToastOptions): ToastId {
  return toastStore.show({ ...options, type, title });
}

function settle<T>(
  id: ToastId,
  type: "success" | "error",
  message: OutcomeMessage<T> | undefined,
  value: T,
  options: ToastOptions,
) {
  if (message === undefined) {
    toastStore.dismiss(id);
    return;
  }
  const resolved = typeof message === "function" ? message(value) : message;
  const cleared: ToastOptions = {
    action: undefined,
    cancel: undefined,
    details: undefined,
    ...options,
    id,
  };
  if (isOutcome(resolved)) {
    const { title, ...rest } = resolved;
    show(type, title, { ...cleared, ...rest, id });
    return;
  }
  show(type, resolved, cleared);
}

function promise<T>(
  work: Promise<T> | (() => Promise<T>),
  { loading, success, error, finally: onFinally, ...options }: ToastPromiseOptions<T>,
): ToastId {
  const id = options.id ?? toastStore.nextId();
  if (loading !== undefined) {
    show("loading", loading, { ...options, id });
  }
  let pending: Promise<T>;
  try {
    pending = Promise.resolve(typeof work === "function" ? work() : work);
  } catch (reason) {
    pending = Promise.reject(reason);
  }
  void pending
    .then(
      (value) => settle(id, "success", success, value, options),
      (reason: unknown) => settle(id, "error", error, reason, options),
    )
    .finally(() => onFinally?.());
  return id;
}

/**
 * Raises a toast. The call signatures match sonner's, which this replaced: `toast(title)`,
 * `toast.success(title, { description, action })`, `toast.promise(work, { loading, success,
 * error, finally })`, and the same `id` to change a toast in place. See "Toasts" in
 * docs/engineering/design-system.md for when a toast is the right answer at all.
 */
export const toast = Object.assign(
  (title: ReactNode, options?: ToastOptions) => show("default", title, options),
  {
    success: (title: ReactNode, options?: ToastOptions) => show("success", title, options),
    error: (title: ReactNode, options?: ToastOptions) => show("error", title, options),
    warning: (title: ReactNode, options?: ToastOptions) => show("warning", title, options),
    info: (title: ReactNode, options?: ToastOptions) => show("info", title, options),
    loading: (title: ReactNode, options?: ToastOptions) => show("loading", title, options),
    promise,
    /** Draws its own content in the toast's place; the stack, timer and swipe still apply. */
    custom: (render: (id: ToastId) => ReactNode, options?: ToastOptions) =>
      toastStore.show({ ...options, type: "default", title: null, render }),
    /** Dismisses one toast, or every toast when called without an id. */
    dismiss: (id?: ToastId) => toastStore.dismiss(id),
  },
);
