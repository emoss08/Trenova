import {
  PanelDirtyTracker,
  usePanelCloseGuard,
  type PanelCloseGuard,
} from "@/components/form-changes";
import { AlertTriangleIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  use,
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
  type ReactNode,
  type RefObject,
} from "react";
import { createPortal } from "react-dom";
import type { FieldValues, UseFormReturn } from "react-hook-form";
import {
  WorkPaneIdentityContext,
  WorkPanePartContext,
  useWorkPaneApi,
  useWorkPanePart,
  useWorkPaneState,
} from "./work-pane-context";

/** How long the host takes to fold away, so a closing pane keeps its content until it has. */
const CLOSE_MS = 320;

/** Popups and dialogs that answer Escape themselves; a key pressed in one is not the pane's. */
const ESCAPE_OWNERS =
  '[role="dialog"], [role="alertdialog"], [role="menu"], [role="listbox"], [cmdk-root], [data-slot="popover-content"]';

const FIRST_FIELD = [
  "input:not([type=hidden]):not([type=file]):not([disabled])",
  "textarea:not([disabled])",
  "select:not([disabled])",
  '[role="combobox"]:not([aria-disabled="true"])',
  "button:not([disabled])",
  "a[href]",
  '[tabindex]:not([tabindex="-1"])',
]
  .map((selector) => `[data-slot="work-pane-body"] ${selector}`)
  .join(", ");

function escapeIsForPane(target: EventTarget | null, pane: HTMLElement | null): boolean {
  if (!(target instanceof Element)) return true;
  if (target.getAttribute("aria-expanded") === "true") return false;
  const owner = target.closest(ESCAPE_OWNERS);
  return owner === null || owner === pane;
}

function firstFocusable(content: HTMLElement): HTMLElement | null {
  return (
    content.querySelector<HTMLElement>("[autofocus], [data-autofocus]") ??
    content.querySelector<HTMLElement>(FIRST_FIELD) ??
    content.querySelector<HTMLElement>('[data-slot="work-pane-close"]')
  );
}

export type WorkPaneProps = {
  /** Whether this pane is open; a page drives it from its address (`form=`, a record param). */
  open: boolean;
  /** Told `false` once the pane closes for good: after any confirmation, or when another opens. */
  onOpenChange: (open: boolean) => void;
  title: ReactNode;
  /** One line under the header, at the top of the body. */
  description?: ReactNode;
  /** Changing it replays the content's entrance, for a read pane stepping to another record. */
  contentKey?: string;
  /** Where focus goes on close; the control that opened the pane by default. */
  finalFocus?: RefObject<HTMLElement | null>;
  /** `WorkPaneBody` and `WorkPaneFooter`, directly or inside a `<Form>`. */
  children: ReactNode;
};

/**
 * A form or a record's details beside the list it belongs to, drawn in the page's
 * `WorkPaneHost`. Its parts mirror `Dialog`'s, so a dialog's form moves here by swapping the
 * host: `title`/`description` for `DialogHeader`, `WorkPaneBody` for the fields,
 * `WorkPaneFooter` for `DialogFooter`. Escape and the close button ask before discarding a form
 * that holds changes (`WorkPaneDirtyTracker`); focus moves in on open and back on close. Keep it
 * mounted and drive `open`, so its close can play.
 */
