import { usePermissionStore } from "@/stores/permission-store";
import { Operation } from "@/types/permission";
import { redirect, type LoaderFunction } from "react-router";

export function createPermissionLoader(
  resource: string,
  operation: number = Operation.Read,
): LoaderFunction {
  return async () => {
    const { manifest, hasPermission } = usePermissionStore.getState();

    if (!manifest) {
      return redirect("/login");
    }

    if (!hasPermission(resource, operation)) {
      throw new Response("Forbidden", { status: 403 });
    }

    return null;
  };
}

export function combineLoaders(...loaders: LoaderFunction[]): LoaderFunction {
  return async (args) => {
    for (const loader of loaders) {
      const result = await loader(args);
      if (result !== null) {
        return result;
      }
    }
    return null;
  };
}
