import { getCurrentUser, validateSession } from "@/services/auth";
import { useAuthStore } from "@/stores/user-store";
import { LoaderFunctionArgs, redirect } from "react-router";

export async function checkAuthStatus() {
  try {
    const { data: sessionData } = await validateSession();

    if (!sessionData.valid) {
      return null;
    }

    const { data: userData } = await getCurrentUser();
    return userData;
  } catch {
    return null;
  }
}

export async function authLoader() {
  const { isInitialized } = useAuthStore.getState();

  if (!isInitialized) {
    const user = await checkAuthStatus();
    if (user) {
      useAuthStore.getState().setUser(user);
    }
    useAuthStore.getState().setInitialized(true);
  }

  const { user } = useAuthStore.getState();

  if (user) {
    return redirect("/");
  }

  return null;
}

export async function protectedLoader({ request }: LoaderFunctionArgs) {
  const { isInitialized } = useAuthStore.getState();

  if (!isInitialized) {
    const user = await checkAuthStatus();
    if (user) {
      useAuthStore.getState().setUser(user);
    }
    useAuthStore.getState().setInitialized(true);
  }

  const { user, isAuthenticated } = useAuthStore.getState();

  if (!user || !isAuthenticated) {
    const params = new URLSearchParams();
    params.set("from", new URL(request.url).pathname);
    return redirect("/auth?" + params.toString());
  }

  return null;
}
