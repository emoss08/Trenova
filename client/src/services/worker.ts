import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import { API_BASE_URL } from "@/lib/constants";
import {
  workerSchema,
  type ListUpcomingPTORequest,
  type PTOChartDataRequest,
  type Worker,
  type WorkerPTO,
} from "@/types/worker";

export class WorkerService {
  public async approvePTO(id: WorkerPTO["id"]) {
    await api.post<WorkerPTO>(`/workers/pto/${id}/approve/`);
  }

  public async rejectPTO(id: WorkerPTO["id"], reason: string) {
    await api.post<WorkerPTO>(`/workers/pto/${id}/reject/`, { reason });
  }

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

  public async getPTOChartData(req: PTOChartDataRequest) {
    const fetchURL = new URL(`${API_BASE_URL}/worker-pto/chart/`, window.location.origin);
    fetchURL.searchParams.set("startDateFrom", req.startDateFrom.toString());
    fetchURL.searchParams.set("startDateTo", req.startDateTo.toString());
    fetchURL.searchParams.set("type", req.type ?? "");
    fetchURL.searchParams.set("timezone", req.timezone ?? "");
    fetchURL.searchParams.set("workerId", req.workerId ?? "");

    const response = await fetch(fetchURL.href, {
      credentials: "include",
    });
    if (!response.ok) {
      throw new Error("Failed to fetch PTO chart data");
    }

    return response.json();
  }

  public async listUpcomingPTO(req: ListUpcomingPTORequest) {
    const fetchURL = new URL(`${API_BASE_URL}/worker-pto/upcoming/`, window.location.origin);

    fetchURL.searchParams.set("type", req.type ?? "");
    fetchURL.searchParams.set("status", req.status ?? "");
    fetchURL.searchParams.set("startDate", req.startDate?.toString() ?? "");
    fetchURL.searchParams.set("endDate", req.endDate?.toString() ?? "");
    fetchURL.searchParams.set("workerId", req.workerId ?? "");
    fetchURL.searchParams.set("timezone", req.timezone ?? "");

    const response = await fetch(fetchURL.href, {
      credentials: "include",
    });
    if (!response.ok) {
      throw new Error("Failed to fetch upcoming PTO");
    }

    return response.json();
  }

  public async getOption(id: Worker["id"]) {
    const response = await api.get<Worker>(`/workers/select-options/${id}`);

    return safeParse(workerSchema, response, "Worker");
  }
}
