import { http } from "@/lib/http-client";
import type { DatabaseBackupListResponse } from "@/types/database-backup";

export class DatabaseBackupAPI {
  async get() {
    return http.get<DatabaseBackupListResponse>("/database-backups/");
  }

  async create() {
    return http.post("/database-backups/");
  }

  async download(fileName: string) {
    return http.get(`/database-backups/download/${fileName}/`);
  }

  async restore(fileName: string) {
    return http.post(`/database-backups/restore/`, { fileName });
  }

  async delete(fileName: string) {
    return http.delete(`/database-backups/${fileName}/`);
  }
}
