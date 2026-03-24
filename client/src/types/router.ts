export interface RouteHandle {
  crumb?: string | ((data: any) => React.ReactNode);
  title?: string | ((data: any) => string);
  showBreadcrumbs?: boolean;
}

export interface BreadcrumbMatch {
  id: string;
  pathname: string;
  params: Record<string, string>;
  data: unknown;
  handle?: RouteHandle;
}
