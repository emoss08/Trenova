/** Type for Hazardous Class Choices */
export enum HazardousClassChoiceProps {
  HazardClass1 = "HazardClass1",
  HazardClass1And1 = "HazardClass1And1",
  HazardClass1And2 = "HazardClass1And2",
  HazardClass1And3 = "HazardClass1And3",
  HazardClass1And4 = "HazardClass1And4",
  HazardClass1And5 = "HazardClass1And5",
  HazardClass1And6 = "HazardClass1And6",
  HazardClass2And1 = "HazardClass2And1",
  HazardClass2And2 = "HazardClass2And2",
  HazardClass2And3 = "HazardClass2And3",
  HazardClass3 = "HazardClass3",
  HazardClass4And1 = "HazardClass4And1",
  HazardClass4And2 = "HazardClass4And2",
  HazardClass4And3 = "HazardClass4And3",
  HazardClass5And1 = "HazardClass5And1",
  HazardClass5And2 = "HazardClass5And2",
  HazardClass6And1 = "HazardClass6And1",
  HazardClass6And2 = "HazardClass6And2",
  HazardClass7 = "HazardClass7",
  HazardClass8 = "HazardClass8",
  HazardClass9 = "HazardClass9",
}

/** Type for Packing Group Choices */
export enum PackingGroupChoiceProps {
  PackingGroupI = "I",
  PackingGroupII = "II",
  PackingGroupIII = "III",
}

export const mapToHazardousClassChoice = (
  hazardousClassChoice: HazardousClassChoiceProps,
) => {
  const hazardousClassChoiceLabels = {
    HazardClass1: "Division 1: Explosive Hazard",
    HazardClass1And1: "Division 1.1: Mass Explosive Hazard",
    HazardClass1And2: "Division 1.2: Projection Hazard",
    HazardClass1And3:
      "Division 1.3: Fire and/or Minor Blast/Minor Projection Hazard",
    HazardClass1And4: "Division 1.4: Minor Explosion Hazard",
    HazardClass1And5:
      "Division 1.5: Very Insensitive With Mass Explosion Hazard",
    HazardClass1And6:
      "Division 1.6: Extremely Insensitive; No Mass Explosion Hazard",
    HazardClass2And1: "Division 2.1: Flammable Gases",
    HazardClass2And2: "Division 2.2: Non-Flammable Gases",
    HazardClass2And3: "Division 2.3: Poisonous Gases",
    HazardClass3: "Division 3: Flammable Liquids",
    HazardClass4And1: "Division 4.1: Flammable Solids",
    HazardClass4And2: "Division 4.2: Spontaneously Combustible Solids",
    HazardClass4And3: "Division 4.3: Dangerous When Wet",
    HazardClass5And1: "Division 5.1: Oxidizing Substances",
    HazardClass5And2: "Division 5.2: Organic Peroxides",
    HazardClass6And1: "Division 6.1: Toxic Inhalation Hazard",
    HazardClass6And2: "Division 6.2: Toxic Ingestion Hazard",
    HazardClass7: "Division 7: Radioactive Materials",
    HazardClass8: "Division 8: Corrosive Materials",
    HazardClass9: "Division 9: Miscellaneous Hazardous Materials",
  };

  return hazardousClassChoiceLabels[hazardousClassChoice];
};
