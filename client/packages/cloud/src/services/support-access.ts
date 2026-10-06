import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  currentSessionSchema,
  grantStateSchema,
  grantedOrganizationsSchema,
  sessionViewSchema,
  staffProfileSchema,
  supportGrantSchema,
  type CreateGrantRequest,
  type CurrentSession,
  type ElevateRequest,
  type GrantState,
  type GrantedOrganization,
  type SessionView,
  type StaffProfile,
  type StartSessionRequest,
  type SupportGrant,
} from "../types/support-access";

export const supportAccessService = {
  async grantState(): Promise<GrantState> {
    const response = await api.get<unknown>("/support-access/");
    return safeParse(grantStateSchema, response, "Support access");
  },

  async createGrant(request: CreateGrantRequest): Promise<SupportGrant> {
    const response = await api.post<unknown>("/support-access/grant/", request);
    return safeParse(supportGrantSchema, response, "Support access");
  },

  async revokeGrant(): Promise<void> {
    await api.post("/support-access/grant/revoke/", {});
  },

  async staffProfile(): Promise<StaffProfile> {
    const response = await api.get<unknown>("/support/profile/");
    return safeParse(staffProfileSchema, response, "Staff profile");
  },

  async grantedOrganizations(): Promise<GrantedOrganization[]> {
    const response = await api.get<unknown>("/support/organizations/");
    const parsed = await safeParse(grantedOrganizationsSchema, response, "Organizations");
    return parsed.items;
  },

  async startSession(request: StartSessionRequest): Promise<SessionView> {
    const response = await api.post<unknown>("/support/sessions/", request);
    return safeParse(sessionViewSchema, response, "Support session");
  },

  async currentSession(): Promise<CurrentSession> {
    const response = await api.get<unknown>("/support/sessions/current/");
    return safeParse(currentSessionSchema, response, "Support session");
  },

  async elevate(request: ElevateRequest): Promise<SessionView> {
    const response = await api.post<unknown>("/support/sessions/current/elevate/", request);
    return safeParse(sessionViewSchema, response, "Support session");
  },

  async dropElevation(): Promise<SessionView> {
    const response = await api.post<unknown>("/support/sessions/current/drop-elevation/", {});
    return safeParse(sessionViewSchema, response, "Support session");
  },

  async endSession(): Promise<void> {
    await api.post("/support/sessions/current/end/", {});
  },
};
