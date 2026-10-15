import type { FactoringNoa } from "@/lib/graphql/factoring-noa";
import { lazyComponent, preloadable } from "@/lib/lazy-component";
import { LazyComponent } from "@trenova/shared/components/error-boundary";

type NoaDialogProps = {
  noa: FactoringNoa | null;
  onOpenChange: (open: boolean) => void;
};

const loadNoaDialogs = preloadable(() => import("./factoring-noa-dialogs"));

/**
 * Reads the notice dialogs' code ahead of the click: called when the pointer
 * or focus reaches a button that opens one. Repeat calls share the read.
 */
export function preloadFactoringNoaDialogs(): void {
  void loadNoaDialogs().catch(() => undefined);
}

const SendDialog = lazyComponent(
  loadNoaDialogs,
  (module) => module.SendFactoringNoaDialog,
).Component;
const AcknowledgeDialog = lazyComponent(
  loadNoaDialogs,
  (module) => module.AcknowledgeFactoringNoaDialog,
).Component;
const ReleaseDialog = lazyComponent(
  loadNoaDialogs,
  (module) => module.ReleaseFactoringNoaDialog,
).Component;

/** Emails a notice to its customer; its code is read only once a notice is chosen to send. */
export function LazySendFactoringNoaDialog({
  noa,
  onOpenChange,
  onSent,
}: NoaDialogProps & { onSent?: (noa: FactoringNoa) => void }) {
  if (!noa) return null;
  return (
    <LazyComponent fallback={null}>
      <SendDialog noa={noa} onOpenChange={onOpenChange} onSent={onSent} />
    </LazyComponent>
  );
}

/** Records the customer's signed acknowledgement; read only once a notice is chosen. */
export function LazyAcknowledgeFactoringNoaDialog({ noa, onOpenChange }: NoaDialogProps) {
  if (!noa) return null;
  return (
    <LazyComponent fallback={null}>
      <AcknowledgeDialog noa={noa} onOpenChange={onOpenChange} />
    </LazyComponent>
  );
}

/** Releases a notice; read only once a notice is chosen to release. */
export function LazyReleaseFactoringNoaDialog({ noa, onOpenChange }: NoaDialogProps) {
  if (!noa) return null;
  return (
    <LazyComponent fallback={null}>
      <ReleaseDialog noa={noa} onOpenChange={onOpenChange} />
    </LazyComponent>
  );
}
