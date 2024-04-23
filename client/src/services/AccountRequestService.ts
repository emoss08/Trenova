import axios from "@/lib/axiosConfig";
import { UserFavorite } from "@/types/accounts";

/**
 * Gets user favorites from the server
 * @returns An array of user favorites from the server
 */
export async function getUserFavorites(): Promise<UserFavorite> {
  const response = await axios.get("/user-favorites/");
  return response.data.results;
}

export type CheckEmailResponse = {
  exists: boolean;
  accountStatus: string;
  message: string;
};

export async function checkUserEmail(
  emailAddress: string,
): Promise<CheckEmailResponse> {
  const response = await axios.post("auth/check-email/", {
    emailAddress: emailAddress,
  });
  return response.data;
}
