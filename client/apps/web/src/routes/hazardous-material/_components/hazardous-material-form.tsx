import { useT } from "@trenova/shared/i18n/use-t";
import { fetchOptions } from "@/components/fields/autocomplete/autocomplete-content";
import { FieldWrapper } from "@/components/fields/field-components";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { Input } from "@trenova/shared/components/ui/input";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { hazardousClassChoices, packingGroupChoices, statusChoices } from "@/lib/choices";
import type { DotHazmatReference } from "@/types/dot-hazmat-reference";
import type { HazardousClass, HazardousMaterial } from "@/types/hazardous-material";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useRef, useState } from "react";
import { Controller, useFormContext, useWatch } from "react-hook-form";

const dotClassToHazardClass: Record<string, HazardousClass> = {
  "1": "HazardClass1",
  "1.1": "HazardClass1And1",
  "1.2": "HazardClass1And2",
  "1.3": "HazardClass1And3",
  "1.4": "HazardClass1And4",
  "1.5": "HazardClass1And5",
  "1.6": "HazardClass1And6",
  "2.1": "HazardClass2And1",
  "2.2": "HazardClass2And2",
  "2.3": "HazardClass2And3",
  "3": "HazardClass3",
  "4.1": "HazardClass4And1",
  "4.2": "HazardClass4And2",
  "4.3": "HazardClass4And3",
  "5.1": "HazardClass5And1",
  "5.2": "HazardClass5And2",
  "6.1": "HazardClass6And1",
  "6.2": "HazardClass6And2",
  "7": "HazardClass7",
  "8": "HazardClass8",
  "9": "HazardClass9",
};

function mapDotClassToEnum(dotClass: string): HazardousClass | undefined {
  const trimmed = dotClass.trim();
  return dotClassToHazardClass[trimmed];
}

function DotHazmatNameField({ onSelect }: { onSelect: (option: DotHazmatReference) => void }) {
  const t = useT();

  const { control } = useFormContext<HazardousMaterial>();
  const [focused, setFocused] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const debouncedSearch = useDebounce(searchTerm, 300);
  const containerRef = useRef<HTMLDivElement>(null);

  const enabled = focused && debouncedSearch.length >= 2;

  const { data, isLoading } = useQuery({
    queryKey: ["dot-hazmat-search", debouncedSearch],
    queryFn: () =>
      fetchOptions<DotHazmatReference>(
        "/dot-hazmat-references/select-options/",
        debouncedSearch,
        1,
        10,
      ),
    enabled,
    staleTime: 2 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });

  const results = data?.results ?? [];
  const showDropdown = enabled && results.length > 0;

  const handleSelect = useCallback(
    (option: DotHazmatReference) => {
      setFocused(false);
      onSelect(option);
    },
    [onSelect],
  );

  const handleBlur = useCallback(() => {
    setTimeout(() => {
      if (!containerRef.current?.contains(document.activeElement)) {
        setFocused(false);
      }
    }, 150);
  }, []);

  return (
    <Controller<HazardousMaterial>
      name="name"
      control={control}
      rules={{ required: true }}
      render={({ field, fieldState }) => (
        <FieldWrapper
          label={t("Name")}
          required
          description={t("The name of the hazardous material. Type to search DOT references.")}
          error={fieldState.error?.message}
        >
          <div ref={containerRef} className="relative">
            <Input
              {...field}
              value={field.value as string}
              placeholder={t("Name")}
              maxLength={100}
              autoComplete="off"
              aria-invalid={fieldState.invalid}
              onFocus={() => {
                setFocused(true);
                setSearchTerm(field.value as string);
              }}
              onChange={(e) => {
                field.onChange(e);
                setSearchTerm(e.target.value);
              }}
              onBlur={(e) => {
                field.onBlur();
                handleBlur();
                void e;
              }}
            />
            {showDropdown && (
              <div className="bg-popover ring-foreground/10 absolute top-full left-0 z-50 mt-1 w-full overflow-hidden rounded-md shadow-md ring-1">
                <div className="max-h-[250px] overflow-y-auto p-1">
                  {results.map((option) => (
                    <button
                      key={option.id}
                      type="button"
                      className="hover:bg-accent flex w-full cursor-pointer flex-col items-start rounded-sm px-2 py-1.5 text-left text-sm"
                      onMouseDown={(e) => {
                        e.preventDefault();
                        handleSelect(option);
                      }}
                    >
                      <span>{t("UN{0} — {1}", option.unNumber, option.properShippingName)}</span>
                      <span className="text-2xs text-muted-foreground">
                        {t(
                          "Class {0}{1}",
                          option.hazardClass,
                          option.packingGroup ? ` ${t("| PG {0}", option.packingGroup)}` : "",
                        )}
                      </span>
                    </button>
                  ))}
                </div>
              </div>
            )}
            {enabled && isLoading && (
              <div className="bg-popover ring-foreground/10 absolute top-full left-0 z-50 mt-1 flex w-full justify-center rounded-md p-3 shadow-md ring-1">
                <Spinner className="size-4" />
              </div>
            )}
          </div>
        </FieldWrapper>
      )}
    />
  );
}

