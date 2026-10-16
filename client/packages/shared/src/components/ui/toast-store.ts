import type { MouseEvent, ReactNode } from "react";

export type ToastId = string | number;

export type ToastType = "default" | "success" | "error" | "warning" | "info" | "loading";

export type ToastAction = {
  label: ReactNode;
  onClick: (event: MouseEvent<HTMLButtonElement>) => void;
};

/** One line of evidence behind Details, such as a request reference to quote to support. */
export type ToastDetail = {
  label: string;
  value: string;
  monospace?: boolean;
};

export type ToastOptions = {
  /** Calling again with the same id changes that toast in place instead of adding one. */
  id?: ToastId;
  /** The `id` of the Toaster that shows it; unset, or not mounted, means the app's own. */
  toasterId?: string;
  description?: ReactNode;
  action?: ToastAction;
  cancel?: ToastAction;
  details?: readonly ToastDetail[];
  /** Milliseconds on screen; `Infinity` keeps it until dismissed. Unset reads the content. */
  duration?: number;
  /** Interrupts a screen reader the way an error does. */
  urgent?: boolean;
  onDismiss?: (toast: ToastRecord) => void;
  onAutoClose?: (toast: ToastRecord) => void;
};

export type ToastRecord = {
  id: ToastId;
  toasterId: string | undefined;
  type: ToastType;
  title: ReactNode;
  description: ReactNode;
  action: ToastAction | undefined;
  cancel: ToastAction | undefined;
  details: readonly ToastDetail[] | undefined;
  /** What the caller asked for; `duration` is what the toast actually stays for. */
  requestedDuration: number | undefined;
  duration: number;
  urgent: boolean;
  render: ((id: ToastId) => ReactNode) | undefined;
  /** How many identical toasts this one stands for; each repeat restarts its clock. */
  count: number;
  /** Advances when the toast is changed in place, so the card redraws it and restarts its clock. */
  revision: number;
  onDismiss: ((toast: ToastRecord) => void) | undefined;
  onAutoClose: ((toast: ToastRecord) => void) | undefined;
};

export type ToastInput = ToastOptions & {
  type: ToastType;
  title: ReactNode;
  render?: (id: ToastId) => ReactNode;
};

export type ToastSnapshot = {
  toasts: readonly ToastRecord[];
  /** The ids of the named Toasters on screen now. */
  toasters: ReadonlySet<string>;
};

const BASE_DURATION: Record<ToastType, number> = {
  default: 5000,
  success: 5000,
  info: 6000,
  warning: 7000,
  error: 8000,
  loading: Infinity,
};
const ACTION_DURATION = 8000;
const MAX_DURATION = 16000;
const READING_LEAD_MS = 1500;
const READING_MS_PER_WORD = 260;
const MAX_TOASTS = 50;

function wordCount(node: ReactNode): number {
  if (typeof node === "string") {
    return node.split(/\s+/).filter(Boolean).length;
  }
  if (typeof node === "number") {
    return 1;
  }
  return 0;
}

/**
 * How long a toast stays when the caller did not say: long enough to read what it says at
 * an unhurried pace, longer again when it offers something to press.
 */
export function resolveDuration(
  type: ToastType,
  requested: number | undefined,
  content: { title: ReactNode; description: ReactNode; hasAction: boolean },
): number {
  if (requested !== undefined) {
    return requested;
  }
  const base = BASE_DURATION[type];
  if (!Number.isFinite(base)) {
    return base;
  }
  const words = wordCount(content.title) + wordCount(content.description);
  const reading = READING_LEAD_MS + words * READING_MS_PER_WORD;
  const floor = content.hasAction ? Math.max(base, ACTION_DURATION) : base;
  return Math.min(MAX_DURATION, Math.max(floor, reading));
}

/**
 * Plain news that repeats folds into the toast already on screen. A toast that carries an
 * action never folds: two "Shipment archived · Undo" toasts undo two different shipments.
 */
function foldKey(input: ToastInput): string | null {
  if (
    input.id !== undefined ||
    input.render ||
    input.action ||
    input.cancel ||
    input.details ||
    input.type === "loading" ||
    typeof input.title !== "string" ||
    (input.description !== undefined && typeof input.description !== "string")
  ) {
    return null;
  }
  return [input.toasterId ?? "", input.type, input.title, input.description ?? ""].join("\u0000");
}

function recordFoldKey(record: ToastRecord): string | null {
  return foldKey({
    type: record.type,
    title: record.title,
    description: record.description ?? undefined,
    toasterId: record.toasterId,
    action: record.action,
    cancel: record.cancel,
    details: record.details,
    render: record.render,
  });
}

