import { InputFieldSkeleton } from "@/components/fields/field-skeletons";
import { lazyComponent } from "@trenova/shared/lib/lazy-component";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { reportFailure } from "@trenova/shared/lib/failure-reporter";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { Suspense, useId, useState, type FunctionComponent, type ReactNode } from "react";
import { ErrorBoundary } from "react-error-boundary";
import { WorkPaneIdentityContext } from "./work-pane-context";
import { WorkPane, WorkPaneBody, WorkPaneCancel, WorkPaneFooter } from "./work-pane";

type PaneProps = {
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  onClose?: () => void;
};

export type LazyWorkPaneOptions<P> = {
  /** The pane's title while its code loads; a shimmer when left out. */
  title?: (props: P, t: TranslateFn) => ReactNode;
  /** Whether the pane is open; `props.open` when left out. */
  isOpen?: (props: P) => boolean;
  /** Closes the pane from its stand-in; `onOpenChange(false)`, else `onClose()`, when left out. */
  close?: (props: P) => void;
  /**
   * The record the pane is open on, for a pane that can move to another record while it stays
   * open (a read pane stepping through a list): a new key mounts the pane fresh, as an opening does.
   */
  openKey?: (props: P) => string | null | undefined;
  /** How many fields the stand-in shimmers; 5 when left out. */
  fields?: number;
};

/** A work pane whose code is read the first time it opens, or earlier on intent. */
export type LazyWorkPane<P extends object> = {
  /** Draws nothing until first opened; then the pane, with a stand-in while its code loads. */
  Component: FunctionComponent<P>;
  /** Reads the code now, so opening draws the pane at once. Call it on intent. */
  preload: () => Promise<unknown>;
};

function defaultIsOpen(props: object): boolean {
  return (props as PaneProps).open === true;
}

function defaultClose(props: object): void {
  const pane = props as PaneProps;
  if (pane.onOpenChange) {
    pane.onOpenChange(false);
  } else {
    pane.onClose?.();
  }
}

/**
 * A form in shimmer: the body of a form pane whose code or data is still on its way. The
 * stand-in draws it while a pane's module loads; a pane whose defaults wait on a read draws it
 * in its own body, so both look the same.
 */
export function PaneFormSkeleton({ fields = 5 }: { fields?: number }) {
  const t = useT();
  return (
    <div aria-busy aria-label={t("Loading the form")} className="flex flex-col gap-4">
      {Array.from({ length: fields }, (_, index) => (
        <InputFieldSkeleton key={index} />
      ))}
    </div>
  );
}

/** Where a pane's primary action will be, in shimmer. */
function PrimaryActionShimmer() {
  return <span aria-hidden className="block h-7 w-24 ui-shimmer rounded-md" />;
}

/**
 * The body and footer of a form pane whose defaults wait on a read (a preset type, a template):
 * the form in shimmer, Cancel, and a shimmer where the primary goes. Drawn inside the pane's own
 * `WorkPane`, so it matches the stand-in shown while the pane's code loads.
 */
export function PaneFormLoading({ fields = 5 }: { fields?: number }) {
  return (
    <>
      <WorkPaneBody>
        <PaneFormSkeleton fields={fields} />
      </WorkPaneBody>
      <WorkPaneFooter>
        <WorkPaneCancel />
        <PrimaryActionShimmer />
      </WorkPaneFooter>
    </>
  );
}

/**
 * The pane while its code loads or after it failed to: the real pane's header with its title,
 * the body, and a footer whose Cancel closes it. It is a `WorkPane` under the same id as the
 * pane that replaces it, so the host stays open and nothing moves when the form arrives.
 */
