import { api, ApiRequestError } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import type {
  LadingContext,
  LadingRunInput,
  LadingRunResult,
  LadingTestReport,
} from "@trenova/shared/edi/lading";
import {
  flowInstanceDetailSchema,
  ladingProjectSchema,
  ladingReplayListSchema,
  ladingReplaySchema,
  ladingShadowReportSchema,
  ladingVersionDetailSchema,
  ladingVersionListSchema,
  ladingVersionSchema,
  type CreateLadingProjectRequest,
  type SaveLadingDraftRequest,
  type StartLadingReplayRequest,
} from "@trenova/shared/types/lading";

type Options = { signal?: AbortSignal };

/** The Lading studio's REST calls (docs/engineering/lading.md, "Versions, replay and shadow"). */
export class LadingService {
  /** The organisation and custom fields projects are checked against, as the server checks them. */
  public async getContext(options?: Options) {
    return api.get<LadingContext>("/edi/lading/context/", { signal: options?.signal });
  }

  public async createProject(request: CreateLadingProjectRequest) {
    const response = await api.post("/edi/lading/projects/", request);
    return safeParse(ladingProjectSchema, response, "LadingProject");
  }

  /** The partner's project, or null for a partner that has none yet. */
  public async getPartnerProject(partnerId: string, options?: Options) {
    try {
      const response = await api.get(`/edi/lading/projects/by-partner/${partnerId}/`, {
        signal: options?.signal,
      });
      return safeParse(ladingProjectSchema, response, "LadingProject");
    } catch (error) {
      if (error instanceof ApiRequestError && error.status === 404) return null;
      throw error;
    }
  }

  public async listVersions(projectId: string, options?: Options) {
    const response = await api.get(`/edi/lading/projects/${projectId}/versions/`, {
      signal: options?.signal,
    });
    return (await safeParse(ladingVersionListSchema, response, "LadingVersionList")).items;
  }

  public async newDraft(projectId: string, fromVersionId: string) {
    const response = await api.post(`/edi/lading/projects/${projectId}/drafts/`, {
      fromVersionId,
    });
    return safeParse(ladingVersionSchema, response, "LadingVersion");
  }

  public async updateSettings(
    projectId: string,
    settings: Record<string, unknown>,
    version: number,
  ) {
    const response = await api.put(`/edi/lading/projects/${projectId}/settings/`, {
      settings,
      version,
    });
    return safeParse(ladingProjectSchema, response, "LadingProject");
  }

  public async getVersion(versionId: string, options?: Options) {
    const response = await api.get(`/edi/lading/versions/${versionId}/`, {
      signal: options?.signal,
    });
    return safeParse(ladingVersionDetailSchema, response, "LadingVersionDetail");
  }

  public async saveDraft(versionId: string, request: SaveLadingDraftRequest) {
    const response = await api.put(`/edi/lading/versions/${versionId}/`, request);
    return safeParse(ladingVersionSchema, response, "LadingVersion");
  }

  /** Runs a map of a saved version on the server, its lookups answered from the mappings. */
  public async run(versionId: string, request: { map: string; input: LadingRunInput }) {
    return api.post<LadingRunResult>(`/edi/lading/versions/${versionId}/run/`, {
      ...request,
      rules: true,
    });
  }

  public async test(versionId: string) {
    return api.post<LadingTestReport>(`/edi/lading/versions/${versionId}/test/`);
  }

  public async certify(versionId: string) {
    const response = await api.post(`/edi/lading/versions/${versionId}/certify/`);
    return safeParse(ladingVersionSchema, response, "LadingVersion");
  }

  public async activate(versionId: string) {
    const response = await api.post(`/edi/lading/versions/${versionId}/activate/`);
    return safeParse(ladingVersionSchema, response, "LadingVersion");
  }

  public async archive(versionId: string) {
    const response = await api.post(`/edi/lading/versions/${versionId}/archive/`);
    return safeParse(ladingVersionSchema, response, "LadingVersion");
  }

  public async startShadow(versionId: string, days: number) {
    const response = await api.post(`/edi/lading/versions/${versionId}/shadow/`, { days });
    return safeParse(ladingVersionSchema, response, "LadingVersion");
  }

  public async stopShadow(versionId: string) {
    const response = await api.delete(`/edi/lading/versions/${versionId}/shadow/`);
    return safeParse(ladingVersionSchema, response, "LadingVersion");
  }

  public async shadowResults(versionId: string, onlyDiffering: boolean, options?: Options) {
    const query = onlyDiffering ? "?onlyDiffering=true" : "";
    const response = await api.get(`/edi/lading/versions/${versionId}/shadow-results/${query}`, {
      signal: options?.signal,
    });
    return safeParse(ladingShadowReportSchema, response, "LadingShadowReport");
  }

  public async startReplay(versionId: string, request: StartLadingReplayRequest) {
    const response = await api.post(`/edi/lading/versions/${versionId}/replays/`, request);
    return safeParse(ladingReplaySchema, response, "LadingReplay");
  }

  public async listReplays(versionId: string, options?: Options) {
    const response = await api.get(`/edi/lading/versions/${versionId}/replays/`, {
      signal: options?.signal,
    });
    return (await safeParse(ladingReplayListSchema, response, "LadingReplayList")).items;
  }

  public async getFlowInstance(instanceId: string, options?: Options) {
    const response = await api.get(`/edi/flows/instances/${instanceId}/`, {
      signal: options?.signal,
    });
    return safeParse(flowInstanceDetailSchema, response, "FlowInstanceDetail");
  }

  public async releaseFlowInstance(instanceId: string) {
    return api.post(`/edi/flows/instances/${instanceId}/release/`);
  }
}
