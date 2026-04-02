import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  documentControlSchema,
  type DocumentControl,
} from "@/types/document-control";

export class DocumentControlService {
  public async get() {
    const response = await api.get<DocumentControl>("/document-controls/");

    return safeParse(documentControlSchema, response, "Document Control");
  }

  public async update(data: DocumentControl) {
    const response = await api.put<DocumentControl>(
      "/document-controls/",
      data,
    );

    return safeParse(documentControlSchema, response, "Document Control");
  }
}
