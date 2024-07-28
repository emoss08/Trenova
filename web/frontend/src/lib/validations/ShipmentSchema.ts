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

import {useShipmentControl} from "@/hooks/useQueries";
import {
    CodeTypeProps,
    HazardousClassChoiceProps,
    SegregationTypeChoiceProps,
    ShipmentEntryMethodChoices,
    ShipmentStatusChoiceProps,
} from "@/lib/choices";
import {type StatusChoiceProps} from "@/types";
import type {User} from "@/types/accounts";
import type {
    HazardousMaterialSegregationRuleFormValues,
    ReasonCodeFormValues,
    ServiceTypeFormValues,
    ShipmentControlFormValues,
    ShipmentFormValues,
    ShipmentTypeFormValues,
} from "@/types/shipment";
import {yupResolver} from "@hookform/resolvers/yup";
import {useForm} from "react-hook-form";
import {array, boolean, number, object, ObjectSchema, string} from "yup";
import {stopSchema} from "./StopSchema";

export const shipmentControlSchema: ObjectSchema<ShipmentControlFormValues> =
    object().shape({
        enforceRevCode: boolean().required("Enforce Rev Code is required"),
        enforceVoidedComm: boolean().required("Enforce Voided Comm is required"),
        autoTotalShipment: boolean().required("Auto Shipment Total is required"),
        compareOriginDestination: boolean().required(
            "Compare Origin Destination is required",
        ),
        checkForDuplicateBol: boolean().required(
            "Check for Duplicate BOL is required",
        ),
    });

export const serviceTypeSchema: ObjectSchema<ServiceTypeFormValues> =
    object().shape({
        status: string<StatusChoiceProps>().required("Status is required"),
        code: string()
            .max(10, "Code must be at most 10 characters")
            .required("Code is required"),
        description: string().optional(),
    });

export const reasonCodeSchema: ObjectSchema<ReasonCodeFormValues> =
    object().shape({
        status: string<StatusChoiceProps>().required("Status is required"),
        code: string()
            .max(10, "Code must be at most 10 characters")
            .required("Code is required"),
        codeType: string<CodeTypeProps>().required("Code type is required"),
        description: string().required("Description is required"),
    });

export function useHazmatSegRulesForm(
    hazmatSegRule?: HazardousMaterialSegregationRuleFormValues,
) {
    const hazmatSegRulesSchema: ObjectSchema<HazardousMaterialSegregationRuleFormValues> =
        object().shape({
            classA: string<HazardousClassChoiceProps>().required(
                "Class A is required",
            ),
            classB: string<HazardousClassChoiceProps>().required(
                "Class B is required",
            ),
            segregationType: string<SegregationTypeChoiceProps>().required(
                "Segregation Type is required",
            ),
        });

    const hazmatSegRulesForm =
        useForm<HazardousMaterialSegregationRuleFormValues>({
            resolver: yupResolver(hazmatSegRulesSchema),
            defaultValues: hazmatSegRule,
        });

    return {hazmatSegRulesForm};
}

export const hazmatSegRulesSchema: ObjectSchema<HazardousMaterialSegregationRuleFormValues> =
    object().shape({
        classA: string<HazardousClassChoiceProps>().required("Class A is required"),
        classB: string<HazardousClassChoiceProps>().required("Class B is required"),
        segregationType: string<SegregationTypeChoiceProps>().required(
            "Segregation Type is required",
        ),
    });

export const shipmentTypeSchema: ObjectSchema<ShipmentTypeFormValues> =
    object().shape({
        status: string<StatusChoiceProps>().required("Status is required"),
        code: string()
            .max(10, "Name must be at most 100 characters")
            .required("Code is required"),
        description: string().optional(),
        color: string().optional(),
    });

