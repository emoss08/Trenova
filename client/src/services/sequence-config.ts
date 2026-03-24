import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  sequenceConfigDocumentSchema,
  type SequenceConfigDocument,
} from "@/types/sequence-config";

export class SequenceConfigService {
  public async get() {
    const response = await api.get<SequenceConfigDocument>("/sequence-configs/");
    return safeParse(sequenceConfigDocumentSchema, response, "SequenceConfigDocument");
  }

  public async update(data: SequenceConfigDocument) {
    const response = await api.put<SequenceConfigDocument>("/sequence-configs/", data);
    return safeParse(sequenceConfigDocumentSchema, response, "SequenceConfigDocument");
  }
}
