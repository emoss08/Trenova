import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import { dataEntryControlSchema, type DataEntryControl } from "@/types/data-entry-control";

export class DataEntryControlService {
  public async get() {
    const response = await api.get<DataEntryControl>("/data-entry-controls/");
    return safeParse(dataEntryControlSchema, response, "Data Entry Control");
  }

  public async update(data: DataEntryControl) {
    const response = await api.put<DataEntryControl>("/data-entry-controls/", data);
    return safeParse(dataEntryControlSchema, response, "Data Entry Control");
  }
}
