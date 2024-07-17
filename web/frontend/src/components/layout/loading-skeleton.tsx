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
            <u className="font-bold underline decoration-blue-600">Trenova</u>{" "}
            is gearing up for you.
          </p>
          <p className="text-muted-foreground mt-1 text-sm">
            We're working at lightning speed to get things ready. If this takes
            longer than a coffee break (10 seconds), please check your internet
            connection. <br />
            <u className="text-foreground decoration-blue-600">
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
