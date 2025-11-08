import { http } from "@/lib/http-client";
import type {
  UpdatePreferenceDataSchema,
  UserPreferenceSchema,
} from "@/lib/schemas/user-preference-schema";

export class UserPreferenceAPI {
  /**
   * Get current user's preferences
   * Auto-creates default preferences if none exist
   */
  async get() {
    const response = await http.get<UserPreferenceSchema>(
      "/users/me/preferences",
    );
    return response.data;
  }

  /**
   * Replace user preferences entirely (PUT)
   */
  async update(data: UserPreferenceSchema) {
    const response = await http.put<UserPreferenceSchema>(
      "/users/me/preferences",
      data,
    );
    return response.data;
  }

  /**
   * Merge preferences with existing (PATCH)
   * Only updates provided fields
   */
  async merge(data: UpdatePreferenceDataSchema) {
    const response = await http.patch<UserPreferenceSchema>(
      "/users/me/preferences",
      data,
    );
    return response.data;
  }
}
