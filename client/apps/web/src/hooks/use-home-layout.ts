import {
  DEFAULT_HOME_LAYOUT,
  createHomeLayoutPreset,
  deleteHomeLayoutPreset,
  resetHomeLayout,
  toHomeLayout,
  toHomeLayoutPreset,
  updateHomeLayout,
  updateHomeLayoutPreset,
  type HomeLayout,
} from "@/lib/graphql/home-layout";
import { queries } from "@/lib/queries";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

const HOME_LAYOUT_STALE_TIME = 60_000;
const HOME_CATALOG_STALE_TIME = 5 * 60_000;

export function useHomeLayout() {
  return useQuery({
    ...queries.homeLayout.effective(),
    staleTime: HOME_LAYOUT_STALE_TIME,
    select: (data): HomeLayout => toHomeLayout(data.homeLayout),
  });
}

/**
 * The catalog changes only when the deployed build or the viewer's permissions
 * do, so it is fetched once and held. It is loaded on demand rather than with
 * the layout: a viewer who never opens the customize flow never needs it.
 */
export function useHomeWidgetCatalog(enabled = true) {
  return useQuery({
    ...queries.homeLayout.catalog(),
    enabled,
    staleTime: HOME_CATALOG_STALE_TIME,
    select: (data) => data.homeWidgetCatalog,
  });
}

export function useHomeLayoutPresets(enabled = true) {
  return useQuery({
    ...queries.homeLayout.presets(),
    enabled,
    select: (data) => data.homeLayoutPresets.map(toHomeLayoutPreset),
  });
}

export function useHomeLayoutPreset(id: string | undefined) {
  return useQuery({
    ...queries.homeLayout.preset(id ?? ""),
    enabled: Boolean(id),
    select: (data) => toHomeLayoutPreset(data.homeLayoutPreset),
  });
}

export function useHomeLayoutPreview(presetId: string | null, roleId: string | null) {
  return useQuery({
    ...queries.homeLayout.preview(presetId, roleId),
    enabled: roleId != null,
    select: (data): HomeLayout => toHomeLayout(data.homeLayoutPreview),
  });
}

/**
 * Writes the server's answer straight into the cache. The server re-resolves
 * after every save — narrowing by permission and dropping retired widgets — so
 * what comes back is what will render, and echoing the request instead would
 * show the user a layout the next load would contradict.
 */
function useHomeLayoutWriter<TArgs>(mutationFn: (args: TArgs) => Promise<HomeLayout>) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn,
    onSuccess: (updated) => {
      queryClient.setQueryData(queries.homeLayout.effective().queryKey, {
        homeLayout: updated,
      });
      void queryClient.invalidateQueries({
        queryKey: queries.homeLayout.effective().queryKey,
      });
    },
  });
}

export function useUpdateHomeLayout() {
  return useHomeLayoutWriter(updateHomeLayout);
}

export function useResetHomeLayout() {
  return useHomeLayoutWriter(() => resetHomeLayout());
}

function useHomeLayoutPresetWriter<TArgs, TResult>(mutationFn: (args: TArgs) => Promise<TResult>) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn,
    onSuccess: () => {
      // A preset edit can change what every user resolves to, including this
      // one, so everything under the homeLayout key space — the list, the
      // edited preset itself, previews, and the viewer's own layout — goes
      // stale together. Invalidating the preset detail is what keeps a second
      // consecutive save from carrying a stale version into the optimistic
      // lock.
      void queryClient.invalidateQueries({ queryKey: queries.homeLayout._def });
    },
  });
}

export function useCreateHomeLayoutPreset() {
  return useHomeLayoutPresetWriter(createHomeLayoutPreset);
}

export function useUpdateHomeLayoutPreset() {
  return useHomeLayoutPresetWriter(updateHomeLayoutPreset);
}

export function useDeleteHomeLayoutPreset() {
  return useHomeLayoutPresetWriter(deleteHomeLayoutPreset);
}

export { DEFAULT_HOME_LAYOUT };
