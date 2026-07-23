import { describe, expect, it } from "vitest";
import type {
  EDIDiagnostic,
  EDITemplateScriptLibrary,
  EDITemplateSegment,
  EDITemplateVersion,
} from "@trenova/shared/types/edi";
import {
  createTemplateDesignerStore,
  type TemplateDesignerStore,
} from "@/stores/template-designer-store";

describe("template designer store", () => {
  it("hydrates metadata, segments, and scripts from a selected version", () => {
    const store = createTemplateDesignerStore();
    const version = createVersion();

    store.getState().hydrateVersion(version);

    expect(store.getState().metadataDraft).toEqual({
      versionNotes: "Initial notes",
      x12Version: "004010",
      functionalGroupId: "SM",
    });
    expect(store.getState().segmentsDraft).toHaveLength(1);
    expect(store.getState().scriptDraft).toHaveLength(1);
    expect(store.getState().hydratedVersionKey).toBe("version-1:7");
  });

  it("deep-clones segment and element draft data before editing", () => {
    const store = createTemplateDesignerStore();
    const version = createVersion();

    store.getState().hydrateVersion(version);
    store.getState().updateElement("segment-1", 10, (element) => ({
      ...element,
      validation: { ...element.validation, required: false },
      transformPipeline: [
        {
          operation: "trim",
          arguments: { value: "changed" },
        },
      ],
    }));

    expect(version.segments[0]?.elements[0]?.validation.required).toBe(true);
    expect(version.segments[0]?.elements[0]?.transformPipeline[0]?.arguments).toEqual({
      value: "original",
    });
    expect(store.getState().segmentsDraft[0]?.elements[0]?.validation.required).toBe(false);
  });

  it("marks only the relevant dirty flag on metadata, segment, element, and script updates", () => {
    const metadataStore = createTemplateDesignerStore();
    metadataStore.getState().hydrateVersion(createVersion());
    metadataStore.getState().updateMetadata({ versionNotes: "Changed" });
    expect(readDirtyFlags(metadataStore.getState())).toEqual({
      metadataDirty: true,
      scriptsDirty: false,
      segmentsDirty: false,
    });

    const segmentStore = createTemplateDesignerStore();
    segmentStore.getState().hydrateVersion(createVersion());
    segmentStore.getState().updateSegment("segment-1", (segment) => ({
      ...segment,
      condition: "shipment.id",
    }));
    expect(readDirtyFlags(segmentStore.getState())).toEqual({
      metadataDirty: false,
      scriptsDirty: false,
      segmentsDirty: true,
    });

    const elementStore = createTemplateDesignerStore();
    elementStore.getState().hydrateVersion(createVersion());
    elementStore.getState().updateElement("segment-1", 10, (element) => ({
      ...element,
      value: "constant",
    }));
    expect(readDirtyFlags(elementStore.getState())).toEqual({
      metadataDirty: false,
      scriptsDirty: false,
      segmentsDirty: true,
    });

    const scriptStore = createTemplateDesignerStore();
    scriptStore.getState().hydrateVersion(createVersion());
    scriptStore.getState().replaceScripts([createScript({ id: "script-2" })]);
    expect(readDirtyFlags(scriptStore.getState())).toEqual({
      metadataDirty: false,
      scriptsDirty: true,
      segmentsDirty: false,
    });
  });

  it("does not mark segments dirty when stale segment or element edits target missing drafts", () => {
    const store = createTemplateDesignerStore();
    store.getState().hydrateVersion(createVersion());

    store.getState().updateSegment("missing-segment", (segment) => ({
      ...segment,
      condition: "shipment.id",
    }));
    store.getState().updateElement("segment-1", 999, (element) => ({
      ...element,
      value: "constant",
    }));

    expect(readDirtyFlags(store.getState())).toEqual({
      metadataDirty: false,
      scriptsDirty: false,
      segmentsDirty: false,
    });
  });

  it("does not overwrite dirty drafts when hydrating the same or refetched version", () => {
    const store = createTemplateDesignerStore();
    const version = createVersion();

    store.getState().hydrateVersion(version);
    store.getState().updateMetadata({ versionNotes: "Unsaved notes" });
    store.getState().hydrateVersion({
      ...version,
      notes: "Server notes",
      segments: [
        {
          ...version.segments[0]!,
          condition: "server.condition",
        },
      ],
    });

    expect(store.getState().metadataDraft.versionNotes).toBe("Unsaved notes");
    expect(store.getState().segmentsDraft[0]?.condition).toBe("");
  });

  it("refreshes an empty same-version draft when full version details arrive", () => {
    const store = createTemplateDesignerStore();
    const version = createVersion();

    store.getState().hydrateVersion({
      ...version,
      segments: [],
      scriptLibraries: [],
    });
    store.getState().hydrateVersion(version);

    expect(store.getState().hydratedVersionKey).toBe("version-1:7");
    expect(store.getState().segmentsDraft).toHaveLength(1);
    expect(store.getState().scriptDraft).toHaveLength(1);
  });

  it("clears diagnostics and dirty flags on reset", () => {
    const store = createTemplateDesignerStore();
    const diagnostics: EDIDiagnostic[] = [
      {
        severity: "Error",
        code: "EDI001",
        segmentId: "B2",
        elementPosition: 10,
        path: "B2.10",
        message: "Missing value",
        suggestedFix: null,
      },
    ];

    store.getState().hydrateVersion(createVersion());
    store.getState().updateMetadata({ versionNotes: "Changed" });
    store.getState().updateSegment("segment-1", (segment) => ({
      ...segment,
      condition: "shipment.id",
    }));
    store.getState().replaceScripts([createScript({ id: "script-2" })]);
    store.getState().setDiagnostics(diagnostics);
    store.getState().resetDraftState();

    expect(store.getState().diagnostics).toEqual([]);
    expect(readDirtyFlags(store.getState())).toEqual({
      metadataDirty: false,
      scriptsDirty: false,
      segmentsDirty: false,
    });
    expect(store.getState().segmentsDraft).toEqual([]);
    expect(store.getState().hydratedVersionKey).toBe("");
  });
});

