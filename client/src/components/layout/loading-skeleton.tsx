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

import { faSpinnerThird } from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";

export default function LoadingSkeleton() {
  return (
    <div className="flex min-h-screen flex-row items-center justify-center text-center">
      <div className="border-border bg-card flex w-[700px] flex-col rounded-md border sm:flex-row sm:items-center sm:justify-center">
        <div className="space-y-4 p-8">
          <FontAwesomeIcon
            icon={faSpinnerThird}
            size="3x"
            className="motion-safe:animate-spin"
          />
          <p className="font-xl mb-2 font-semibold">
            Hang tight!{" "}
            <u className="font-bold underline decoration-orange-600">Trenova</u>{" "}
            is gearing up for you.
          </p>
          <p className="text-muted-foreground mt-1 text-sm">
            We're working at lightning speed to get things ready. If this takes
            longer than a coffee break (10 seconds), please check your internet
            connection. <br />
            <u className="text-foreground decoration-orange-600">
              Still stuck?
            </u>{" "}
            Your friendly system administrator is just a call away for a swift
            rescue!
          </p>
        </div>
      </div>
    </div>
  );
}
