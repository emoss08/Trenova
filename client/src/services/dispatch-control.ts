import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  dispatchControlSchema,
  type DispatchControl,
} from "@/types/dispatch-control";

export class DispatchControlService {
  public async get() {
    const response = await api.get<DispatchControl>("/dispatch-controls/");
    return safeParse(dispatchControlSchema, response, "Dispatch Control");
  }

  public async update(data: DispatchControl) {
    const response = await api.put<DispatchControl>(
      "/dispatch-controls/",
      data,
    );
    return safeParse(dispatchControlSchema, response, "Dispatch Control");
  }
}