export function HazardousMaterialForm({ isEditing }: { isEditing?: boolean }) {
  const t = useT();

  const { control, setValue } = useFormContext<HazardousMaterial>();
  const isReportableQuantity = useWatch({
    control,
    name: "isReportableQuantity",
  });

  function handleDotReferenceChange(option: DotHazmatReference | null) {
    if (!option) return;

    setValue("name", option.properShippingName);
    setValue("properShippingName", option.properShippingName);
    const descParts = [`UN${option.unNumber} ${option.properShippingName}`];
    if (option.hazardClass) descParts.push(`Class ${option.hazardClass}`);
    if (option.subsidiaryHazard && option.subsidiaryHazard !== option.hazardClass)
      descParts.push(`Subsidiary hazard: ${option.subsidiaryHazard}`);
    if (option.packingGroup) descParts.push(`PG ${option.packingGroup}`);
    if (option.ergGuide) descParts.push(`ERG Guide ${option.ergGuide}`);
    setValue("description", descParts.join(", "));
    setValue("unNumber", option.unNumber);

    if (option.specialProvisions) {
      setValue("specialProvisions", option.specialProvisions);
    }

    if (option.subsidiaryHazard) {
      setValue("subsidiaryHazardClass", option.subsidiaryHazard);
    }

    if (option.ergGuide) {
      setValue("ergGuideNumber", option.ergGuide);
    }

    const mappedClass = mapDotClassToEnum(option.hazardClass);
    if (mappedClass) {
      setValue("class", mappedClass);
    }

    if (
      option.packingGroup === "I" ||
      option.packingGroup === "II" ||
      option.packingGroup === "III"
    ) {
      setValue("packingGroup", option.packingGroup);
    }

    setValue("placardRequired", !!option.hazardClass);
    setValue("isReportableQuantity", !!option.symbols && option.symbols.includes("RQ"));
    setValue("inhalationHazard", option.hazardClass === "2.3" || option.hazardClass === "6.1");
  }

  return (
    <div className="space-y-6">
      <FormSection
        title={t("General Information")}
        description={t("Basic identification for this hazardous material.")}
        className="border-b pb-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="status"
              label={t("Status")}
              placeholder={t("Status")}
              description={t("The current status of the hazardous material.")}
              options={statusChoices}
            />
          </FormControl>
          <FormControl>
            {isEditing ? (
              <DotHazmatNameField onSelect={handleDotReferenceChange} />
            ) : (
              <InputField
                control={control}
                rules={{ required: true }}
                name="name"
                label={t("Name")}
                placeholder={t("Name")}
                description={t("The name of the hazardous material.")}
                maxLength={100}
              />
            )}
          </FormControl>
          {!isEditing && (
            <FormControl cols="full">
              <InputField
                control={control}
                name="code"
                label={t("Code")}
                placeholder={t("Code")}
                description={t("The system-generated code for the hazardous material.")}
                readOnly
              />
            </FormControl>
          )}
          <FormControl cols="full">
            <TextareaField
              control={control}
              rules={{ required: true }}
              name="description"
              label={t("Description")}
              placeholder={t("Description")}
              description={t("A detailed description of the hazardous material.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title={t("DOT Classification")}
        description={t(
          "Hazard class, packing group, and regulatory identifiers per 49 CFR 172.101.",
        )}
        className="border-b pb-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="class"
              label={t("Class")}
              placeholder={t("Class")}
              description={t("The hazardous material classification.")}
              options={hazardousClassChoices}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="subsidiaryHazardClass"
              label={t("Subsidiary Hazard Class")}
              placeholder={t("e.g. 3, 8")}
              description={t("Secondary hazard classifications for this material.")}
              maxLength={20}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="packingGroup"
              label={t("Packing Group")}
              placeholder={t("Packing Group")}
              description={t("The packing group indicating the degree of danger.")}
              options={packingGroupChoices}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="unNumber"
              label={t("UN Number")}
              placeholder={t("UN Number")}
              description={t("The United Nations number identifying the hazardous substance.")}
              maxLength={4}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="ergGuideNumber"
              label={t("ERG Guide Number")}
              placeholder={t("e.g. 128")}
              description={t("Emergency Response Guidebook guide number for first responders.")}
              maxLength={10}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="labelCodes"
              label={t("Label Codes")}
              placeholder={t("e.g. 3, 8")}
              description={t("Required label codes for packages containing this material.")}
              maxLength={50}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="properShippingName"
              label={t("Proper Shipping Name")}
              placeholder={t("Proper Shipping Name")}
              description={t(
                "The proper shipping name as designated by transportation regulations.",
              )}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="specialProvisions"
              label={t("Special Provisions")}
              placeholder={t("Special Provisions")}
              description={t("Any special provisions or exceptions that apply to this material.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title={t("Compliance Flags")}
        description={t(
          "Regulatory indicators that affect placarding, reporting, and special handling.",
        )}
        className="border-b pb-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="placardRequired"
              label={t("Placard Required")}
              description={t("Whether a placard is required when transporting this material.")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="isReportableQuantity"
              label={t("Reportable Quantity")}
              description={t("Whether this material meets the reportable quantity threshold.")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="marinePollutant"
              label={t("Marine Pollutant")}
              description={t("Whether this material is classified as a marine pollutant.")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="inhalationHazard"
              label={t("Inhalation Hazard")}
              description={t("Whether this material poses a toxic or poison inhalation hazard.")}
            />
          </FormControl>
          {isReportableQuantity && (
            <FormControl cols="full">
              <InputField
                control={control}
                rules={{ required: isReportableQuantity }}
                name="quantityThreshold"
                label={t("Reportable Quantity Threshold")}
                placeholder={t("e.g. 100 lbs")}
                description={t("The quantity threshold that triggers reporting requirements.")}
                maxLength={20}
              />
            </FormControl>
          )}
        </FormGroup>
      </FormSection>
      <FormSection
        title={t("Handling & Emergency")}
        description={t(
          "Instructions and contact information for safe handling and emergency response.",
        )}
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="handlingInstructions"
              label={t("Handling Instructions")}
              placeholder={t("Handling Instructions")}
              description={t("Specific instructions for safely handling this material.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="emergencyContact"
              label={t("Emergency Contact")}
              placeholder={t("Emergency Contact")}
              description={t("The name or organization to contact in an emergency.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="emergencyContactPhoneNumber"
              label={t("Emergency Contact Phone")}
              placeholder={t("Emergency Contact Phone")}
              description={t("The phone number for the emergency contact.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
