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

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useFeatureFlags } from "@/hooks/useQueries";
import { FeatureFlag } from "@/types/organization";
import DOMPurify from "dompurify";

function FeatureFlagRow({ featureFlag }: { featureFlag: FeatureFlag }) {
  const sanitizedDescription = DOMPurify.sanitize(
    featureFlag.description || "No description",
  );

  return (
    <li
      key={featureFlag.code}
      className="flex items-center justify-between gap-x-6 py-5"
    >
      <div className="min-w-0">
        <div className="flex items-center gap-x-3">
          <p className="text-sm font-semibold leading-6 text-foreground">
            {featureFlag.name}
          </p>
          <Badge
            className="h-5 px-2.5 py-0.5"
            variant={featureFlag.enabled ? "default" : "destructive"}
          >
            {featureFlag.enabled ? "Enabled" : "Disabled"}
          </Badge>
        </div>
        <div
          className="mt-2 text-sm text-foreground-variant"
          dangerouslySetInnerHTML={{ __html: sanitizedDescription }}
        />
      </div>
      <div className="flex flex-none items-center gap-x-4">
        <Button
          size="sm"
          variant={featureFlag.enabled ? "destructive" : "default"}
        >
          {featureFlag.enabled ? "Disable" : "Enable"}
        </Button>
      </div>
    </li>
  );
}

export function FeatureManagementPage() {
  const { featureFlagsData } = useFeatureFlags();
  return (
    <>
      <ul role="list" className="divide-y divide-muted-foreground/20">
        {(featureFlagsData as FeatureFlag[]) &&
          (featureFlagsData as FeatureFlag[]).map((featureFlag) => (
            <FeatureFlagRow key={featureFlag.code} featureFlag={featureFlag} />
          ))}
      </ul>
    </>
  );
}
