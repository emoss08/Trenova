/**
 * The work pane: a form or a record's details opened beside the list it belongs to, instead of
 * a dialog over it. Decisions with one or two fields and every confirmation stay dialogs.
 *
 * One host per page:
 *
 *   <WorkPaneProvider onDirtyChange={setPaneDirty}>      // above the header, tabs and guard
 *     <UnsavedChangesGuard when={profileDirty || paneDirty} />
 *     <RecordHeader … />                                  // its menus call useWorkPane().closeThen
 *     <WorkPaneHost>{tab content}</WorkPaneHost>          // the row the pane opens beside
 *   </WorkPaneProvider>
 *
 * A pane is a `WorkPane` kept mounted in the tab that owns it, with `open` driven by the page's
 * address. Inside it: `WorkPaneBody` (the description line, then the fields) and
 * `WorkPaneFooter` (`WorkPaneCancel`, then the primary; `leading` for secondary and destructive
 * actions), directly or inside the form's `<Form>`. A form adds `<WorkPaneDirtyTracker form />`
 * so Escape, the close button, Cancel and `closeThen` ask before discarding its changes.
 *
 * - Wrap every tab, view or pane change in `useWorkPane().closeThen(then)`: a dirty pane asks
 *   first in its footer; Discard closes it (its `onOpenChange(false)`) and then runs `then`.
 * - Only one pane shows at a time; opening a second closes the first without asking, which is
 *   why openers go through `closeThen`.
 * - A pane in a hidden tab (an `Activity` that is hidden) gives up the host and comes back with
 *   its state when the tab is shown again.
 * - When the row, measured with `useElementWidth`, is under 1100px wide, the pane lies over the
 *   content as a sheet with a scrim, `min(26rem, row width)` wide.
 * - The host never traps fixed-position children: its row and content box carry no container
 *   query, `contain`, transform, filter or perspective, so a save dock, `ActionDock` or the
 *   documents upload panel inside a tab stays on the viewport. Keep it that way.
 * - The host is a containing block (an inline-size container), so nothing `position: fixed` goes
 *   inside it: a page's save dock is mounted beside the host, not in a tab within it.
 */
export { useWorkPane, type WorkPaneControls } from "./work-pane-context";
export { WorkPaneHost } from "./work-pane-host";
export { WorkPaneProvider } from "./work-pane-provider";
export {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
  type WorkPaneProps,
} from "./work-pane";
