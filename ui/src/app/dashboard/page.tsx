import { MetaTags } from "@/components/meta-tags";
import { Button } from "@/components/ui/button";
import { http } from "@/lib/http-client";
import { version } from "react";
import { toast } from "sonner";

export function Dashboard() {
  const refreshPermissionCache = () => {
    http.post("/permissions/refresh/").then(() => {
      toast.success("Permission cache refreshed");
    });
  };

  const clearPermissionCache = () => {
    http.post("/permissions/invalidate-cache/").then(() => {
      toast.success("Permission cache cleared");
    });
  };

  return (
    <>
      <MetaTags title="Dashboard" description="Dashboard" />
      <span>React Version: {`${version}`}</span>

      <h3>Permission Cache (Removed in production)</h3>
      <div className="flex flex-col gap-2">
        <Button className="w-[200px]" onClick={refreshPermissionCache}>
          Refresh Permission Cache
        </Button>
        <Button className="w-[200px]" onClick={clearPermissionCache}>
          Clear Permission Cache
        </Button>
      </div>
    </>
  );
}
