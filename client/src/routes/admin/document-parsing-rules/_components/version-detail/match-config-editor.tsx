import { FormSection, FormGroup, FormControl } from "@/components/ui/form";
import { useFormContext } from "react-hook-form";
import { TagInput } from "../shared/tag-input";
import type { RuleVersionFormValues } from "@/types/document-parsing-rule";

export function MatchConfigEditor() {
  const { control } = useFormContext<RuleVersionFormValues>();

  return (
    <div className="space-y-4">
      <FormSection
        title="Provider Matching"
        description="Identify documents by their source provider or file name. These criteria are checked before the document content is analyzed."
      >
        <FormGroup cols={2}>
          <FormControl>
            <TagInput
              control={control}
              name="matchConfig.providerFingerprints"
              label="Provider Fingerprints"
              description="Unique identifiers for document providers (e.g. carrier SCAC codes). A document matches if its provider fingerprint appears in this list."
              placeholder="Add fingerprint..."
            />
          </FormControl>
          <FormControl>
            <TagInput
              control={control}
              name="matchConfig.fileNameContains"
              label="File Name Contains"
              description="Substrings to look for in the uploaded file name. Useful for providers that use consistent naming conventions."
              placeholder="Add pattern..."
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Content Matching"
        description="Match documents based on text found within the document body. These criteria help distinguish documents from the same provider that may have different formats."
      >
        <FormGroup cols={2}>
          <FormControl>
            <TagInput
              control={control}
              name="matchConfig.requiresAll"
              label="Requires All"
              description="Every term listed here must appear somewhere in the document text for the rule to match."
              placeholder="Add required term..."
            />
          </FormControl>
          <FormControl>
            <TagInput
              control={control}
              name="matchConfig.requiresAny"
              label="Requires Any"
              description="At least one of these terms must appear in the document text. Use this for documents that vary in wording."
              placeholder="Add term..."
            />
          </FormControl>
          <FormControl cols={2}>
            <TagInput
              control={control}
              name="matchConfig.sectionAnchors"
              label="Section Anchors"
              description="Headings or labels in the document that indicate this rule applies. The parser looks for these strings as section boundaries."
              placeholder="Add anchor..."
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
