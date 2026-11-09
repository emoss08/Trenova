import {
  toast as ding,
  type ToastAction as DingAction,
  type ToastItem,
  type ToastPatch,
  type ToastType as DingType,
} from "@yusufceng1z/ding";

export type ToastId = string | number;

export type ToastAction = {
  label: string;
  /**
   * Runs on press, then the toast closes, unless the handler raised the next toast in its
   * place ("Undo" becoming "Restored"), which stays.
   */
  onClick: () => void;
};

export type ToastOptions = {
  /** Calling again with the same id changes that toast in place instead of raising another. */
  id?: ToastId;
  /** One line of plain text under the title. */
  description?: string;
  action?: ToastAction;
  /** Milliseconds on screen; `Infinity` keeps it until dismissed. Unset reads the content. */
  duration?: number;
  /** 0–1 draws a bar inside the toast; moving it under the same id is silent. */
  progress?: number;
};

/** A whole toast as a value: what a promise's loading state or outcome shows. */
export type ToastOutcome = { title: string } & Omit<ToastOptions, "id">;

type ToastMessage = string | ToastOutcome;

/** What a settled promise shows; `null` closes the toast quietly. */
type OutcomeMessage<T> = ToastMessage | ((value: T) => ToastMessage | null);

export type ToastPromiseOptions<T> = {
  id?: ToastId;
  loading?: ToastMessage;
  success?: OutcomeMessage<T>;
  error?: OutcomeMessage<unknown>;
  /** Runs once the promise settles, either way, after the toast has changed. */
  finally?: () => void | Promise<void>;
};

type ToastKind = "default" | DingType;

/** What the toast on screen says, and which call owns it. */
type Shown = {
  key: ToastId;
  ding: string;
  type: DingType;
  title: string;
  description: string;
  action: DingAction | undefined;
  duration: number;
};

const BASE_DURATION = 5000;
const ACTION_DURATION = 8000;
const MAX_DURATION = 16000;
const READING_LEAD_MS = 1500;
const READING_MS_PER_WORD = 260;
const MAX_REMEMBERED = 200;

let sequence = 0;
let current: Shown | undefined;
const owners = new Map<ToastId, string>();

function nextDingId(): string {
  sequence += 1;
  return `trenova-toast-${sequence}`;
}

function remember(key: ToastId, dingId: string) {
  owners.delete(key);
  owners.set(key, dingId);
  if (owners.size > MAX_REMEMBERED) {
    const oldest = owners.keys().next();
    if (!oldest.done) owners.delete(oldest.value);
  }
}

function liveToast(): ToastItem | undefined {
  return ding
    .getState()
    .toasts.filter((item) => item.phase === "enter" || item.phase === "open")
    .at(-1);
}

function words(text: string): number {
  return text.split(/\s+/).filter(Boolean).length;
}

function clampProgress(fraction: number): number {
  return Number.isFinite(fraction) ? Math.min(1, Math.max(0, fraction)) : 0;
}

/**
 * How long a toast stays, in ding's terms (0 keeps it). Errors and loading toasts stay until
 * they change or the person closes them; anything else stays long enough to read at an
 * unhurried pace, and at least eight seconds when it offers something to press.
 */
export function toastDuration(
  type: DingType,
  requested: number | undefined,
  content: { title: string; description: string; hasAction: boolean },
): number {
  if (requested !== undefined) {
    return Number.isFinite(requested) ? Math.max(1, Math.round(requested)) : 0;
  }
  if (type === "error" || type === "loading") {
    return 0;
  }
  const reading =
    READING_LEAD_MS + (words(content.title) + words(content.description)) * READING_MS_PER_WORD;
  const floor = content.hasAction ? ACTION_DURATION : BASE_DURATION;
  return Math.min(MAX_DURATION, Math.max(floor, reading));
}

/** What keeps a toast as the handler left it, so ding does not close it after the press. */
function keep(shown: Shown): ToastPatch {
  return {
    action: shown.action ?? null,
    duration: shown.duration,
  };
}

function wrapAction(action: ToastAction): DingAction {
  return {
    label: action.label,
    onClick: () => {
      const before = current;
      action.onClick();
      return current && current !== before && liveToast() ? keep(current) : undefined;
    },
  };
}

function sameText(item: ToastItem, type: DingType, title: string, description: string): boolean {
  return item.type === type && item.title === title && (item.description ?? "") === description;
}