function build(input: ToastInput, id: ToastId, previous?: ToastRecord): ToastRecord {
  const merged = previous ? { ...previous, ...input } : input;
  const record: ToastRecord = {
    id,
    toasterId: merged.toasterId,
    type: input.type,
    title: input.title,
    description: merged.description,
    action: merged.action,
    cancel: merged.cancel,
    details: merged.details,
    requestedDuration: "duration" in input ? input.duration : undefined,
    duration: 0,
    urgent: merged.urgent ?? false,
    render: input.render,
    count: 1,
    revision: previous ? previous.revision + 1 : 0,
    onDismiss: merged.onDismiss,
    onAutoClose: merged.onAutoClose,
  };
  record.duration = resolveDuration(record.type, record.requestedDuration, {
    title: record.title,
    description: record.description,
    hasAction: Boolean(record.action || record.cancel || record.details),
  });
  return record;
}

export type ToastStore = {
  subscribe: (listener: () => void) => () => void;
  getSnapshot: () => ToastSnapshot;
  show: (input: ToastInput) => ToastId;
  dismiss: (id?: ToastId) => void;
  expire: (id: ToastId) => void;
  registerToaster: (id: string) => () => void;
  /** Mints an id for a toast that will be filled in later, such as a promise's. */
  nextId: () => ToastId;
};

export function createToastStore(): ToastStore {
  let snapshot: ToastSnapshot = { toasts: [], toasters: new Set() };
  const listeners = new Set<() => void>();
  const mountCounts = new Map<string, number>();
  let sequence = 0;

  const publish = (next: Partial<ToastSnapshot>) => {
    snapshot = { ...snapshot, ...next };
    for (const listener of listeners) {
      listener();
    }
  };

  const nextId = (): ToastId => {
    sequence += 1;
    return `toast-${sequence}`;
  };

  const remove = (predicate: (toast: ToastRecord) => boolean, reason: "dismiss" | "expire") => {
    const removed: ToastRecord[] = [];
    const kept: ToastRecord[] = [];
    for (const toast of snapshot.toasts) {
      (predicate(toast) ? removed : kept).push(toast);
    }
    if (removed.length === 0) {
      return;
    }
    publish({ toasts: kept });
    for (const toast of removed) {
      if (reason === "expire") {
        toast.onAutoClose?.(toast);
      } else {
        toast.onDismiss?.(toast);
      }
    }
  };

  const show = (input: ToastInput): ToastId => {
    const toasts = snapshot.toasts;

    if (input.id !== undefined) {
      const index = toasts.findIndex((toast) => toast.id === input.id);
      if (index !== -1) {
        const next = [...toasts];
        next[index] = build(input, input.id, toasts[index]);
        publish({ toasts: next });
        return input.id;
      }
    }

    const key = foldKey(input);
    if (key !== null) {
      const twin = toasts.find((toast) => recordFoldKey(toast) === key);
      if (twin) {
        const folded: ToastRecord = { ...twin, count: twin.count + 1 };
        publish({ toasts: [folded, ...toasts.filter((toast) => toast !== twin)] });
        return twin.id;
      }
    }

    const id = input.id ?? nextId();
    const record = build(input, id);
    const next = [record, ...toasts];
    const overflow = next.length > MAX_TOASTS ? next.slice(MAX_TOASTS) : [];
    publish({ toasts: overflow.length ? next.slice(0, MAX_TOASTS) : next });
    for (const dropped of overflow) {
      dropped.onDismiss?.(dropped);
    }
    return id;
  };

  return {
    subscribe(listener) {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    getSnapshot: () => snapshot,
    show,
    dismiss(id) {
      remove((toast) => id === undefined || toast.id === id, "dismiss");
    },
    expire(id) {
      remove((toast) => toast.id === id, "expire");
    },
    registerToaster(id) {
      mountCounts.set(id, (mountCounts.get(id) ?? 0) + 1);
      publish({ toasters: new Set(mountCounts.keys()) });
      return () => {
        const count = (mountCounts.get(id) ?? 1) - 1;
        if (count <= 0) {
          mountCounts.delete(id);
        } else {
          mountCounts.set(id, count);
        }
        publish({ toasters: new Set(mountCounts.keys()) });
      };
    },
    nextId,
  };
}

export const toastStore = createToastStore();

/**
 * Whether a Toaster shows a toast: its own, or, for the app's unnamed Toaster, any toast
 * addressed to a Toaster that is not on screen, so news sent from a page the person has
 * left still reaches them.
 */
export function belongsToToaster(
  toast: ToastRecord,
  toasterId: string | undefined,
  mounted: ReadonlySet<string>,
): boolean {
  if (toasterId !== undefined) {
    return toast.toasterId === toasterId;
  }
  return toast.toasterId === undefined || !mounted.has(toast.toasterId);
}
