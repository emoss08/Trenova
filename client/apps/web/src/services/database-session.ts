import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  listDatabaseSessionsResponseSchema,
  terminateDatabaseSessionResponseSchema,
  type ListDatabaseSessionsResponse,
  type TerminateDatabaseSessionResponse,
} from "@/types/database-session";

export class DatabaseSessionService {
  public async listBlocked(): Promise<ListDatabaseSessionsResponse> {
    const response = await api.get<ListDatabaseSessionsResponse>(
      "/admin/database-sessions/",
    );

    return safeParse(listDatabaseSessionsResponseSchema, response, "Database Session List");
  }

  public async terminate(
    pid: number,
  ): Promise<TerminateDatabaseSessionResponse> {
    const response = await api.post<TerminateDatabaseSessionResponse>(
      `/admin/database-sessions/${pid}/terminate/`,
    );

    return safeParse(terminateDatabaseSessionResponseSchema, response, "Terminate Database Session");
  }
}
