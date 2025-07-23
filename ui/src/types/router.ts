/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

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
