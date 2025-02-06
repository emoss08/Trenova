import { http } from "@/lib/http-client";

export type TractorAssignment = {
  primaryWorkerId: string;
  secondaryWorkerId?: string;
};

export async function getTractorAssignments(tractorId?: string) {
  return http.get<TractorAssignment>(`/tractors/${tractorId}/assignment/`);
}
