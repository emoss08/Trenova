import { api } from "@/lib/api";

export class StoredMileageService {
  public async delete(id: string) {
    await api.delete(`/stored-mileages/${id}/`);
  }
}
