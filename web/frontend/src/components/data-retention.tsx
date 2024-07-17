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

import { useTranslation } from "react-i18next";
import { Separator } from "./ui/separator";

export default function GoogleApi() {
  const { t } = useTranslation(["admin.generalpage", "common"]);

  return (
    <>
      <div className="space-y-3">
        <div>
          <h1 className="text-foreground text-2xl font-semibold">
            {t("title")}
          </h1>
          <p className="text-muted-foreground text-sm">{t("subTitle")}</p>
        </div>
        <Separator />
      </div>
      <ul role="list" className="divide-foreground divide-y">
        <li className="flex py-4">
          <div className="ml-3">
            <p className="text-foreground text-sm font-medium">TEST</p>
            <p className="text-muted-foreground text-sm">TEST</p>
          </div>
        </li>
      </ul>
    </>
  );
}
