import { AppSession } from "@/routes/app-layout";

/**
 * The Desk's outermost frame, which is to say: none.
 *
 * Everything the app normally wraps a page in — the sidebar, the
 * breadcrumb, the page header — is chrome that explains where you are.
 * The Desk does not need explaining; it is one room and you are in it. So
 * this renders the session's obligations and then gets out of the way,
 * handing the whole viewport to the conversation and the work beside it.
 */
export function DeskShellLayout() {
  return <AppSession>{(outlet) => outlet}</AppSession>;
}
