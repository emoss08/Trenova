import { http } from "@/lib/http-client";
import { Tractor } from "@/types/tractor";

export type TractorAssignment = {
  primaryWorkerId: string;
  secondaryWorkerId?: string;
};

export async function getTractorAssignments(tractorId?: Tractor["id"]) {
  return http.get<TractorAssignment>(`/tractors/${tractorId}/assignment/`);
}
