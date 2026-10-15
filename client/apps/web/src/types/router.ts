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

export interface RouteHandle {
  crumb?: string | ((data: any) => React.ReactNode);
  title?: string | ((data: any) => string);
  showBreadcrumbs?: boolean;
  frame?: ShellFrame;
  roomRail?: RoomRail;
}

export interface BreadcrumbMatch {
  id: string;
  pathname: string;
  params: Record<string, string>;
  data: unknown;
  handle?: RouteHandle;
}