export function useShipmentForm({user}: { user: User }) {
    const {data: shipmentControlData, isLoading: isShipmentControlLoading} =
        useShipmentControl();

    // Shipment Form validation schema
    const shipmentSchema: ObjectSchema<ShipmentFormValues> = object().shape({
        proNumber: string().required("Pro number is required."),
        shipmentType: string().required("Shipment type is required."),
        serviceType: string().required("Service type is required."),
        status: string<ShipmentStatusChoiceProps>().required("Status is required."),
        revenueCode:
            shipmentControlData && shipmentControlData.enforceRevCode
                ? string().required("Revenue code is required.")
                : string(),
        originLocation: string()
            .test({
                name: "originLocation",
                test: function (value) {
                    if (!value) {
                        return this.parent.originAddress !== "";
                    }
                    return true;
                },
                message: "Origin location is required.",
            })
            .test({
                name: "originLocation",
                test: function (value) {
                    return !(
                        shipmentControlData &&
                        shipmentControlData.enforceOriginDestination &&
                        value === this.parent.destinationLocation
                    );
                },
                message: "Origin and Destination locations cannot be the same.",
            }),
        originAddress: string().test({
            name: "originAddress",
            test: function (value) {
                if (!value) {
                    return false;
                }
                return true;
            },
            message: "Origin address is required.",
        }),
        originAppointmentWindowStart: string().required(
            "Origin appointment window start is required.",
        ),
        originAppointmentWindowEnd: string().required(
            "Origin appointment window end is required.",
        ),
        destinationLocation: string()
            .test({
                name: "destinationLocation",
                test: function (value) {
                    if (!value) {
                        return this.parent.destinationAddress !== "";
                    }
                    return true;
                },
                message: "Destination location is required.",
            })
            .test({
                name: "destinationLocation",
                test: function (value) {
                    return !(
                        shipmentControlData &&
                        shipmentControlData.enforceOriginDestination &&
                        value === this.parent.originLocation
                    );
                },
                message: "Origin and Destination locations cannot be the same.",
            }),
        destinationAddress: string().test({
            name: "destinationAddress",
            test: function (value) {
                if (!value) {
                    return this.parent.destinationLocation !== "";
                }
                return true;
            },
            message: "Destination address is required.",
        }),
        destinationAppointmentWindowStart: string().required(
            "Destination appointment window start is required.",
        ),
        destinationAppointmentWindowEnd: string().required(
            "Destination appointment window end is required.",
        ),
        ratingUnits: number().required("Rating units is required."),
        rate: string(),
        mileage: number(),
        otherChargeAmount: string(),
        freightChargeAmount: string(),
        rateMethod: string(),
        customer: string().required("Customer is required."),
        pieces: number(),
        weight: string(),
        readyToBill: boolean().required("Ready to bill is required."),
        trailer: string(),
        trailerType: string().required("Trailer type is required."),
        tractorType: string(),
        temperatureMin: string(),
        temperatureMax: string(),
        bolNumber: string().required("BOL number is required."),
        consigneeRefNumber: string(),
        comment: string().max(100, "Comment must be less than 100 characters."),
        voidedComm: string(),
        autoRate: boolean().required("Auto rate is required."),
        formulaTemplate: string(),
        enteredBy: string().required("Entered by is required."),
        subTotal: string(),
        serviceTye: string(),
        entryMethod: string<ShipmentEntryMethodChoices>().required(
            "Entry method is required.",
        ),
        copyAmount: number().required("Copy amount is required."),
        stops: array().of(stopSchema),
    });

    // Form state and methods
    const shipmentForm = useForm<ShipmentFormValues>({
        resolver: yupResolver(shipmentSchema),
        defaultValues: {
            status: "N",
            proNumber: "",
            originLocation: "",
            originAddress: "",
            destinationLocation: "",
            destinationAddress: "",
            bolNumber: "",
            entryMethod: "MANUAL",
            comment: "",
            ratingUnits: 1,
            autoRate: false,
            readyToBill: false,
            copyAmount: 0,
            enteredBy: user?.id || "",
            temperatureMin: "",
            temperatureMax: "",
            tractorType: "",
            trailerType: "",
            stops: [],
        },
    });

    return {shipmentForm, isShipmentControlLoading, shipmentControlData};
}
