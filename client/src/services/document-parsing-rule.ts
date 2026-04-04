import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  fixtureSchema,
  ruleSetSchema,
  ruleVersionSchema,
  simulationResultSchema,
  type Fixture,
  type RuleSet,
  type RuleVersion,
  type SimulationRequest,
  type SimulationResult,
} from "@/types/document-parsing-rule";

export class DocumentParsingRuleService {
  async list(documentKind?: string): Promise<RuleSet[]> {
    const params = documentKind ? `?documentKind=${documentKind}` : "";
    const response = await api.get<RuleSet[]>(
      `/document-parsing-rules/${params}`,
    );
    return safeParse(ruleSetSchema.array(), response, "Rule Sets");
  }

  async get(id: string): Promise<RuleSet> {
    const response = await api.get<RuleSet>(`/document-parsing-rules/${id}/`);
    return safeParse(ruleSetSchema, response, "Rule Set");
  }

  async create(data: Partial<RuleSet>): Promise<RuleSet> {
    const response = await api.post<RuleSet>("/document-parsing-rules/", data);
    return safeParse(ruleSetSchema, response, "Rule Set");
  }

  async update(id: string, data: RuleSet): Promise<RuleSet> {
    const response = await api.put<RuleSet>(
      `/document-parsing-rules/${id}/`,
      data,
    );
    return safeParse(ruleSetSchema, response, "Rule Set");
  }

  async delete(id: string): Promise<void> {
    await api.delete(`/document-parsing-rules/${id}/`);
  }

  async listVersions(ruleSetId: string): Promise<RuleVersion[]> {
    const response = await api.get<RuleVersion[]>(
      `/document-parsing-rules/${ruleSetId}/versions/`,
    );
    return safeParse(ruleVersionSchema.array(), response, "Rule Versions");
  }

  async createVersion(
    ruleSetId: string,
    data: Partial<RuleVersion>,
  ): Promise<RuleVersion> {
    const response = await api.post<RuleVersion>(
      `/document-parsing-rules/${ruleSetId}/versions/`,
      data,
    );
    return safeParse(ruleVersionSchema, response, "Rule Version");
  }

  async getVersion(versionId: string): Promise<RuleVersion> {
    const response = await api.get<RuleVersion>(
      `/document-parsing-rules/versions/${versionId}/`,
    );
    return safeParse(ruleVersionSchema, response, "Rule Version");
  }

  async updateVersion(
    versionId: string,
    data: RuleVersion,
  ): Promise<RuleVersion> {
    const response = await api.put<RuleVersion>(
      `/document-parsing-rules/versions/${versionId}/`,
      data,
    );
    return safeParse(ruleVersionSchema, response, "Rule Version");
  }

  async publishVersion(versionId: string): Promise<RuleVersion> {
    const response = await api.post<RuleVersion>(
      `/document-parsing-rules/versions/${versionId}/publish/`,
    );
    return safeParse(ruleVersionSchema, response, "Rule Version");
  }

  async simulate(
    versionId: string,
    data: SimulationRequest,
  ): Promise<SimulationResult> {
    const response = await api.post<SimulationResult>(
      `/document-parsing-rules/versions/${versionId}/simulate/`,
      data,
    );
    return safeParse(simulationResultSchema, response, "Simulation Result");
  }

  async listFixtures(ruleSetId: string): Promise<Fixture[]> {
    const response = await api.get<Fixture[]>(
      `/document-parsing-rules/${ruleSetId}/fixtures/`,
    );
    return safeParse(fixtureSchema.array(), response, "Fixtures");
  }

  async createFixture(
    ruleSetId: string,
    data: Partial<Fixture>,
  ): Promise<Fixture> {
    const response = await api.post<Fixture>(
      `/document-parsing-rules/${ruleSetId}/fixtures/`,
      data,
    );
    return safeParse(fixtureSchema, response, "Fixture");
  }

  async getFixture(fixtureId: string): Promise<Fixture> {
    const response = await api.get<Fixture>(
      `/document-parsing-rules/fixtures/${fixtureId}/`,
    );
    return safeParse(fixtureSchema, response, "Fixture");
  }

  async updateFixture(fixtureId: string, data: Fixture): Promise<Fixture> {
    const response = await api.put<Fixture>(
      `/document-parsing-rules/fixtures/${fixtureId}/`,
      data,
    );
    return safeParse(fixtureSchema, response, "Fixture");
  }

  async deleteFixture(fixtureId: string): Promise<void> {
    await api.delete(`/document-parsing-rules/fixtures/${fixtureId}/`);
  }
}
