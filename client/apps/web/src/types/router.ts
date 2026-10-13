/**
 * How the app shell frames a route. A page sits under the sidebar and the bar
 * that says where you are. A room keeps the app header and nothing else: it
 * brings its own rail in the sidebar's place, folded by the same control and
 * key, and its own title in that rail.
 */
export type ShellFrame = "page" | "room";

export interface RouteHandle {
  crumb?: string | ((data: any) => React.ReactNode);
  title?: string | ((data: any) => string);
  showBreadcrumbs?: boolean;
  frame?: ShellFrame;
}

export interface BreadcrumbMatch {
  id: string;
  pathname: string;
  params: Record<string, string>;
  data: unknown;
  handle?: RouteHandle;
}
