/**
 * What a route gate throws when the person may not open the page: no permission, a capability
 * switched off, a feature the plan leaves out, a page this install does not have. The route's
 * error component presents it through describeError, the way a refused read is presented.
 * `data` is the body the API would send for the same refusal, or the words to show.
 */
export class RouteRefusal extends Error {
  readonly status: number;
  readonly statusText: string;
  readonly data: unknown;

  constructor({
    status,
    statusText,
    data,
  }: {
    status: number;
    statusText: string;
    data?: unknown;
  }) {
    super(statusText);
    this.name = "RouteRefusal";
    this.status = status;
    this.statusText = statusText;
    this.data = data;
  }
}

export function isRouteRefusal(error: unknown): error is RouteRefusal {
  return error instanceof RouteRefusal;
}
