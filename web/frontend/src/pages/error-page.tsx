import { Button } from "@/components/ui/button";

import { useNavigate } from "react-router-dom";

function ErrorPage() {
  const navigate = useNavigate();

  return (
    <div className="bg-background flex h-full flex-col">
      <div className="flex grow items-center justify-center">
        <div className="m-auto flex flex-col items-center justify-center gap-2">
          <h1 className="text-[7rem] font-bold leading-tight">404</h1>
          <span className="font-medium">Oops! Page Not Found!</span>
          <p className="text-muted-foreground text-center">
            It seems like the page you're looking for <br />
            does not exist or might have been removed.
          </p>
          <div className="mt-6 flex gap-4">
            <Button variant="outline" onClick={() => navigate(-1)}>
              Go Back
            </Button>
            <Button onClick={() => navigate("/")}>Back to Home</Button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default ErrorPage;