function show(kind: ToastKind, title: string, options: ToastOptions = {}): ToastId {
  const type: DingType = kind === "default" ? "info" : kind;
  const description = options.description ?? "";
  const callerId = options.id;
  const owned = callerId === undefined ? undefined : owners.get(callerId);
  const live = liveToast();
  const ownsScreen = owned !== undefined && live !== undefined && current?.ding === owned;

  // A bar moving under the same words is not news: move it without announcing again.
  if (
    ownsScreen &&
    live &&
    current &&
    options.progress !== undefined &&
    sameText(live, type, title, description) &&
    (current.action?.label ?? null) === (options.action?.label ?? null)
  ) {
    ding.update(current.ding, { progress: clampProgress(options.progress) });
    return current.key;
  }

  // A newer toast took this one's place. A late result that needs the person still
  // reaches them; one that only reports progress or plain success gives way.
  if (callerId !== undefined && owned !== undefined && !ownsScreen && live !== undefined) {
    const needsPerson = type === "error" || type === "warning" || options.action !== undefined;
    if (!needsPerson) {
      return callerId;
    }
  }

  // Plain news repeated while it is still up counts up (×2) instead of replacing itself.
  const plain =
    callerId === undefined && options.action === undefined && options.progress === undefined;
  if (
    plain &&
    live &&
    current &&
    ding.getState().toasts.at(-1) === live &&
    !live.action &&
    !live.meta &&
    !live.image &&
    sameText(live, type, title, description)
  ) {
    ding.show(type, title, { description: live.description });
    return current.key;
  }

  const dingId = nextDingId();
  const action = options.action ? wrapAction(options.action) : undefined;
  const duration = toastDuration(type, options.duration, {
    title,
    description,
    hasAction: action !== undefined,
  });
  const key = callerId ?? dingId;
  current = { key, ding: dingId, type, title, description, action, duration };
  remember(key, dingId);
  ding.show(type, title, {
    id: dingId,
    description: description || undefined,
    action,
    duration,
    progress: options.progress === undefined ? undefined : clampProgress(options.progress),
  });
  return key;
}

function dismiss(id?: ToastId) {
  if (id === undefined) {
    ding.dismiss();
    return;
  }
  const dingId = owners.get(id);
  if (dingId !== undefined && current?.ding === dingId) {
    ding.dismiss(dingId);
  }
}

function normalize(message: ToastMessage): ToastOutcome {
  return typeof message === "string" ? { title: message } : message;
}

function settle<T>(
  id: ToastId,
  type: "success" | "error",
  message: OutcomeMessage<T> | undefined,
  value: T,
) {
  const resolved = typeof message === "function" ? message(value) : message;
  if (resolved === undefined || resolved === null) {
    dismiss(id);
    return;
  }
  const { title, ...rest } = normalize(resolved);
  show(type, title, { ...rest, id });
}

function promise<T>(
  work: Promise<T> | (() => Promise<T>),
  { id: requestedId, loading, success, error, finally: onFinally }: ToastPromiseOptions<T>,
): ToastId {
  const id = requestedId ?? nextDingId();
  if (loading !== undefined) {
    const { title, ...rest } = normalize(loading);
    show("loading", title, { ...rest, id });
  }
  let pending: Promise<T>;
  try {
    pending = Promise.resolve(typeof work === "function" ? work() : work);
  } catch (reason) {
    pending = Promise.reject(reason);
  }
  void pending
    .then(
      (value) => settle(id, "success", success, value),
      (reason: unknown) => settle(id, "error", error, reason),
    )
    .finally(() => onFinally?.());
  return id;
}

/**
 * Raises a toast: `toast(title)`, `toast.success(title, { description, action })`,
 * `toast.promise(work, { loading, success, error, finally })`, and the same `id` to change a
 * toast in place. ding draws one toast at a time, so each new one takes the place of the
 * last. See "Toasts" in docs/engineering/design-system.md for when a toast is the right
 * answer at all.
 */
export const toast = Object.assign(
  (title: string, options?: ToastOptions) => show("default", title, options),
  {
    success: (title: string, options?: ToastOptions) => show("success", title, options),
    error: (title: string, options?: ToastOptions) => show("error", title, options),
    warning: (title: string, options?: ToastOptions) => show("warning", title, options),
    info: (title: string, options?: ToastOptions) => show("info", title, options),
    loading: (title: string, options?: ToastOptions) => show("loading", title, options),
    promise,
    /** Closes this caller's toast, or whatever is up when called without an id. */
    dismiss,
  },
);
