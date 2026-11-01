import {
  ActivateEdiLadingVersionDocument,
  ArchiveEdiLadingVersionDocument,
  CertifyEdiLadingVersionDocument,
  CreateEdiLadingDraftDocument,
  CreateEdiLadingProjectDocument,
  EdiFlowInstanceDetailDocument,
  EdiLadingContextDocument,
  EdiLadingPartnerProjectDocument,
  EdiLadingReplaysDocument,
  EdiLadingShadowResultsDocument,
  EdiLadingVersionDocument,
  EdiLadingVersionsDocument,
  ReleaseEdiFlowInstanceDocument,
  RunEdiLadingVersionDocument,
  StartEdiLadingReplayDocument,
  StartEdiLadingShadowDocument,
  StopEdiLadingShadowDocument,
  UpdateEdiLadingSettingsDocument,
} from "@trenova/graphql/generated/graphql";
import type { LadingContext, LadingRunInput, LadingRunResult } from "@trenova/shared/edi/lading";
import { api } from "@trenova/shared/lib/api";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  flowInstanceDetailSchema,
  ladingProjectSchema,
  ladingReplaySchema,
  ladingShadowReportSchema,
  ladingVersionDetailSchema,
  ladingVersionSchema,
  type CreateLadingProjectRequest,
  type SaveLadingDraftRequest,
  type StartLadingReplayRequest,
} from "@trenova/shared/types/lading";

type Options = { signal?: AbortSignal };

/**
 * The Lading studio's calls (docs/engineering/lading.md, "Versions, replay and shadow"). Saving a
 * draft stays a REST upload: it carries the project's files and samples, past the GraphQL body limit.
 */
export class LadingService {
  /** The organisation and custom fields projects are checked against, as the server checks them. */
  public async getContext(options?: Options) {
    const response = await requestGraphQL({
      document: EdiLadingContextDocument,
      operationName: "EdiLadingContext",
      signal: options?.signal,
    });
    return response.ediLadingContext as LadingContext;
  }

  public async createProject(request: CreateLadingProjectRequest) {
    const response = await requestGraphQL({
      document: CreateEdiLadingProjectDocument,
      operationName: "CreateEdiLadingProject",
      variables: { input: request },
    });
    return response.createEdiLadingProject;
  }

  /** The partner's project, or null for a partner that has none yet. */
  public async getPartnerProject(partnerId: string, options?: Options) {
    const response = await requestGraphQL({
      document: EdiLadingPartnerProjectDocument,
      operationName: "EdiLadingPartnerProject",
      variables: { partnerId },
      signal: options?.signal,
    });
    const project = response.ediLadingPartnerProject;
    if (project === null || project === undefined) return null;
    return safeParse(ladingProjectSchema, project, "LadingProject");
  }

  public async listVersions(projectId: string, options?: Options) {
    const response = await requestGraphQL({
      document: EdiLadingVersionsDocument,
      operationName: "EdiLadingVersions",
      variables: { projectId },
      signal: options?.signal,
    });
    return safeParse(
      ladingVersionSchema.array(),
      response.ediLadingVersions ?? [],
      "LadingVersionList",
    );
  }

  public async newDraft(projectId: string, fromVersionId: string) {
    const response = await requestGraphQL({
      document: CreateEdiLadingDraftDocument,
      operationName: "CreateEdiLadingDraft",
      variables: { projectId, fromVersionId },
    });
    return response.createEdiLadingDraft;
  }

  public async updateSettings(
    projectId: string,
    settings: Record<string, unknown>,
    version: number,
  ) {
    const response = await requestGraphQL({
      document: UpdateEdiLadingSettingsDocument,
      operationName: "UpdateEdiLadingSettings",
      variables: { projectId, settings, version },
    });
    return response.updateEdiLadingSettings;
  }

  public async getVersion(versionId: string, options?: Options) {
    const response = await requestGraphQL({
      document: EdiLadingVersionDocument,
      operationName: "EdiLadingVersion",
      variables: { id: versionId },
      signal: options?.signal,
    });
    return safeParse(ladingVersionDetailSchema, response.ediLadingVersion, "LadingVersionDetail");
  }

