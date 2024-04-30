import { ExclamationTriangleIcon } from "@radix-ui/react-icons";
import { QueryErrorResetBoundary } from "@tanstack/react-query";
import ReactDOM from "react-dom/client";
import { ErrorBoundary } from "react-error-boundary";
import App from "./App";
import { Button } from "./components/ui/button";
import "./i18n";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <QueryErrorResetBoundary>
    {({ reset }) => (
      <ErrorBoundary
        onReset={reset}
        fallbackRender={({ resetErrorBoundary }) => (
          <div className="flex h-screen items-center justify-center bg-gray-100">
            <div className="text-center">
              <ExclamationTriangleIcon className="mx-auto size-12 text-red-500" />
              <h1 className="mt-4 text-2xl font-bold text-gray-800">
                There was an error!
              </h1>
              <p className="text-gray-600">Please try again.</p>
              <Button
                onClick={resetErrorBoundary}
                className="mt-6 rounded-md px-4 py-2 shadow"
              >
                Try again
              </Button>
            </div>
          </div>
        )}
      >
        <App />
      </ErrorBoundary>
    )}
  </QueryErrorResetBoundary>,
);
