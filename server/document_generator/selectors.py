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
from document_generator import models


def get_template_content(
    template: models.DocumentTemplate, version_number: int | None = None
) -> str:
    """Retrieves the template's content from DocumentTemplateVersion for given version number.

    If the version number is not provided, function will return the content of current version.

    Args:
        template (models.DocumentTemplate): The DocumentTemplate instance to retrieve content from.
        version_number (int, optional): The version number to retrieve content from. Defaults to None, in which case
        the function will return the content of the template's current version.

    Returns:
        str: The content of the specified version of the template.

    Raises:
        models.DocumentTemplateVersion.DoesNotExist: If no DocumentTemplateVersion exists for the given template
        and version_number.
    """
    if not version_number:
        # By default, return the content of the current version
        return template.current_version.content
    version = models.DocumentTemplateVersion.objects.get(
        template=template, version_number=version_number
    )
    return version.content
