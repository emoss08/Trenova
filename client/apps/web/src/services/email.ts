import {
  CreateEmailProfileDocument,
  DeleteEmailProfileDocument,
  EmailLogsDocument,
  EmailProfileAssignmentsDocument,
  SendEmailProfileTestDocument,
  UpdateEmailProfileAssignmentsDocument,
  UpdateEmailProfileDocument,
  type EmailProfileInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  emailMessageSchema,
  emailProfileAssignmentSchema,
  emailProfileSchema,
  testEmailProfileRequestSchema,
  type EmailProfile,
  type EmailProfileAssignment,
  type TestEmailProfileRequest,
} from "@trenova/shared/types/email";

type Options = { signal?: AbortSignal };

/** How many sends the email log lists, newest first. */
const EMAIL_LOG_PAGE_SIZE = 20;

function toEmailProfileInput(payload: EmailProfile): EmailProfileInput {
  const profile = emailProfileSchema.parse(payload);
  return {
    name: profile.name,
    description: profile.description ?? undefined,
    senderName: profile.senderName,
    senderEmail: profile.senderEmail,
    replyToEmail: profile.replyToEmail || undefined,
    provider: profile.provider,
    status: profile.status,
  };
}

export class EmailService {
  public async createProfile(payload: EmailProfile) {
    const response = await requestGraphQL({
      document: CreateEmailProfileDocument,
      operationName: "CreateEmailProfile",
      variables: { input: toEmailProfileInput(payload) },
    });
    return response.createEmailProfile;
  }

  public async updateProfile(id: string, payload: EmailProfile) {
    const response = await requestGraphQL({
      document: UpdateEmailProfileDocument,
      operationName: "UpdateEmailProfile",
      variables: { id, input: toEmailProfileInput(payload) },
    });
    return response.updateEmailProfile;
  }

  public async deleteProfile(id: string) {
    await requestGraphQL({
      document: DeleteEmailProfileDocument,
      operationName: "DeleteEmailProfile",
      variables: { id },
    });
  }

  public async testProfile(id: string, payload: TestEmailProfileRequest) {
    const response = await requestGraphQL({
      document: SendEmailProfileTestDocument,
      operationName: "SendEmailProfileTest",
      variables: { id, input: testEmailProfileRequestSchema.parse(payload) },
    });
    return response.sendEmailProfileTest;
  }

  public async listAssignments(options?: Options) {
    const response = await requestGraphQL({
      document: EmailProfileAssignmentsDocument,
      operationName: "EmailProfileAssignments",
      signal: options?.signal,
    });
    return safeParse(
      emailProfileAssignmentSchema.array(),
      response.emailProfileAssignments,
      "Email Profile Assignments",
    );
  }

  public async updateAssignments(assignments: EmailProfileAssignment[]) {
    const response = await requestGraphQL({
      document: UpdateEmailProfileAssignmentsDocument,
      operationName: "UpdateEmailProfileAssignments",
      variables: {
        input: assignments.map(({ purpose, profileId }) => ({ purpose, profileId })),
      },
    });
    return response.updateEmailProfileAssignments;
  }

  public async listLogs(options?: Options) {
    const response = await requestGraphQL({
      document: EmailLogsDocument,
      operationName: "EmailLogs",
      variables: { input: { first: EMAIL_LOG_PAGE_SIZE } },
      signal: options?.signal,
    });
    return safeParse(
      emailMessageSchema.array(),
      response.emailLogs.edges.map((edge) => edge.node),
      "Email Logs",
    );
  }
}
