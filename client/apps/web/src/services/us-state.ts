import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  usStateSchema,
  usStateSelectOptionResponseSchema,
  type UsState,
  type UsStateSelectOptionResponse,
} from "@/types/us-state";

export class UsStateService {
  public async options() {
    const response =
      await api.get<UsStateSelectOptionResponse>("/us-states/options");

    return safeParse(usStateSelectOptionResponseSchema, response, "US State Options");
  }

  public async getOption(id: UsState["id"]) {
    const response = await api.get<UsState>(`/us-states/${id}`);

    return safeParse(usStateSchema, response, "US State");
  }
}