  public async saveDraft(versionId: string, request: SaveLadingDraftRequest) {
    const response = await api.put(`/edi/lading/versions/${versionId}/`, request);
    return safeParse(ladingVersionSchema, response, "LadingVersion");
  }

  /** Runs a map of a saved version on the server, its lookups answered from the mappings. */
  public async run(versionId: string, request: { map: string; input: LadingRunInput }) {
    const response = await requestGraphQL({
      document: RunEdiLadingVersionDocument,
      operationName: "RunEdiLadingVersion",
      variables: { versionId, mapName: request.map, input: request.input, rules: true },
    });
    return response.runEdiLadingVersion as LadingRunResult;
  }

  public async certify(versionId: string) {
    const response = await requestGraphQL({
      document: CertifyEdiLadingVersionDocument,
      operationName: "CertifyEdiLadingVersion",
      variables: { id: versionId },
    });
    return response.certifyEdiLadingVersion;
  }

  public async activate(versionId: string) {
    const response = await requestGraphQL({
      document: ActivateEdiLadingVersionDocument,
      operationName: "ActivateEdiLadingVersion",
      variables: { id: versionId },
    });
    return response.activateEdiLadingVersion;
  }

  public async archive(versionId: string) {
    const response = await requestGraphQL({
      document: ArchiveEdiLadingVersionDocument,
      operationName: "ArchiveEdiLadingVersion",
      variables: { id: versionId },
    });
    return response.archiveEdiLadingVersion;
  }

  public async startShadow(versionId: string, days: number) {
    const response = await requestGraphQL({
      document: StartEdiLadingShadowDocument,
      operationName: "StartEdiLadingShadow",
      variables: { id: versionId, days },
    });
    return response.startEdiLadingShadow;
  }

  public async stopShadow(versionId: string) {
    const response = await requestGraphQL({
      document: StopEdiLadingShadowDocument,
      operationName: "StopEdiLadingShadow",
      variables: { id: versionId },
    });
    return response.stopEdiLadingShadow;
  }

  public async shadowResults(versionId: string, onlyDiffering: boolean, options?: Options) {
    const response = await requestGraphQL({
      document: EdiLadingShadowResultsDocument,
      operationName: "EdiLadingShadowResults",
      variables: { versionId, onlyDiffering },
      signal: options?.signal,
    });
    return safeParse(
      ladingShadowReportSchema,
      response.ediLadingShadowResults,
      "LadingShadowReport",
    );
  }

  public async startReplay(versionId: string, request: StartLadingReplayRequest) {
    const response = await requestGraphQL({
      document: StartEdiLadingReplayDocument,
      operationName: "StartEdiLadingReplay",
      variables: { versionId, input: request },
    });
    return response.startEdiLadingReplay;
  }

  public async listReplays(versionId: string, options?: Options) {
    const response = await requestGraphQL({
      document: EdiLadingReplaysDocument,
      operationName: "EdiLadingReplays",
      variables: { versionId },
      signal: options?.signal,
    });
    return safeParse(
      ladingReplaySchema.array(),
      response.ediLadingReplays ?? [],
      "LadingReplayList",
    );
  }

  public async getFlowInstance(instanceId: string, options?: Options) {
    const response = await requestGraphQL({
      document: EdiFlowInstanceDetailDocument,
      operationName: "EdiFlowInstanceDetail",
      variables: { id: instanceId },
      signal: options?.signal,
    });
    return safeParse(
      flowInstanceDetailSchema,
      response.ediFlowInstanceDetail,
      "FlowInstanceDetail",
    );
  }

  public async releaseFlowInstance(instanceId: string) {
    const response = await requestGraphQL({
      document: ReleaseEdiFlowInstanceDocument,
      operationName: "ReleaseEdiFlowInstance",
      variables: { id: instanceId },
    });
    return response.releaseEdiFlowInstance;
  }
}
