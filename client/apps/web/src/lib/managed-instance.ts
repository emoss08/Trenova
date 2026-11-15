import { queryClient } from "@/lib/query-client";
import { publicConfigQueryOptions } from "@trenova/shared/hooks/use-public-config";
import type { LoaderFunction } from "react-router";

/**
 * Keeps a page that only a Managed Trenova instance has out of reach everywhere else. The public
 * config falls back to self-hosted when it cannot be read, so an install it cannot vouch for
 * never shows the page.
 */
export const managedInstanceLoader: LoaderFunction = async () => {
  const config = await queryClient.ensureQueryData(publicConfigQueryOptions);
  if (config.managed) {
    return null;
  }

  throw new Response("Not Found", { status: 404, statusText: "Not Found" });
};