export function WorkPane({
  open,
  onOpenChange,
  title,
  description,
  contentKey,
  finalFocus,
  children,
}: WorkPaneProps) {
  const t = useT();
  const api = useWorkPaneApi();
  const { activeId, host } = useWorkPaneState();
  const ownId = useId();
  const sharedId = use(WorkPaneIdentityContext);
  const id = sharedId ?? ownId;
  const titleId = useId();
  const descriptionId = useId();
  const contentRef = useRef<HTMLDivElement>(null);
  const pendingRef = useRef<(() => void) | null>(null);
  const shown = open && activeId === id;
  // A pane that takes over from its loading stand-in is already in the host: it draws without
  // replaying the entrance. A later opening of the same pane enters as usual.
  const [quietEntrance, setQuietEntrance] = useState(sharedId !== null && shown);

  const closeForGood = useLatestCallback(() => {
    onOpenChange(false);
    const then = pendingRef.current;
    pendingRef.current = null;
    then?.();
  });
  const panelGuard = usePanelCloseGuard({ open: shown, onClose: closeForGood });
  const requestClose = useLatestCallback((then?: () => void) => {
    pendingRef.current = then ?? null;
    panelGuard.requestClose();
  });
  const keepEditing = useLatestCallback(() => {
    pendingRef.current = null;
    panelGuard.keepEditing();
  });
  const displace = useLatestCallback(() => onOpenChange(false));
  const reportDirty = useLatestCallback((count: number) => {
    panelGuard.reportDirty(count);
    api.setDirty(id, count > 0);
  });
  const guard: PanelCloseGuard = { ...panelGuard, reportDirty, keepEditing };

  useEffect(() => {
    if (!open) return undefined;
    return api.register(id, { requestClose, displace });
  }, [open, api, id, requestClose, displace]);

  const [lingering, setLingering] = useState(false);
  const [wasShown, setWasShown] = useState(shown);
  if (wasShown !== shown) {
    setWasShown(shown);
    setLingering(!shown);
    if (shown) setQuietEntrance(false);
  }
  useEffect(() => {
    if (!lingering) return undefined;
    const timer = window.setTimeout(() => setLingering(false), CLOSE_MS);
    return () => window.clearTimeout(timer);
  }, [lingering]);

  useEffect(() => {
    if (!shown) return undefined;
    const active = document.activeElement;
    const opener = active instanceof HTMLElement && active !== document.body ? active : null;
    const returnTo = finalFocus?.current ?? opener;
    const content = contentRef.current;
    if (content && !content.contains(active)) {
      firstFocusable(content)?.focus({ preventScroll: true });
    }
    return () => {
      const now = document.activeElement;
      const focusWasInPane =
        now === null || now === document.body || (content?.contains(now) ?? false);
      if (returnTo?.isConnected && focusWasInPane) returnTo.focus({ preventScroll: true });
    };
  }, [shown, finalFocus]);

  const onEscape = useLatestCallback((event: KeyboardEvent) => {
    if (event.key !== "Escape" || event.defaultPrevented) return;
    if (!escapeIsForPane(event.target, contentRef.current)) return;
    event.preventDefault();
    if (panelGuard.confirmingClose) {
      keepEditing();
    } else {
      requestClose();
    }
  });
  useEffect(() => {
    if (!shown) return undefined;
    document.addEventListener("keydown", onEscape);
    return () => document.removeEventListener("keydown", onEscape);
  }, [shown, onEscape]);

  const [footers, setFooters] = useState(0);
  const registerFooter = useLatestCallback(() => {
    setFooters((count) => count + 1);
    return () => setFooters((count) => count - 1);
  });

  const drawn = host !== null && (shown || (lingering && activeId === null));
  if (!drawn) return null;

  return createPortal(
    <WorkPaneIdentityContext value={null}>
      <WorkPanePartContext.Provider
        value={{
          guard,
          requestClose: () => requestClose(),
          description,
          descriptionId,
          registerFooter,
        }}
      >
        <div
          ref={contentRef}
          key={contentKey}
          role="dialog"
          aria-modal={false}
          aria-labelledby={titleId}
          aria-describedby={description ? descriptionId : undefined}
          className={cn(
            "flex min-h-0 flex-1 flex-col",
            !quietEntrance && "animate-panel-in motion-reduce:animate-none",
            "[&>form]:flex [&>form]:min-h-0 [&>form]:flex-1 [&>form]:flex-col",
            "[&>form>[role=alert]]:mx-4 [&>form>[role=alert]]:mt-4 [&>form>[role=alert]]:mb-0",
          )}
        >
          <header className="flex h-11 shrink-0 items-center gap-2 border-b pr-2 pl-4">
            <h2 id={titleId} className="min-w-0 flex-1 truncate text-sm font-semibold">
              {title}
            </h2>
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              data-slot="work-pane-close"
              aria-label={t("Close")}
              onClick={() => requestClose()}
            >
              <XCloseIcon className="size-3.5" />
            </Button>
          </header>
          {children}
          {panelGuard.confirmingClose && footers === 0 ? <WorkPaneFooter /> : null}
        </div>
      </WorkPanePartContext.Provider>
    </WorkPaneIdentityContext>,
    host,
  );
}

