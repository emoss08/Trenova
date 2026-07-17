import {
  SidebarCustomizationOptionsDocument,
  SidebarPreferencesDocument,
} from "@/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const sidebarPreferences = createQueryKeys("sidebarPreferences", {
  effective: () => ({
    queryKey: ["effective"],
    queryFn: async () =>
      requestGraphQL({
        document: SidebarPreferencesDocument,
        operationName: "SidebarPreferences",
      }),
  }),
  options: () => ({
    queryKey: ["options"],
    queryFn: async () =>
      requestGraphQL({
        document: SidebarCustomizationOptionsDocument,
        operationName: "SidebarCustomizationOptions",
      }),
  }),
});
