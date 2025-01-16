import { http } from "@/lib/http-client";
import { SelectOptionResponse } from "@/types/server";
import { type UsState } from "@/types/us-state";

export async function getUsStates() {
  const response = await http.get<UsState[]>("/us-states");
  return response.data;
}

export async function getUsStateOptions() {
  const response = await http.get<SelectOptionResponse>(
    "/us-states/select-options",
  );
  return response.data;
}
