import {
  SidebarCustomizationOptionsDocument,
  SidebarPreferencesDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const sidebarPreferences = createQueryKeys("sidebarPreferences", {
  effective: () => ({
    queryKey: ["effective"],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: SidebarPreferencesDocument,
        operationName: "SidebarPreferences",
        signal,
      }),
  }),
  options: () => ({
    queryKey: ["options"],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: SidebarCustomizationOptionsDocument,
        operationName: "SidebarCustomizationOptions",
        signal,
      }),
  }),
});
