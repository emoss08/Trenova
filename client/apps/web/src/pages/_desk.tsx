import { createFileRoute } from "@tanstack/react-router";
import { SHOWN_AT_ONCE, protectedGate } from "@/lib/route-gates";
import { DeskLoadingScreen } from "@/routes/desk/desk-loading-screen";
import { DeskShellLayout } from "@/routes/desk/shell-layout";

// The Desk runs outside the app chrome.
//
// It is a room rather than a page: a conversation on one side and
// the work it produces on the other, both of which want the whole
// window. Mounted under AppLayout it paid for a sidebar, a
// breadcrumb and a page header before a word of conversation —
// roughly 190px of furniture on a surface whose entire job is to
// give the work room. It keeps the protected loader, so it is no
// less guarded than anything inside; it simply wears none of the
// frame. The way back is in its own chrome.
export const Route = createFileRoute("/_desk")({
  beforeLoad: protectedGate(),
  component: DeskShellLayout,
  pendingComponent: DeskLoadingScreen,
  ...SHOWN_AT_ONCE,
});
