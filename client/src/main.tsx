/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { ExclamationTriangleIcon } from "@radix-ui/react-icons";
import { QueryErrorResetBoundary } from "@tanstack/react-query";
import React from "react";
import ReactDOM from "react-dom/client";
import { ErrorBoundary } from "react-error-boundary";
import App from "./App";
import { Button } from "./components/ui/button";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <QueryErrorResetBoundary>
      {({ reset }) => (
        <ErrorBoundary
          onReset={reset}
          fallbackRender={({ resetErrorBoundary }) => (
            <div className="flex items-center justify-center h-screen bg-gray-100">
              <div className="text-center">
                <ExclamationTriangleIcon className="w-12 h-12 mx-auto text-red-500" />
                <h1 className="mt-4 text-2xl font-bold text-gray-800">
                  There was an error!
                </h1>
                <p className="text-gray-600">Please try again.</p>
                <Button
                  onClick={resetErrorBoundary}
                  className="mt-6 px-4 py-2 rounded-md shadow"
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
    </QueryErrorResetBoundary>
  </React.StrictMode>,
);
