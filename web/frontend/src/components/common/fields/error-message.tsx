/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { faTriangleExclamation } from "@fortawesome/pro-regular-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";

export function ErrorMessage({ formError }: { formError?: string }) {
  return (
    <div className="mt-2 inline-block rounded bg-red-50 px-2 py-1 text-xs leading-tight text-red-500 dark:bg-red-300 dark:text-red-800">
      {formError ? formError : "An Error has occurred. Please try again."}
    </div>
  );
}

export function FieldErrorMessage({ formError }: { formError?: string }) {
  return (
    <>
      <div className="pointer-events-none absolute inset-y-0 right-0 mr-2.5 mt-1.5">
        <FontAwesomeIcon
          icon={faTriangleExclamation}
          className="text-red-500"
        />
      </div>
      <ErrorMessage formError={formError} />
    </>
  );
}
