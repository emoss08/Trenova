import { http } from "@/lib/http-client";
import { TestConnectionSchema } from "@/lib/schemas/email-profile-schema";

type TestConnectionResponse = {
  success: boolean;
};

export class EmailProfileAPI {
  async testConnection(data: TestConnectionSchema) {
    return http.post<TestConnectionResponse>(
      `/email-profiles/test-connection/`,
      {
        ...data,
      },
    );
  }
}
