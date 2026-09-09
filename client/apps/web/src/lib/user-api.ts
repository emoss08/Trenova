import { api } from "@trenova/shared/lib/api";

// Trailing slash matters: the user routes are registered with one, and a POST that
// relies on Gin's redirect is a 307 the browser may or may not replay with its body.
export async function resetUserPassword(userId: string): Promise<{ message: string }> {
  return api.post<{ message: string }>(`/users/${userId}/reset-password/`, {});
}
