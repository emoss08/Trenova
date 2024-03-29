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

import { useFeatureFlags } from "@/hooks/useQueries";
import { FeatureFlag } from "@/types/organization";
import { faCircleInfo } from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import DOMPurify from "dompurify";
import { Label } from "./common/fields/label";
import { Alert, AlertDescription, AlertTitle } from "./ui/alert";
import { Badge } from "./ui/badge";
import { ScrollArea } from "./ui/scroll-area";
import { Switch } from "./ui/switch";

function FeatureFlagRow({ featureFlag }: { featureFlag: FeatureFlag }) {
  const sanitizedDescription = DOMPurify.sanitize(
    featureFlag.description || "No description",
  );

  return (
    <li
      key={featureFlag.code}
      className="flex flex-col overflow-hidden rounded-lg border bg-card text-center text-card-foreground"
    >
      <div className="flex flex-1 flex-col p-8">
        <div className="flex flex-1 flex-col items-center justify-center">
          <h3 className="text-2xl font-semibold text-foreground">
            {featureFlag.name}
          </h3>
          <div className="mt-2 flex">
            {featureFlag.beta ? (
              <Badge className="mr-2" variant="info">
                {featureFlag.beta ? "Beta" : "Released"}
              </Badge>
            ) : (
              <Badge className="mr-2" variant="active">
                Released
              </Badge>
            )}
            {featureFlag.paidOnly && (
              <Badge className="mr-2" variant="warning">
                Paid only
              </Badge>
            )}
          </div>
        </div>
        <dl className="mt-1 grow">
          <ScrollArea className="mb-4 h-48">
            <dd
              dangerouslySetInnerHTML={{ __html: sanitizedDescription }}
              className="p-4 text-sm text-muted-foreground"
            ></dd>
          </ScrollArea>
        </dl>
      </div>
      <div className="flex items-center justify-between border-t px-4 py-2">
        <div className="flex items-center gap-x-2">
          <Switch defaultChecked={featureFlag.enabled} id="enable" />
          <Label htmlFor="enable">
            {featureFlag.enabled ? "Disable" : "Enable"} Feature
          </Label>
        </div>
        <div>
          <button className="text-sm text-primary hover:underline hover:decoration-blue-600">
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
    <>
      <Alert className="mb-5">
        <FontAwesomeIcon icon={faCircleInfo} className="size-4" />
        <AlertTitle>Information!</AlertTitle>
        <AlertDescription>
          All features marked{" "}
          <u className="font-bold underline decoration-blue-600">Paid Only</u>{" "}
          are only available to non-paid users during the beta phase. Once the
          beta phase is over, these features will be available to paid users
          only.
        </AlertDescription>
      </Alert>
      <ul
        role="list"
        className="mb-5 grid grid-cols-1 gap-6 lg:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4"
      >
        {(featureFlagsData as FeatureFlag[]) &&
          (featureFlagsData as FeatureFlag[]).map((featureFlag) => (
            <FeatureFlagRow key={featureFlag.code} featureFlag={featureFlag} />
          ))}
      </ul>
    </>
  );
}
