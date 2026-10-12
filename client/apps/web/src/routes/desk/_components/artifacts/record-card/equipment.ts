import { choiceLabel, equipmentOwnershipChoices } from "@/lib/choices";
import type { EquipmentOwnershipType, EquipmentStatus } from "@trenova/graphql/generated/graphql";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { DueCheck } from "./card-due";
import type { CardFact, CardTone } from "./record-card";
import { cardDate, dueReading } from "./record-card-values";

/*
What a tractor's card and a trailer's card read the same way: whether the unit
can run, what it is called, its registration and lease, and the facts on file.
*/

/** Registration expiring inside this many days is worth a look. */
export const REGISTRATION_NOTICE_DAYS = 30;
/** A lease ending inside this many days is worth a look. */
export const LEASE_NOTICE_DAYS = 60;

/** What every piece of equipment carries that its card reads. */
export type EquipmentRecord = {
  code: string;
  status: EquipmentStatus;
  make: string;
  model: string;
  year?: number | null;
  vin: string;
  licensePlateNumber: string;
  registrationNumber: string;
  registrationExpiry?: number | null;
  externalId: string;
  ownershipType: EquipmentOwnershipType;
  lessorName: string;
  leaseReference: string;
  leaseEndDate?: number | null;
  createdAt: number;
  equipmentManufacturer?: { name: string } | null;
};

/** How a unit's status reads on its card: in service, held, or gone. */
export type EquipmentStanding = { label: string; tone: CardTone; mark: "dot" | "check" | "pause" };

export function equipmentStanding(status: EquipmentStatus, t: TranslateFn): EquipmentStanding {
  switch (status) {
    case "Available":
      return { label: t("Available"), tone: "success", mark: "dot" };
    case "AtMaintenance":
      return { label: t("At maintenance"), tone: "warn", mark: "pause" };
    case "OutOfService":
      return { label: t("Out of service"), tone: "warn", mark: "pause" };
    case "Sold":
      return { label: t("Sold"), tone: "muted", mark: "dot" };
  }
}

/** The unit by what it is, "2022 Freightliner Cascadia"; its code when nothing is on file. */
export function equipmentName(record: Pick<EquipmentRecord, "year" | "make" | "model" | "code">) {
  const name = [record.year ? String(record.year) : "", record.make, record.model]
    .map((part) => part.trim())
    .filter(Boolean)
    .join(" ");
  return name || record.code;
}

/** A unit held out of service or at maintenance; null for one that can run. */
export function equipmentStatusCheck(status: EquipmentStatus, t: TranslateFn): DueCheck | null {
  if (status === "OutOfService") {
    return {
      key: "status",
      state: "bad",
      title: t("Out of service"),
      detail: t("It cannot be dispatched until it is put back in service."),
    };
  }
  if (status === "AtMaintenance") {
    return {
      key: "status",
      state: "warn",
      title: t("At maintenance"),
      detail: t("It cannot be dispatched until maintenance releases it."),
    };
  }
  return null;
}

/** The unit's registration against now: none on file, expired, close, or good. */
export function registrationCheck(
  expiry: number | null | undefined,
  now: number,
  t: TranslateFn,
): DueCheck {
  if (expiry === null || expiry === undefined) {
    return {
      key: "registration",
      state: "warn",
      title: t("Registration"),
      detail: t("No expiry on file"),
    };
  }
  const due = dueReading(expiry, now, REGISTRATION_NOTICE_DAYS);
  return {
    key: "registration",
    state: due.state,
    title:
      due.state === "bad"
        ? t("Registration expired")
        : due.state === "warn"
          ? t("Registration expires in {0, plural, one {# day} other {# days}}", due.days)
          : t("Registration"),
    detail:
      due.state === "bad" ? t("Expired {0}", cardDate(expiry)) : t("Through {0}", cardDate(expiry)),
  };
}

/** The lease on a leased unit against now, naming the lessor; null for a unit not leased. */
export function leaseCheck(
  record: Pick<EquipmentRecord, "ownershipType" | "leaseEndDate" | "lessorName">,
  now: number,
  t: TranslateFn,
): DueCheck | null {
  if (record.ownershipType !== "Leased" || !record.leaseEndDate) return null;
  const end = record.leaseEndDate;
  const due = dueReading(end, now, LEASE_NOTICE_DAYS);
  const lessor = record.lessorName.trim();
  const ended = due.state === "bad";
  return {
    key: "lease",
    state: due.state,
    title: ended
      ? t("Lease ended")
      : due.state === "warn"
        ? t("Lease ends in {0, plural, one {# day} other {# days}}", due.days)
        : t("Lease"),
    detail: lessor
      ? ended
        ? t("Ended {0} · {1}", cardDate(end), lessor)
        : t("Through {0} · {1}", cardDate(end), lessor)
      : ended
        ? t("Ended {0}", cardDate(end))
        : t("Through {0}", cardDate(end)),
  };
}

/**
 * The facts on file for a unit, in the order a dispatcher reads them. `extra`
 * goes after the manufacturer: a tractor's fuel, a trailer's maximum load.
 */
export function equipmentFacts(
  record: EquipmentRecord,
  plateState: string | null | undefined,
  extra: readonly CardFact[],
  t: TranslateFn,
): CardFact[] {
  const plate = record.licensePlateNumber.trim();
  const ownership = choiceLabel(equipmentOwnershipChoices, record.ownershipType);
  const lessor = record.lessorName.trim();
  const lease = record.leaseReference.trim();
  return [
    {
      key: "plate",
      label: t("Plate"),
      value: plate ? [plate, plateState ?? ""].filter(Boolean).join(" · ") : t("None on file"),
    },
    {
      key: "registration",
      label: t("Registration number"),
      value: record.registrationNumber.trim() || t("None on file"),
    },
    { key: "vin", label: t("VIN"), value: record.vin.trim() || t("None on file") },
    {
      key: "manufacturer",
      label: t("Manufacturer"),
      value: record.equipmentManufacturer?.name ?? t("None on file"),
    },
    ...extra,
    {
      key: "ownership",
      label: t("Ownership"),
      value:
        record.ownershipType === "Leased" && (lessor || lease)
          ? [ownership, lessor, lease].filter(Boolean).join(" · ")
          : ownership,
    },
    {
      key: "external",
      label: t("External ID"),
      value: record.externalId.trim() || t("None"),
    },
    { key: "since", label: t("In the fleet since"), value: cardDate(record.createdAt) },
  ];
}
