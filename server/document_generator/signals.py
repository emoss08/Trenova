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
import typing

from document_generator import models, utils


def version_document(
    instance: models.DocumentTemplate, created: bool, **kwargs: typing.Any
) -> None:
    """Creates a new template version on the `DocumentTemplate` instance if it has been freshly created or its content
    has been modified.

    The function will only trigger the creation of a new template version when:
    a) the DocumentTemplate instance has been newly created (i.e., not updated), or
    b) content of the DocumentTemplate instance has been updated (i.e., no instance's content matches the current version's content).

    Args:
        instance (models.DocumentTemplate): instance of the DocumentTemplate model being saved.
        created (bool): boolean flag indicating if instance is being created (True) or updated (False).
        **kwargs (typing.Any): extra arguments.

    Returns:
        None: This function does not return anything.

    Raises:
        Any exceptions raised by utils.save_template_version() function will be propagated further.
    """
    if not created and instance.content != instance.current_version.content:
        utils.save_template_version(instance, instance.content, instance.user_id)

    # Set as 1.0.0 if it's a new template
    if created:
        instance.version = "1.0.0"
        instance.save()
