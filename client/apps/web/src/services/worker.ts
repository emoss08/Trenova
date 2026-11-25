import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import { workerWithoutPersonalDetailsSchema, type Worker } from "@trenova/shared/types/worker";
import { workerPhotoChangeSchema } from "@trenova/shared/types/person-photo";

export class WorkerService {
  public async patch(id: Worker["id"], data: Partial<Worker>) {
    const response = await api.patch<Worker>(`/workers/${id}/`, data);

    return safeParse(workerWithoutPersonalDetailsSchema, response, "Worker");
  }

  public async update(id: Worker["id"], data: Worker) {
    const response = await api.put<Worker>(`/workers/${id}/`, data);
    return safeParse(workerWithoutPersonalDetailsSchema, response, "Worker");
  }

  /** Replaces the worker's photo; `version` is the worker's, refused when another save moved it. */
  public async uploadPhoto(id: string, file: File, version: number) {
    const formData = new FormData();
    formData.append("file", file);
    formData.append("version", String(version));
    const response = await api.upload<unknown>(`/workers/${id}/photo/`, formData);
    return safeParse(workerPhotoChangeSchema, response, "Worker photo");
  }

  public async removePhoto(id: string, version: number) {
    const response = await api.delete<unknown>(
      `/workers/${id}/photo/?version=${encodeURIComponent(String(version))}`,
    );
    return safeParse(workerPhotoChangeSchema, response, "Worker photo");
  }

  public async create(data: Worker) {
    const response = await api.post<Worker>("/workers/", data);
    return safeParse(workerWithoutPersonalDetailsSchema, response, "Worker");
  }
}

export const workerService = new WorkerService();
