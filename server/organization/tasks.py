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

from celery import shared_task

from organization import models


@shared_task
def create_organization_feature_flags(*, feature_flag_id: str) -> None:
    """Task that creates an OrganizationFeatureFlag for each Organization.

    Once an organization is created, it should have an OrganizationFeatureFlag for each FeatureFlag
    that exists in the system. This task is responsible for creating those OrganizationFeatureFlags
    for a given FeatureFlag.

    Args:
        feature_flag_id (str): The id of the FeatureFlag to create the OrganizationFeatureFlags for.

    Returns:
        None: This function does not return anything.
    """
    feature_flag = models.FeatureFlag.objects.get(id__exact=feature_flag_id)
    organizations = models.Organization.objects.all()

    org_feature_flags = [
        models.OrganizationFeatureFlag(
            organization=organization, feature_flag=feature_flag
        )
        for organization in organizations
    ]
    models.OrganizationFeatureFlag.objects.bulk_create(org_feature_flags)