function PaneStandIn({
  open,
  onClose,
  title,
  fields,
  children,
}: {
  open: boolean;
  onClose: () => void;
  title: ReactNode | undefined;
  fields: number;
  children?: ReactNode;
}) {
  const t = useT();
  return (
    <WorkPane
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title={
        title ?? (
          <>
            <span className="sr-only">{t("Loading")}</span>
            <span aria-hidden className="block h-3.5 w-36 ui-shimmer rounded-sm" />
          </>
        )
      }
    >
      <WorkPaneBody>{children ?? <PaneFormSkeleton fields={fields} />}</WorkPaneBody>
      <WorkPaneFooter>
        <WorkPaneCancel />
        {children ? null : <PrimaryActionShimmer />}
      </WorkPaneFooter>
    </WorkPane>
  );
}

/**
 * A work pane read on demand. The tab keeps the returned `Component` mounted with the pane's own
 * props, as it kept the pane: it draws nothing until `open` first turns true, then reads the
 * module and, while it loads, the pane's stand-in (header, a form in shimmer, Cancel). Closing and
 * opening again does not read it again. A failed read shows the failure in the pane with a retry.
 *
 * Every opening mounts the pane fresh: a form computes its `defaultValues` at mount and needs no
 * reset effect, and a refetch of what it was opened on never touches what the person typed.
 *
 *   const CredentialFormPane = lazyWorkPane(
 *     () => import("./credential-form-pane"),
 *     (module) => module.CredentialFormPane,
 *     { title: (props, t) => (props.credential ? t("Edit credential") : t("Add credential")) },
 *   );
 *   <CredentialFormPane.Component open={…} onOpenChange={…} … />
 *
 * The module's component draws exactly one `WorkPane`.
 */
export function lazyWorkPane<M, P extends object>(
  load: () => Promise<M>,
  pick: (module: M) => FunctionComponent<P>,
  options: LazyWorkPaneOptions<P> = {},
): LazyWorkPane<P> {
  const { Component: Loaded, preload } = lazyComponent(load, pick);
  const isOpen = options.isOpen ?? defaultIsOpen;
  const close = options.close ?? defaultClose;
  const fields = options.fields ?? 5;

  function LazyPane(props: P) {
    const t = useT();
    const id = useId();
    const open = isOpen(props);
    const key = options.openKey?.(props) ?? null;
    // Each opening (and each new record while open) mounts the pane anew, so its form starts from
    // the `defaultValues` it computes at mount and no effect resets it. Closing keeps the mounted
    // pane, so it can play its exit with what it showed.
    const [mounting, setMounting] = useState({ open, key, count: open ? 1 : 0 });
    if (open !== mounting.open || (open && key !== mounting.key)) {
      setMounting({
        open,
        key: open ? key : mounting.key,
        count: open ? mounting.count + 1 : mounting.count,
      });
    }
    if (mounting.count === 0) return null;

    const title = options.title?.(props, t);
    const onClose = () => close(props);
    return (
      <WorkPaneIdentityContext value={id}>
        <ErrorBoundary
          resetKeys={[mounting.count]}
          onError={(error) => reportFailure(error, { source: "boundary" })}
          fallbackRender={({ error, resetErrorBoundary }) => (
            <PaneStandIn open={open} onClose={onClose} title={title} fields={fields}>
              <ErrorState error={error} layout="section" onRetry={resetErrorBoundary} />
            </PaneStandIn>
          )}
        >
          <Suspense
            fallback={<PaneStandIn open={open} onClose={onClose} title={title} fields={fields} />}
          >
            <Loaded key={mounting.count} {...props} />
          </Suspense>
        </ErrorBoundary>
      </WorkPaneIdentityContext>
    );
  }

  return { Component: LazyPane, preload };
}

/**
 * The props that read a lazy piece's code on intent: spread them on the control that opens it,
 * so the code is on its way while the pointer travels or focus rests.
 *
 *   <Button {...preloadOn(CredentialFormPane)} onClick={add}>Add credential</Button>
 */
export type PreloadIntent = { onPointerEnter: () => void; onFocus: () => void };

export function preloadOn(lazy: { preload: () => Promise<unknown> }): PreloadIntent {
  const warm = () => {
    lazy.preload().catch(() => undefined);
  };
  return { onPointerEnter: warm, onFocus: warm };
}
