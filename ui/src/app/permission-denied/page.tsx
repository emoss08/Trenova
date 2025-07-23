/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import logoRainbow from "@/assets/logo.webp";
import { MetaTags } from "@/components/meta-tags";
import { Button } from "@/components/ui/button";
import { LazyImage } from "@/components/ui/image";
import { formatResourceName } from "@/components/ui/permission-skeletons";
import { Resource } from "@/types/audit-entry";
import { useNavigate, useSearchParams } from "react-router";

export function PermissionDenied() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const resource = searchParams.get("resource") as Resource | null;

  return (
    <>
      <MetaTags
        title="Permission Denied"
        description="You don't have permission to access this page"
      />
      <div className="flex min-h-svh items-center justify-center">
        <div className="text-center max-w-md">
          <div className="mb-8">
            <LazyImage
              src={logoRainbow}
              alt="Trenova Logo"
              className="size-14 object-contain"
            />
          </div>
          <h1 className="mb-2 text-4xl font-bold">No Access to this page</h1>
          <div className="flex flex-col gap-2">
            <p className="text-lg text-muted-foreground">
              {resource ? (
                <>
                  You don&apos;t have permission to{" "}
                  <span className="font-bold underline decoration-blue-600 text-primary">
                    {formatResourceName(resource)}
                  </span>
                  .
                </>
              ) : (
                "You don&apos;t have permission to access this page."
              )}
            </p>
            <p className="mb-8 text-sm text-muted-foreground">
              Please contact your system administrator if you believe you should
              have access to this resource.
            </p>
          </div>
          <div className="flex justify-center gap-4">
            <Button>Request Access</Button>
            <Button onClick={() => navigate("/")} variant="outline">
              Back to dashboard
            </Button>
          </div>
        </div>
      </div>
    </>
  );
}
