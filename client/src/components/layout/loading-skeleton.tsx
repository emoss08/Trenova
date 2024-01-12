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

export default function LoadingSkeleton() {
  return (
    <div className="flex min-h-screen flex-row items-center justify-center">
      <div className="border">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-center">
          <div className="p-8">
            <p className="text-lg mb-2 font-semibold">
              Trenova is loading. Please wait.
            </p>
            <p className="mt-1 text-sm text-gray-400">
              If the operation exceeds a duration of 10 seconds, kindly verify
              the status of your internet connectivity. <br />
              In case of persistent difficulty, please get in touch with your
              designated system administrator.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
