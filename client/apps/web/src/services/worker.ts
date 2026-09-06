import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import { workerSchema, type Worker } from "@trenova/shared/types/worker";

export class WorkerService {
  public async patch(id: Worker["id"], data: Partial<Worker>) {
    const response = await api.patch<Worker>(`/workers/${id}/`, data);

    return safeParse(workerSchema, response, "Worker");
  }

  public async update(id: Worker["id"], data: Worker) {
    const response = await api.put<Worker>(`/workers/${id}/`, data);
    return safeParse(workerSchema, response, "Worker");
  }

  public async create(data: Worker) {
    const response = await api.post<Worker>("/workers/", data);
    return safeParse(workerSchema, response, "Worker");
  }
}
