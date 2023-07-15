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

import { IChoiceProps } from "@/types";

/** Type for Hazardous Class Choices */
export type HazardousClassChoiceProps =
  | "1.1"
  | "1.2"
  | "1.3"
  | "1.4"
  | "1.5"
  | "1.6"
  | "2.1"
  | "2.2"
  | "2.3"
  | "3"
  | "4.1"
  | "4.2"
  | "4.3"
  | "5.1"
  | "5.2"
  | "6.1"
  | "6.2"
  | "7"
  | "8"
  | "9";

export const hazardousClassChoices: IChoiceProps<HazardousClassChoiceProps>[] =
  [
    { value: "1.1", label: "Division 1.1: Mass Explosive Hazard" },
    { value: "1.2", label: "Division 1.2: Projection Hazard" },
    {
      value: "1.3",
      label: "Division 1.3: Fire and/or Minor Blast/Minor Projection Hazard",
    },
    { value: "1.4", label: "Division 1.4: Minor Explosion Hazard" },
    {
      value: "1.5",
      label: "Division 1.5: Very Insensitive With Mass Explosion Hazard",
    },
    {
      value: "1.6",
      label: "Division 1.6: Extremely Insensitive; No Mass Explosion Hazard",
    },
    { value: "2.1", label: "Division 2.1: Flammable Gases" },
    { value: "2.2", label: "Division 2.2: Non-Flammable Gases" },
    { value: "2.3", label: "Division 2.3: Poisonous Gases" },
    { value: "3", label: "Division 3: Flammable Liquids" },
    { value: "4.1", label: "Division 4.1: Flammable Solids" },
    { value: "4.2", label: "Division 4.2: Spontaneously Combustible Solids" },
    { value: "4.3", label: "Division 4.3: Dangerous When Wet" },
    { value: "5.1", label: "Division 5.1: Oxidizing Substances" },
    { value: "5.2", label: "Division 5.2: Organic Peroxides" },
    { value: "6.1", label: "Division 6.1: Toxic Substances" },
    { value: "6.2", label: "Division 6.2: Infectious Substances" },
    { value: "7", label: "Division 7: Radioactive Material" },
    { value: "8", label: "Division 8: Corrosive Substances" },
    {
      value: "9",
      label: "Division 9: Miscellaneous Hazardous Substances and Articles",
    },
  ];

/** Type for Hazardous Class Choices */
export type PackingGroupChoiceProps = "I" | "II" | "III";

export const packingGroupChoices: IChoiceProps<PackingGroupChoiceProps>[] = [
  { value: "I", label: "I" },
  { value: "II", label: "II" },
  { value: "III", label: "III" },
];

/** Type for Unit of Measure Choices */
export type UnitOfMeasureChoiceProps =
  | "PALLET"
  | "TOTE"
  | "DRUM"
  | "CYLINDER"
  | "CASE"
  | "AMPULE"
  | "BAG"
  | "BOTTLE"
  | "PAIL"
  | "PIECES"
  | "ISO_TANK";

export const unitOfMeasureChoices: IChoiceProps<UnitOfMeasureChoiceProps>[] = [
  { value: "PALLET", label: "Pallet" },
  { value: "TOTE", label: "Tote" },
  { value: "DRUM", label: "Drum" },
  { value: "CYLINDER", label: "Cylinder" },
  { value: "CASE", label: "Case" },
  { value: "AMPULE", label: "Ampule" },
  { value: "BAG", label: "Bag" },
  { value: "BOTTLE", label: "Bottle" },
  { value: "PAIL", label: "Pail" },
  { value: "PIECES", label: "Pieces" },
  { value: "ISO_TANK", label: "ISO Tank" },
];
