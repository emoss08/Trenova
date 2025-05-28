import { http } from "@/lib/http-client";
import { SelectOptionResponse } from "@/types/server";
import { type UsState } from "@/types/us-state";

export class UsStateAPI {
  // Get all US states
  async getUsStates() {
    const response = await http.get<UsState[]>("/us-states");
    return response.data;
  }
  // Get US state select options
  async getUsStateOptions() {
    const response = await http.get<SelectOptionResponse>(
      "/us-states/select-options",
    );
    return response.data;
  }
}
