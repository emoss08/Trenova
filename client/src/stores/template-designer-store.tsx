import type {
  EDIDiagnostic,
  EDITemplateElement,
  EDITemplateScriptLibrary,
  EDITemplateSegment,
  EDITemplateVersion,
} from "@/types/edi";
import { createContext, useContext, useRef, type ReactNode } from "react";
import { useStore } from "zustand";
import { createStore, type StoreApi } from "zustand/vanilla";
import { cloneSegments } from "@/routes/edi/_components/designer/utils/edi-designer-utils";

export type TemplateDesignerMetadataDraft = {
  versionNotes: string;
  x12Version: string;
  functionalGroupId: string;
};

type TemplateDesignerDirtyState = {
  segmentsDirty: boolean;
  scriptsDirty: boolean;
  metadataDirty: boolean;
};

type TemplateDesignerDraftState = TemplateDesignerDirtyState & {
  segmentsDraft: EDITemplateSegment[];
  scriptDraft: EDITemplateScriptLibrary[];
  metadataDraft: TemplateDesignerMetadataDraft;
  diagnostics: EDIDiagnostic[];
  hydratedVersionKey: string;
};

type TemplateDesignerStoreActions = {
  hydrateVersion: (version: EDITemplateVersion | undefined) => void;
  resetDraftState: () => void;
  clearDirtyState: () => void;
  clearSegmentsDirty: () => void;
  clearScriptsDirty: () => void;
  clearMetadataDirty: () => void;
  updateMetadata: (patch: Partial<TemplateDesignerMetadataDraft>) => void;
  updateSegment: (
    segmentId: string,
    updater: (segment: EDITemplateSegment) => EDITemplateSegment,
  ) => void;
  updateElement: (
    segmentId: string,
    position: number,
    updater: (element: EDITemplateElement) => EDITemplateElement,
  ) => void;
  replaceScripts: (libraries: EDITemplateScriptLibrary[]) => void;
  setDiagnostics: (diagnostics: EDIDiagnostic[]) => void;
};

export type TemplateDesignerStore = TemplateDesignerDraftState & TemplateDesignerStoreActions;

const defaultMetadataDraft: TemplateDesignerMetadataDraft = {
  versionNotes: "",
  x12Version: "004010",
  functionalGroupId: "SM",
};

const initialDraftState: TemplateDesignerDraftState = {
  segmentsDraft: [],
  scriptDraft: [],
  metadataDraft: defaultMetadataDraft,
  segmentsDirty: false,
  scriptsDirty: false,
  metadataDirty: false,
  diagnostics: [],
  hydratedVersionKey: "",
};

export function getTemplateDesignerVersionDraftKey(version: EDITemplateVersion | undefined) {
  return version ? `${version.id}:${version.version}` : "";
}

export function createTemplateDesignerStore() {
  return createStore<TemplateDesignerStore>()((set, get) => ({
    ...initialDraftState,
    metadataDraft: { ...defaultMetadataDraft },
    hydrateVersion: (version) => {
      const versionKey = getTemplateDesignerVersionDraftKey(version);
      if (!versionKey || !version) return;

      const state = get();
      if (state.hydratedVersionKey === versionKey) return;
      if (state.segmentsDirty || state.scriptsDirty || state.metadataDirty) return;

      set({
        segmentsDraft: cloneSegments(version.segments),
        scriptDraft: cloneScriptLibraries(version.scriptLibraries),
        metadataDraft: {
          versionNotes: version.notes ?? "",
          x12Version: version.x12Version,
          functionalGroupId: version.functionalGroupId,
        },
        diagnostics: [],
        hydratedVersionKey: versionKey,
        segmentsDirty: false,
        scriptsDirty: false,
        metadataDirty: false,
      });
    },
    resetDraftState: () => {
      set({
        ...initialDraftState,
        metadataDraft: { ...defaultMetadataDraft },
      });
    },
    clearDirtyState: () => {
      set({
        segmentsDirty: false,
        scriptsDirty: false,
        metadataDirty: false,
      });
    },
    clearSegmentsDirty: () => set({ segmentsDirty: false }),
    clearScriptsDirty: () => set({ scriptsDirty: false }),
    clearMetadataDirty: () => set({ metadataDirty: false }),
    updateMetadata: (patch) => {
      set((state) => ({
        metadataDraft: { ...state.metadataDraft, ...patch },
        metadataDirty: true,
      }));
    },
    updateSegment: (segmentId, updater) => {
      set((state) => {
        let updated = false;
        const segmentsDraft = state.segmentsDraft.map((segment) => {
          if (segment.id !== segmentId) return segment;
          updated = true;
          return updater(cloneSegment(segment));
        });

        return updated ? { segmentsDraft, segmentsDirty: true } : state;
      });
    },
    updateElement: (segmentId, position, updater) => {
      set((state) => {
        let updated = false;
        const segmentsDraft = state.segmentsDraft.map((segment) => {
          if (segment.id !== segmentId) return segment;
          return {
            ...segment,
            elements: segment.elements.map((element) => {
              if (element.position !== position) return element;
              updated = true;
              return updater(cloneElement(element));
            }),
          };
        });

        return updated ? { segmentsDraft, segmentsDirty: true } : state;
      });
    },
    replaceScripts: (libraries) => {
      set({
        scriptDraft: cloneScriptLibraries(libraries),
        scriptsDirty: true,
      });
    },
    setDiagnostics: (diagnostics) => {
      set({ diagnostics: diagnostics.map((diagnostic) => ({ ...diagnostic })) });
    },
  }));
}

const TemplateDesignerStoreContext = createContext<StoreApi<TemplateDesignerStore> | null>(null);

export function TemplateDesignerStoreProvider({ children }: { children: ReactNode }) {
  const storeRef = useRef<StoreApi<TemplateDesignerStore> | null>(null);
  if (!storeRef.current) {
    storeRef.current = createTemplateDesignerStore();
  }

  return (
    <TemplateDesignerStoreContext.Provider value={storeRef.current}>
      {children}
    </TemplateDesignerStoreContext.Provider>
  );
}

export function useTemplateDesignerStore<T>(selector: (state: TemplateDesignerStore) => T): T {
  const store = useContext(TemplateDesignerStoreContext);
  if (!store) {
    throw new Error("useTemplateDesignerStore must be used within TemplateDesignerStoreProvider");
  }

  return useStore(store, selector);
}

function cloneScriptLibraries(libraries: EDITemplateScriptLibrary[]) {
  return libraries.map((library) => ({
    ...library,
    functionNames: [...library.functionNames],
  }));
}

function cloneSegment(segment: EDITemplateSegment) {
  return cloneSegments([segment])[0] ?? segment;
}

function cloneElement(element: EDITemplateElement): EDITemplateElement {
  return {
    ...element,
    baseSource: element.baseSource ? { ...element.baseSource } : null,
    transformPipeline: element.transformPipeline.map((step) => ({
      operation: step.operation,
      arguments: { ...step.arguments },
    })),
    validation: { ...element.validation },
  };
}