/** The pane's scrolling body: the description line, then the fields or the record's details. */
export function WorkPaneBody({
  children,
  className,
}: {
  children?: ReactNode;
  className?: string;
}) {
  const { description, descriptionId } = useWorkPanePart();
  return (
    <ScrollArea className="min-h-0 flex-1" maskVariant="card">
      <div data-slot="work-pane-body" className={cn("flex flex-col gap-4 p-4", className)}>
        {description ? (
          <p id={descriptionId} className="text-sm text-muted-foreground">
            {description}
          </p>
        ) : null}
        {children}
      </div>
    </ScrollArea>
  );
}

/**
 * The pane's footer: `WorkPaneCancel` then the primary at the right, secondary and destructive
 * actions in `leading` at the left. While closing would lose changes it asks instead.
 */
export function WorkPaneFooter({
  leading,
  children,
  className,
}: {
  leading?: ReactNode;
  children?: ReactNode;
  className?: string;
}) {
  const t = useT();
  const { guard, registerFooter } = useWorkPanePart();
  useLayoutEffect(() => registerFooter(), [registerFooter]);

  return (
    <div
      data-slot="work-pane-footer"
      className={cn(
        "flex shrink-0 items-center justify-end gap-2 border-t bg-card px-4 py-3",
        className,
      )}
    >
      {guard.confirmingClose ? (
        <div
          role="alert"
          className="flex w-full animate-in items-center gap-2 duration-200 ease-settle fade-in-0 slide-in-from-bottom-1 motion-reduce:animate-none"
        >
          <AlertTriangleIcon className="size-4 shrink-0 text-warning" />
          <span className="text-sm font-medium">
            {guard.changedCount === 1
              ? t("Discard 1 unsaved change?")
              : t("Discard {0} unsaved changes?", guard.changedCount)}
          </span>
          <span className="flex-1" />
          <Button type="button" size="sm" variant="outline" autoFocus onClick={guard.keepEditing}>
            {t("Keep editing")}
          </Button>
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="border-danger-border text-danger-foreground hover:bg-danger-subtle"
            onClick={guard.discard}
          >
            {t("Discard and close")}
          </Button>
        </div>
      ) : (
        <>
          {leading ? (
            <div className="mr-auto flex min-w-0 items-center gap-2">{leading}</div>
          ) : null}
          {children}
        </>
      )}
    </div>
  );
}

/** The footer's Cancel: closes the pane, asking first when its form holds changes. */
export function WorkPaneCancel({ children }: { children?: ReactNode }) {
  const t = useT();
  const { requestClose } = useWorkPanePart();
  return (
    <Button type="button" size="sm" variant="outline" onClick={requestClose}>
      {children ?? t("Cancel")}
    </Button>
  );
}

/**
 * Renders nothing; tells the pane how many of `form`'s fields hold unsaved changes, so closing
 * asks first and the page's `useWorkPane().isDirty` follows. Put it anywhere inside the pane.
 */
export function WorkPaneDirtyTracker<T extends FieldValues>({
  form,
}: {
  form: UseFormReturn<T, any, any>;
}) {
  const { guard } = useWorkPanePart();
  return <PanelDirtyTracker form={form} guard={guard} />;
}
