import { useQuery } from "@tanstack/react-query";
import { createContext, useContext } from "react";
import { loadCatalog, type CatalogIndex } from "./catalog";

const CatalogContext = createContext<CatalogIndex | null>(null);

export const CatalogProvider = CatalogContext.Provider;

/**
 * The loaded catalog. Only valid below the provider the explorer page mounts
 * once the fetch resolves, which is why every consumer can treat it as present.
 */
export function useCatalog(): CatalogIndex {
  const index = useContext(CatalogContext);
  if (!index) {
    throw new Error("useCatalog must be used inside the GraphQL explorer's CatalogProvider");
  }
  return index;
}

export function useCatalogQuery() {
  return useQuery({
    queryKey: ["graphql-operation-catalog"],
    queryFn: loadCatalog,
    // Generated at build time — it cannot change while the tab is open.
    staleTime: Infinity,
    gcTime: Infinity,
  });
}
