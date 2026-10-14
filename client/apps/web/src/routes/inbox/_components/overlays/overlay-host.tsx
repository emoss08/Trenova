import { inbox } from "@/lib/queries/inbox";
import { useTriageStore } from "@/stores/triage-store";
import { useQuery } from "@tanstack/react-query";
import { lazy, Suspense, useCallback, type ReactNode } from "react";
import { useSearchParams } from "react-router";
import { WAITING, folderParams, searchParamsWith } from "../folders";
import { useTriageNavigation } from "../triage-navigation";

const DocumentViewer = lazy(() =>
  import("./document/document-viewer").then((module) => ({ default: module.DocumentViewer })),
);
const ShipmentPeek = lazy(() =>
  import("./peeks/shipment-peek").then((module) => ({ default: module.ShipmentPeek })),
);
const CustomerPeek = lazy(() =>
  import("./peeks/customer-peek").then((module) => ({ default: module.CustomerPeek })),
);
const DeskHandoff = lazy(() =>
  import("./desk/desk-handoff").then((module) => ({ default: module.DeskHandoff })),
);
const RuleMatchesDialog = lazy(() =>
  import("./rule-matches/rule-matches-dialog").then((module) => ({
    default: module.RuleMatchesDialog,
  })),
);
const ConnectSource = lazy(() =>
  import("./connect/connect-source").then((module) => ({ default: module.ConnectSource })),
);
const ShortcutsSheet = lazy(() =>
  import("../shortcuts-sheet").then((module) => ({ default: module.ShortcutsSheet })),
);
const LinkMessageDialog = lazy(() =>
  import("../link-dialog").then((module) => ({ default: module.LinkMessageDialog })),
);

/**
 * Everything drawn over Triage, in one place: whichever overlay the store
 * names, from whatever asked for it (a key, a button, a menu item). Each
 * overlay is its own chunk, read the first time it opens, and closes with
 * Esc or by clicking outside it.
 */
export function TriageOverlayHost() {
  const overlay = useTriageStore((store) => store.overlay);
  const close = useTriageStore((store) => store.closeOverlay);
  const navigation = useTriageNavigation();
  const [, setSearchParams] = useSearchParams();

  const openMessage = useCallback(
    (id: string) =>
      navigation.filter() === null ? navigation.openInQueue(id) : navigation.open(id),
    [navigation],
  );
  const openWaitingFor = useCallback(
    (search: string) =>
      setSearchParams((current) => searchParamsWith(folderParams(current, WAITING), search)),
    [setSearchParams],
  );

  if (overlay === null) {
    return null;
  }

  let drawn: ReactNode;
  switch (overlay.kind) {
    case "doc":
      drawn = (
        <DocumentViewer
          messageId={overlay.messageId}
          attachmentId={overlay.attachmentId}
          onClose={close}
        />
      );
      break;
    case "shipmentPeek":
      drawn = <ShipmentPeek shipmentId={overlay.shipmentId} onClose={close} />;
      break;
    case "customerPeek":
      drawn = (
        <CustomerPeek
          customerId={overlay.customerId}
          messageId={overlay.messageId}
          onClose={close}
          onOpenMessage={openMessage}
          onOpenWaiting={openWaitingFor}
        />
      );
      break;
    case "desk":
      drawn = <DeskHandoff messageId={overlay.messageId} onClose={close} />;
      break;
    case "ruleMatches":
      drawn = <RuleMatchesDialog target={overlay.target} onClose={close} />;
      break;
    case "connect":
      drawn = <ConnectSource mailboxId={overlay.mailboxId} onClose={close} />;
      break;
    case "shortcuts":
      drawn = <ShortcutsSheet open onClose={close} />;
      break;
    case "link":
      drawn = <LinkOverlay messageId={overlay.messageId} onClose={close} />;
      break;
  }

  return <Suspense fallback={null}>{drawn}</Suspense>;
}

function LinkOverlay({ messageId, onClose }: { messageId: string; onClose: () => void }) {
  const message = useQuery(inbox.message(messageId)).data;
  if (message === undefined) {
    return null;
  }

  return (
    <LinkMessageDialog
      message={message}
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      onLinked={onClose}
    />
  );
}
