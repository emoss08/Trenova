import { http } from "@/lib/http-client";
import { SearchRequest, SearchResponse } from "@/types/search";

export class SearchAPI {
  async search(req: SearchRequest) {
    const response = await http.post<SearchResponse>("/search/", req);
    return response.data;
  }
}
