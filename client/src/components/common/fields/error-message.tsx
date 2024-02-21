/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

import { faTriangleExclamation } from "@fortawesome/pro-regular-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";

export function ErrorMessage({ formError }: { formError?: string }) {
  return (
    <div className="mt-2 inline-block rounded bg-red-50 px-2 py-1 text-xs leading-tight text-red-500 dark:bg-red-300 dark:text-red-800 ">
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
