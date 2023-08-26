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
from accounts.models import User
from document_generator import models
from organization.models import Organization, BusinessUnit


def save_template_version(
    *,
    template: models.DocumentTemplate,
    new_content: str,
    user: User,
    change_reason: str | None = None,
    organization: Organization,
    business_unit: BusinessUnit,
):
    """Saves a new version of a DocumentTemplate. The function first determines the next version number,
    then creates a new DocumentTemplateVersion with the specified arguments.
    Afterward, updates the template's current version to the newly created version and save the template.

    Args:
        template (models.DocumentTemplate): The DocumentTemplate instance that the new version belongs to.
        new_content (str): The content of the new version.
        user (User): The user who creates this new version.
        change_reason (str, optional): The reason why this change is made, if not given, assumed to be None.
        organization (Organization): The organization that the template belongs to.
        business_unit (BusinessUnit): The business unit that the template belongs to.

    Returns:
        None: This function does not return anything.

    Raises:
        Any exceptions raised by models.DocumentTemplateVersion.objects.create() or template.save()
        will be propagated further.
    """
    if latest_version := template.current_version:
        next_version_number = latest_version.version_number + 1
    else:
        next_version_number = 1

    # Create the new version
    new_version = models.DocumentTemplateVersion.objects.create(
        template=template,
        version_number=next_version_number,
        content=new_content,
        created_by=user,
        change_reason=change_reason,
        organization=organization,
        business_unit=business_unit,
    )

    # Update the template's current version
    template.current_version = new_version
    template.save()
