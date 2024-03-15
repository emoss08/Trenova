# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
from typing import Any

from django.core.management.base import BaseCommand

from dispatch.models import CommentType
from organization.models import Organization

STANDARD_CODES = [
    {
        "Name": "Dispatch",
        "Description": "Type of comment used when dispatching a shipment to a driver.",
    },
    {
        "Name": "Billing",
        "Description": "Type of comment that will be transferred along with shipment to the billing queue.",
    },
    {
        "Name": "Hot",
        "Description": "Type of comment that will show for the shipment overall. Usually includes important information about the shipment.",
    },
]


class Command(BaseCommand):
    help = "Generates standard comments types for all organizations."

    def get_all_organization(self) -> list[Organization]:
        # This instance method now correctly uses self to access model objects
        return Organization.objects.all()

    def create_comment_types(self) -> list[CommentType]:
        """Create the Dispatch, Billing, and Hot Comment Types for each organization if
        the comment types do not exist. If they do exist then skip over that organization.

        Returns:
            List[CommentType]: The list of comment types created.
        """
        comment_types = []
        for org in self.get_all_organization():
            for code in STANDARD_CODES:
                # Correctly unpacking the 'code' dictionary to map to the model fields
                comment_type, created = CommentType.objects.get_or_create(
                    organization=org,
                    business_unit=org.business_unit,
                    name=code["Name"],
                    defaults={"description": code["Description"]},
                )
                if created:
                    comment_types.append(comment_type)
        return comment_types

    def handle(self, *args: Any, **options: Any) -> None:
        """Create the standard comment types for all organizations."""
        self.stdout.write("Creating standard comment types for all organizations...")
        comment_types = self.create_comment_types()
        self.stdout.write(f"Created {len(comment_types)} comment types.")
