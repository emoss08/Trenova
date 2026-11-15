/**
 * How the app shell frames a route. A page sits under the sidebar and the bar
 * that says where you are. A room keeps the app header and nothing else: it
 * brings its own rail in the sidebar's place, folded by the same control and
 * key, and its own title in that rail.
 */
export type ShellFrame = "page" | "room";

/**
 * Whether a room's own rail folds with the sidebar's control and key, or
 * always stands, with the module panel kept folded beside it so the rail's
 * tiles open their module's pages on hover.
 */
export type RoomRail = "folds" | "stands";

/** What a route tells the shell about itself, as its `staticData`. */
export interface RouteHandle {
  frame?: ShellFrame;
  roomRail?: RoomRail;
}

declare module "@tanstack/react-router" {
  interface StaticDataRouteOption extends RouteHandle {}

  interface HistoryState {
    /** The case was opened to be worked: an unowned case becomes the person's (START_CASE). */
    startCase?: boolean;
  }
}

/** A matched route as the shell reads it: only what it said about itself. */
export interface HandleMatch {
  staticData?: RouteHandle;
}
