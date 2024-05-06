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
      className="flex flex-col overflow-hidden rounded-lg border bg-card text-center text-card-foreground"
    >
      <div className="flex flex-1 flex-col p-8">
        <div className="flex flex-1 flex-col items-center justify-center">
          <h3 className="text-2xl font-semibold text-foreground">
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
              className="p-4 text-sm text-muted-foreground"
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
