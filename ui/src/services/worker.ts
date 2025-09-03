import { http } from "@/lib/http-client";
import { WorkerPTOSchema } from "@/lib/schemas/worker-schema";
import { LimitOffsetOptions, LimitOffsetResponse } from "@/types/server";
import { PTOStatus, PTOType } from "@/types/worker";

export type ListUpcomingPTORequest = {
  filter: LimitOffsetOptions;
  type?: PTOType;
  status?: PTOStatus;
  startDate?: number;
  endDate?: number;
};

export type PTOChartDataRequest = {
  startDate: number;
  endDate: number;
  type?: string;
};

export type PTOChartDataPoint = {
  date: string;
  vacation: number;
  sick: number;
  holiday: number;
  bereavement: number;
  maternity: number;
  paternity: number;
  workers: Record<
    string,
    Array<{
      id: string;
      firstName: string;
      lastName: string;
      ptoType: string;
    }>
  >;
};

export class WorkerAPI {
  async listUpcomingPTO(req: ListUpcomingPTORequest) {
    const response = await http.get<LimitOffsetResponse<WorkerPTOSchema>>(
      `/workers/upcoming-pto/`,
      {
        params: {
          type: req.type,
          status: req.status,
          startDate: req.startDate,
          endDate: req.endDate,
          ...req.filter,
        },
      },
    );
    return response.data;
  }

  async getPTOChartData(req: PTOChartDataRequest) {
    const response = await http.get<PTOChartDataPoint[]>(
      `/workers/pto-chart-data/`,
      {
        params: {
          startDate: req.startDate,
          endDate: req.endDate,
          type: req.type,
        },
      },
    );
    return response.data;
  }

  async approvePTO(ptoID: WorkerPTOSchema["id"]) {
    await http.post(`/workers/pto/${ptoID}/approve/`);
  }

  async rejectPTO(ptoID: WorkerPTOSchema["id"], reason: string) {
    await http.post(`/workers/pto/${ptoID}/reject/`, { reason });
  }
}
