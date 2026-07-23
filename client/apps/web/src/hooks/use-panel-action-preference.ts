import { useLocalStorage } from "./use-local-storage";

export type CreatePanelSaveAction = "save" | "save-close" | "save-add-another";
export type EditPanelSaveAction = "save" | "save-close";

const CREATE_PANEL_KEY = "panel-create-default-action";
const EDIT_PANEL_KEY = "panel-edit-default-action";

export function useCreatePanelActionPreference() {
  return useLocalStorage<CreatePanelSaveAction>(CREATE_PANEL_KEY, "save-close");
}

export function useEditPanelActionPreference() {
  return useLocalStorage<EditPanelSaveAction>(EDIT_PANEL_KEY, "save-close");
}
