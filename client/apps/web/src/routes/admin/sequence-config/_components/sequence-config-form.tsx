import { SettingsForm } from "@/components/settings/settings-form";
import { SettingsSections } from "@/components/settings/settings-section";
import { sequenceConfig } from "@/lib/queries/sequence-config";
import { sequenceConfigService } from "@/services/sequence-config";
import { sequenceConfigDocumentSchema, type SequenceConfigDocument } from "@/types/sequence-config";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useState } from "react";
import { prepareSequenceConfigs } from "../sequence-config-submit";
import {
  indexBySequence,
  sequenceConfigTabs,
  TAB_SEQUENCES,
  type SequenceConfigTab,
} from "../sequence-config-tabs";
import { LocationCodeSection } from "./location-code-sections";
import { SequenceSection } from "./sequence-form-sections";

/**
 * The numbering of every record Trenova issues a number to: one document, saved once, with
 * a tab for each part of the business and a section for each sequence in it.
 */
export default function SequenceConfigForm() {
  const { data } = useSuspenseQuery({ ...sequenceConfig.get() });
  const positions = indexBySequence(data.configs);
  // One moment for every sample on the page, so they agree with one another.
  const [now] = useState(() => new Date());

  return (
    <SettingsForm<SequenceConfigDocument, SequenceConfigTab>
      values={data}
      schema={sequenceConfigDocumentSchema}
      queryKey={sequenceConfig.get._def}
      save={(values) => sequenceConfigService.update(values)}
      resourceName="Sequence Configuration"
      prepare={prepareSequenceConfigs}
      tabs={sequenceConfigTabs(data.configs)}
      content={(tab) => (
        <SettingsSections>
          {TAB_SEQUENCES[tab].map((type) => {
            const index = positions[type];
            if (index === undefined) return null;
            return type === "location_code" ? (
              <LocationCodeSection key={type} index={index} now={now} />
            ) : (
              <SequenceSection key={type} index={index} type={type} now={now} />
            );
          })}
        </SettingsSections>
      )}
    />
  );
}
