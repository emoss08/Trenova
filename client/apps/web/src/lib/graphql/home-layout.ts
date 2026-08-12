import {
  CreateHomeLayoutPresetDocument,
  DeleteHomeLayoutPresetDocument,
  HomeLayoutFieldsFragmentDoc,
  HomeLayoutPresetFieldsFragmentDoc,
  HomeWidgetFieldsFragmentDoc,
  ResetHomeLayoutDocument,
  UpdateHomeLayoutDocument,
  UpdateHomeLayoutPresetDocument,
  type HomeLayoutFieldsFragment,
  type HomeLayoutInput,
  type HomeLayoutPresetFieldsFragment,
  type HomeWidgetCatalogQuery,
  type HomeWidgetFieldsFragment,
  type HomeWidgetInput,
  type SaveHomeLayoutPresetInput,
  type UpdateHomeLayoutPresetInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData, type FragmentType } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type HomeWidget = HomeWidgetFieldsFragment;
export type HomeWidgetConfig = HomeWidget["config"];

/**
 * The layout with its widgets unmasked. Every consumer needs the widget fields
 * — a canvas cannot render a fragment reference — so masking is undone once
 * here rather than at each of the two dozen call sites.
 */
export type HomeLayout = Omit<HomeLayoutFieldsFragment, "widgets"> & {
  widgets: HomeWidget[];
};

export type HomeLayoutPreset = Omit<HomeLayoutPresetFieldsFragment, "widgets"> & {
  widgets: HomeWidget[];
};

export type HomeLayoutSource = HomeLayoutFieldsFragment["source"];
export type HomeWidgetCatalog = HomeWidgetCatalogQuery["homeWidgetCatalog"];
export type HomeWidgetOption = HomeWidgetCatalog["widgets"][number];
export type HomeWidgetCategory = HomeWidgetCatalog["categories"][number];
export type HomeMetricOption = HomeWidgetCatalog["metrics"][number];

function unmaskWidgets(
  widgets: readonly FragmentType<typeof HomeWidgetFieldsFragmentDoc>[],
): HomeWidget[] {
  return widgets.map((widget) => getFragmentData(HomeWidgetFieldsFragmentDoc, widget));
}

export function toHomeLayout(masked: FragmentType<typeof HomeLayoutFieldsFragmentDoc>): HomeLayout {
  const layout = getFragmentData(HomeLayoutFieldsFragmentDoc, masked);
  return { ...layout, widgets: unmaskWidgets(layout.widgets) };
}

export function toHomeLayoutPreset(
  masked: FragmentType<typeof HomeLayoutPresetFieldsFragmentDoc>,
): HomeLayoutPreset {
  const preset = getFragmentData(HomeLayoutPresetFieldsFragmentDoc, masked);
  return { ...preset, widgets: unmaskWidgets(preset.widgets) };
}

/**
 * What the canvas renders before the server answers. It is deliberately empty
 * rather than a guessed layout: widgets that appear and then vanish read as a
 * bug, while an empty canvas that fills in reads as loading.
 */
export const DEFAULT_HOME_LAYOUT: HomeLayout = {
  schemaVersion: 1,
  version: 0,
  source: "BUILT_IN",
  presetId: null,
  presetName: null,
  locked: false,
  canCustomize: true,
  density: "comfortable",
  widgets: [],
};

/**
 * Strips a rendered widget down to the fields the mutation accepts. The
 * fragment carries `__typename`s the input type rejects, so the mapping is
 * explicit rather than a spread.
 */
export function toWidgetInput(widget: HomeWidget): HomeWidgetInput {
  return {
    id: widget.id,
    key: widget.key,
    title: widget.title,
    w: widget.w,
    h: widget.h,
    config: {
      metric: widget.config.metric,
      metrics: widget.config.metrics,
      definitionId: widget.config.definitionId,
      cannedKey: widget.config.cannedKey,
      chartId: widget.config.chartId,
      columnId: widget.config.columnId,
      dashboardId: widget.config.dashboardId,
      text: widget.config.text,
      limit: widget.config.limit,
      windowDays: widget.config.windowDays,
    },
  };
}

export async function updateHomeLayout(input: HomeLayoutInput): Promise<HomeLayout> {
  const data = await requestGraphQL({
    document: UpdateHomeLayoutDocument,
    operationName: "UpdateHomeLayout",
    variables: { input },
  });

  return toHomeLayout(data.updateHomeLayout);
}

export async function resetHomeLayout(): Promise<HomeLayout> {
  const data = await requestGraphQL({
    document: ResetHomeLayoutDocument,
    operationName: "ResetHomeLayout",
  });

  return toHomeLayout(data.resetHomeLayout);
}

export async function createHomeLayoutPreset(
  input: SaveHomeLayoutPresetInput,
): Promise<HomeLayoutPreset> {
  const data = await requestGraphQL({
    document: CreateHomeLayoutPresetDocument,
    operationName: "CreateHomeLayoutPreset",
    variables: { input },
  });

  return toHomeLayoutPreset(data.createHomeLayoutPreset);
}

export async function updateHomeLayoutPreset(
  input: UpdateHomeLayoutPresetInput,
): Promise<HomeLayoutPreset> {
  const data = await requestGraphQL({
    document: UpdateHomeLayoutPresetDocument,
    operationName: "UpdateHomeLayoutPreset",
    variables: { input },
  });

  return toHomeLayoutPreset(data.updateHomeLayoutPreset);
}

export async function deleteHomeLayoutPreset(id: string): Promise<boolean> {
  const data = await requestGraphQL({
    document: DeleteHomeLayoutPresetDocument,
    operationName: "DeleteHomeLayoutPreset",
    variables: { id },
  });

  return data.deleteHomeLayoutPreset;
}
