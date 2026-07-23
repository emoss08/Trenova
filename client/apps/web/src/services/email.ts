import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  emailMessageListSchema,
  emailMessageSchema,
  emailProfileAssignmentSchema,
  emailProfileListSchema,
  emailProfileSchema,
  emailSuppressionListSchema,
  emailSuppressionSchema,
  testEmailProfileRequestSchema,
  type EmailProfile,
  type EmailProfileAssignment,
  type EmailSuppression,
  type TestEmailProfileRequest,
} from "@/types/email";

export class EmailService {
  public async listProfiles(params = "") {
    const response = await api.get(`/email-profiles/${params ? `?${params}` : ""}`);
    return safeParse(emailProfileListSchema, response, "Email Profiles");
  }

  public async createProfile(payload: EmailProfile) {
    const response = await api.post("/email-profiles/", emailProfileSchema.parse(payload));
    return safeParse(emailProfileSchema, response, "Email Profile");
  }

  public async updateProfile(id: string, payload: EmailProfile) {
    const response = await api.put(`/email-profiles/${id}/`, emailProfileSchema.parse(payload));
    return safeParse(emailProfileSchema, response, "Email Profile");
  }

  public async deleteProfile(id: string) {
    await api.delete(`/email-profiles/${id}/`);
  }

  public async testProfile(id: string, payload: TestEmailProfileRequest) {
    const response = await api.post(
      `/email-profiles/${id}/test-send/`,
      testEmailProfileRequestSchema.parse(payload),
    );
    return safeParse(emailMessageSchema, response, "Email Test Send");
  }

  public async listAssignments() {
    const response = await api.get("/email-profiles/assignments/");
    return safeParse(emailProfileAssignmentSchema.array(), response, "Email Profile Assignments");
  }

  public async updateAssignments(assignments: EmailProfileAssignment[]) {
    const response = await api.put("/email-profiles/assignments/", assignments);
    return safeParse(emailProfileAssignmentSchema.array(), response, "Email Profile Assignments");
  }

  public async listLogs(params = "") {
    const response = await api.get(`/email-logs/${params ? `?${params}` : ""}`);
    return safeParse(emailMessageListSchema, response, "Email Logs");
  }

  public async getLog(id: string) {
    const response = await api.get(`/email-logs/${id}/`);
    return safeParse(emailMessageSchema, response, "Email Log");
  }

  public async listSuppressions(params = "") {
    const response = await api.get(`/email-suppressions/${params ? `?${params}` : ""}`);
    return safeParse(emailSuppressionListSchema, response, "Email Suppressions");
  }

  public async createSuppression(payload: EmailSuppression) {
    const response = await api.post("/email-suppressions/", emailSuppressionSchema.parse(payload));
    return safeParse(emailSuppressionSchema, response, "Email Suppression");
  }

  public async deleteSuppression(id: string) {
    await api.delete(`/email-suppressions/${id}/`);
  }
}

