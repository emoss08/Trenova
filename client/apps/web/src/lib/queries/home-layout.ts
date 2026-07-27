import {
  HomeLayoutDocument,
  HomeLayoutPresetDocument,
  HomeLayoutPresetsDocument,
  HomeLayoutPreviewDocument,
  HomeWidgetCatalogDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const homeLayout = createQueryKeys("homeLayout", {
  effective: () => ({
    queryKey: ["effective"],
    queryFn: async () =>
      requestGraphQL({
        document: HomeLayoutDocument,
        operationName: "HomeLayout",
      }),
  }),
  catalog: () => ({
    queryKey: ["catalog"],
    queryFn: async () =>
      requestGraphQL({
        document: HomeWidgetCatalogDocument,
        operationName: "HomeWidgetCatalog",
      }),
  }),
  presets: () => ({
    queryKey: ["presets"],
    queryFn: async () =>
      requestGraphQL({
        document: HomeLayoutPresetsDocument,
        operationName: "HomeLayoutPresets",
      }),
  }),
  preset: (id: string) => ({
    queryKey: ["preset", id],
    queryFn: async () =>
      requestGraphQL({
        document: HomeLayoutPresetDocument,
        operationName: "HomeLayoutPreset",
        variables: { id },
      }),
  }),
  preview: (presetId: string | null, roleId: string | null) => ({
    queryKey: ["preview", presetId ?? "", roleId ?? ""],
    queryFn: async () =>
      requestGraphQL({
        document: HomeLayoutPreviewDocument,
        operationName: "HomeLayoutPreview",
        variables: { presetId, roleId },
      }),
  }),
});
