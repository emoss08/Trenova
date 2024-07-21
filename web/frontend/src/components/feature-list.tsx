/**
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

import { useFeatureFlags } from "@/hooks/useQueries";
import { OrganizationFeatureFlag } from "@/types/organization";
import DOMPurify from "dompurify";
import { Label } from "./common/fields/label";
import { Badge } from "./ui/badge";
import { ScrollArea } from "./ui/scroll-area";
import { Switch } from "./ui/switch";

function FeatureFlagRow({
  featureFlag,
}: {
  featureFlag: OrganizationFeatureFlag;
}) {
  const sanitizedDescription = DOMPurify.sanitize(
    featureFlag.edges.featureFlag.description,
  );

  const flag = featureFlag.edges.featureFlag;

  return (
    <li
      key={flag.code}
      className="bg-card text-card-foreground flex flex-col overflow-hidden rounded-lg border text-center"
    >
      <div className="flex flex-1 flex-col p-8">
        <div className="flex flex-1 flex-col items-center justify-center">
          <h3 className="text-foreground text-2xl font-semibold">
            {flag.name}
          </h3>
          <div className="mt-2 flex">
            {flag.beta ? (
              <Badge className="mr-2" variant="info">
                {flag.beta ? "Beta" : "Released"}
              </Badge>
            ) : (
              <Badge className="mr-2" variant="active">
                Released
              </Badge>
            )}
          </div>
        </div>
        <dl className="mt-1 grow">
          <ScrollArea className="mb-4 h-48">
            <dd
              dangerouslySetInnerHTML={{ __html: sanitizedDescription }}
              className="text-muted-foreground p-4 text-sm"
            ></dd>
          </ScrollArea>
        </dl>
      </div>
      <div className="flex items-center justify-between border-t px-4 py-2">
        <div className="flex items-center gap-x-2">
          <Switch defaultChecked={featureFlag.isEnabled} id="enable" />
          <Label htmlFor="enable">
            {featureFlag.isEnabled ? "Disable" : "Enable"} Feature
          </Label>
        </div>
        <div>
          <button className="text-primary text-sm hover:underline hover:decoration-blue-600">
            Send Feedback
          </button>
        </div>
      </div>
    </li>
  );
}

export default function FeatureList() {
  const { featureFlagsData } = useFeatureFlags();

  return (
    <ul
      role="list"
      className="mb-5 grid grid-cols-1 gap-6 lg:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4"
    >
      {(featureFlagsData as OrganizationFeatureFlag[]) &&
        (featureFlagsData as OrganizationFeatureFlag[]).map((featureFlag) => (
          <FeatureFlagRow
            key={featureFlag.edges.featureFlag.code}
            featureFlag={featureFlag}
          />
        ))}
    </ul>
  );
}
