import { MetaTags } from "@/components/meta-tags";
import { Button } from "@/components/ui/button";
import { formatResourceName } from "@/components/ui/permission-skeletons";
import { Resource } from "@/types/audit-entry";
import { useNavigate, useSearchParams } from "react-router";

export function PermissionDenied() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  
  const resource = searchParams.get("resource") as Resource | null;
  const action = searchParams.get("action") || "read";

  return (
    <>
      <MetaTags title="Permission Denied" description="You don't have permission to access this page" />
      <div className="flex min-h-[calc(100vh-12rem)] items-center justify-center">
        <div className="text-center">
          <div className="mb-8">
            <div className="mx-auto flex h-24 w-24 items-center justify-center rounded-full bg-destructive/10">
              <svg
                className="h-12 w-12 text-destructive"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                />
              </svg>
            </div>
          </div>
          
          <h1 className="mb-2 text-4xl font-bold">Permission Denied</h1>
          
          <p className="mb-6 text-lg text-muted-foreground">
            {resource ? (
              <>
                You don't have permission to {action}{" "}
                <span className="font-semibold">{formatResourceName(resource)}</span>.
              </>
            ) : (
              "You don't have permission to access this page."
            )}
          </p>
          
          <p className="mb-8 text-sm text-muted-foreground">
            Please contact your system administrator if you believe you should have access to this resource.
          </p>
          
          <div className="flex justify-center gap-4">
            <Button onClick={() => navigate(-1)} variant="outline">
              Go Back
            </Button>
            <Button onClick={() => navigate("/")}>
              Go to Dashboard
            </Button>
          </div>
        </div>
      </div>
    </>
  );
}