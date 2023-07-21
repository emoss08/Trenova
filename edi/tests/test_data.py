# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  The Monta software is licensed under the Business Source License 1.1. You are granted the right -
#  to copy, modify, and redistribute the software, but only for non-production use or with a total -
#  of less than three server instances. Starting from the Change Date (November 16, 2026), the     -
#  software will be made available under version 2 or later of the GNU General Public License.     -
#  If you use the software in violation of this license, your rights under the license will be     -
#  terminated automatically. The software is provided "as is," and the Licensor disclaims all      -
#  warranties and conditions. If you use this license's text or the "Business Source License" name -
#  and trademark, you must comply with the Licensor's covenants, which include specifying the      -
#  Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use     -
#  Grant, and not modifying the license in any other way.                                          -
# --------------------------------------------------------------------------------------------------

test_edi_segments = [
    {
        "code": "B3",
        "name": "Beginning Segment for Carriers Invoice",
        "parser": "B3*B*%s*%s*%s**%s*%s****%s",
        "sequence": 1,
    },
    {
        "code": "C3",
        "name": "Currency",
        "parser": "C3*USD",
        "sequence": 2,
    },
    {
        "code": "N9",
        "name": "Reference Number",
        "parser": "N9*PO*%s**%s",
        "sequence": 3,
    },
    {
        "code": "N1",
        "name": "Name",
        "parser": "N1*PR*%s*25*%s",
        "sequence": 4,
    },
    {
        "code": "N3",
        "name": "Address Information",
        "parser": "N3*%s",
        "sequence": 5,
    },
    {
        "code": "N4",
        "name": "Geographic Location",
        "parser": "N4*%s*%s",
        "sequence": 6,
    },
    {
        "code": "N7",
        "name": "Equipment Details",
        "parser": "N7*%s*%s*%s*N*%s******RR*%s***000*A****%s**%s",
        "sequence": 7,
    },
    {
        "code": "LX",
        "name": "Assigned Number",
        "parser": "LX*1",  # Hard coded line items in the transaction set as 1
        "sequence": 8,
    },
    {
        "code": "L5",
        "name": "Description, Marks, and Numbers",
        "parser": "L5*1*%s*%s*T",
        "sequence": 9,
    },
    {
        "code": "L0",
        "name": "Line Item - Quantity and Weight",
        "parser": "L0*1***%s***%s*TKR",
        "sequence": 10,
    },
    {
        "code": "L1",
        "name": "Rate and Charges",
        "parser": "L1*1*%s*PM*%s****ENS*********1234*MR",
        "sequence": 11,
    },
    {
        "code": "L3",
        "name": "Total Weight and Charges",
        "parser": "L3*%s*N***%s****0*E",
        "sequence": 12,
    },
]
