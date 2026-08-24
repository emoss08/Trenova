import type { ReactNode } from "react";
import { hasCapability, isFieldRequired, type Capability } from "../lib/capability";
import type { ResolvedModeProfile } from "../types/shipment";
import { CapabilityExplainer } from "./capability-explainer";
import { FormControl, FormGroup } from "./ui/form";

/**
 * What a field needs to render itself against a resolved mode profile.
 *
 * `render` supplies the concrete input rather than the descriptor naming a
 * component type. The field components (NumberField, InputField, the
 * autocomplete fields) live in the app, and a shared package importing them
 * would invert the dependency — every app would then be forced to carry every
 * other app's fields.
 */
export type FieldDescriptor = {
  /** Form field path. Must be unique within a descriptor list. */
  name: string;
  /**
   * Field the profile's rules are matched against, when that differs from the
   * form path. A rule usually names the thing an operator thinks about rather
   * than the input: the dimension rule names `commodities`, and length, width
   * and height all answer to it.
   */
  ruleField?: string;
  /** Column span within the enclosing FormGroup. */
  cols?: 1 | 2 | 3 | 4 | "full";
  /**
   * Capability that must be present for this field to appear. Omit for fields
   * that are part of every profile.
   */
  capability?: Capability;
  /**
   * Renders the input. `required` is resolved from the profile so a descriptor
   * never has to restate a rule the profile already carries.
   */
  render: (props: { required: boolean }) => ReactNode;
  /**
   * Suppresses the "why does this field behave this way" popover. Default is to
   * show it wherever the profile has something to say about the field.
   */
  hideExplainer?: boolean;
};

/**
 * Decides whether a descriptor's field belongs on screen.
 *
 * A null profile means the profile has not resolved yet. Fields are shown in
 * that case rather than hidden: a form that renders empty and then sprouts
 * inputs is worse than one that briefly shows a field the profile turns out not
 * to want. This mirrors isCapabilitySectionVisible.
 */
export function isDescriptorVisible(
  descriptor: FieldDescriptor,
  profile: ResolvedModeProfile | null | undefined,
): boolean {
  if (!descriptor.capability) return true;
  if (!profile) return true;

  return hasCapability(profile, descriptor.capability);
}

export function visibleDescriptors(
  descriptors: FieldDescriptor[],
  profile: ResolvedModeProfile | null | undefined,
): FieldDescriptor[] {
  return descriptors.filter((descriptor) => isDescriptorVisible(descriptor, profile));
}

export function CapabilityFields({
  descriptors,
  profile,
  cols = 2,
}: {
  descriptors: FieldDescriptor[];
  profile: ResolvedModeProfile | null | undefined;
  cols?: 1 | 2 | 3 | 4;
}) {
  const visible = visibleDescriptors(descriptors, profile);

  if (visible.length === 0) return null;

  return (
    <FormGroup cols={cols}>
      {visible.map((descriptor) => {
        const ruleField = descriptor.ruleField ?? descriptor.name;

        return (
          <FormControl key={descriptor.name} cols={descriptor.cols}>
            {descriptor.render({ required: isFieldRequired(profile, ruleField) })}
            {!descriptor.hideExplainer && (
              <CapabilityExplainer profile={profile} field={ruleField} />
            )}
          </FormControl>
        );
      })}
    </FormGroup>
  );
}
