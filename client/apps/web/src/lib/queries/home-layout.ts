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
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: HomeLayoutDocument,
        operationName: "HomeLayout",
        signal,
      }),
  }),
  catalog: () => ({
    queryKey: ["catalog"],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: HomeWidgetCatalogDocument,
        operationName: "HomeWidgetCatalog",
        signal,
      }),
  }),
  presets: () => ({
    queryKey: ["presets"],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: HomeLayoutPresetsDocument,
        operationName: "HomeLayoutPresets",
        signal,
      }),
  }),
  preset: (id: string) => ({
    queryKey: ["preset", id],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: HomeLayoutPresetDocument,
        operationName: "HomeLayoutPreset",
        variables: { id },
        signal,
      }),
  }),
  preview: (presetId: string | null, roleId: string | null) => ({
    queryKey: ["preview", presetId ?? "", roleId ?? ""],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: HomeLayoutPreviewDocument,
        operationName: "HomeLayoutPreview",
        variables: { presetId, roleId },
        signal,
      }),
  }),
});