function readDirtyFlags(state: TemplateDesignerStore) {
  return {
    metadataDirty: state.metadataDirty,
    scriptsDirty: state.scriptsDirty,
    segmentsDirty: state.segmentsDirty,
  };
}

function createVersion(overrides: Partial<EDITemplateVersion> = {}): EDITemplateVersion {
  return {
    id: "version-1",
    businessUnitId: "bu-1",
    organizationId: "org-1",
    templateId: "template-1",
    sourceVersionId: null,
    versionNumber: 1,
    x12Version: "004010",
    functionalGroupId: "SM",
    status: "Draft",
    isActive: false,
    notes: "Initial notes",
    certificationNotes: null,
    activationNotes: null,
    archiveNotes: null,
    deprecatedNotes: null,
    supersededNotes: null,
    certifiedAt: null,
    activatedAt: null,
    archivedAt: null,
    deprecatedAt: null,
    supersededAt: null,
    version: 7,
    createdAt: null,
    updatedAt: null,
    segments: [createSegment()],
    scriptLibraries: [createScript()],
    ...overrides,
  };
}

function createSegment(overrides: Partial<EDITemplateSegment> = {}): EDITemplateSegment {
  return {
    id: "segment-1",
    templateVersionId: "version-1",
    segmentId: "B2",
    name: "Beginning Segment",
    sequence: 1,
    loopId: null,
    repeatPath: null,
    condition: "",
    required: true,
    maxUse: 1,
    usageNotes: null,
    elements: [
      {
        position: 10,
        name: "Shipment ID",
        source: "fieldPath",
        value: null,
        fieldPath: "shipment.id",
        partnerSettingPath: null,
        mappingEntityType: null,
        mappingSourcePath: null,
        runtimeKey: null,
        repeatPath: null,
        baseSource: {
          source: "fieldPath",
          value: null,
          fieldPath: "shipment.id",
          partnerSettingPath: null,
          mappingEntityType: null,
          mappingSourcePath: null,
          runtimeKey: null,
          repeatPath: null,
          default: null,
        },
        transformPipeline: [{ operation: "trim", arguments: { value: "original" } }],
        starlarkFunction: null,
        starlarkScript: null,
        default: null,
        condition: null,
        implementationGuideNote: null,
        validation: {
          required: true,
          maxLength: 30,
          minLength: 0,
          code: null,
          message: null,
        },
      },
    ],
    ...overrides,
  };
}

function createScript(overrides: Partial<EDITemplateScriptLibrary> = {}): EDITemplateScriptLibrary {
  return {
    id: "script-1",
    businessUnitId: "bu-1",
    organizationId: "org-1",
    templateVersionId: "version-1",
    name: "normalizers",
    description: "",
    language: "Starlark",
    script: "def normalize(value):\n    return value\n",
    status: "Draft",
    version: 1,
    createdAt: null,
    updatedAt: null,
    functionNames: ["normalize"],
    ...overrides,
  };
}
