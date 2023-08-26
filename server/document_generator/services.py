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

import logging

from django.core.exceptions import ValidationError
from django.db.models import Model
from django.template import Context, Template
from django.template.exceptions import TemplateSyntaxError

from document_generator import helpers, models

logger = logging.getLogger(__name__)


def render_document(*, template: models.DocumentTemplate, instance: Model) -> str:
    """Renders a DocumentTemplate by substituting placeholders in the template's content with data from the
    provided instance of a model.

    Template rendering is done with Django's Template system. If the template has a theme, its CSS is parsed,
    customized, and applied to the rendered content. Any syntax errors during template rendering are logged and a
    fallback error message is returned.

    Args:
        template (models.DocumentTemplate, keyword only): The DocumentTemplate instance to be rendered.
        instance (Model, keyword only): The Model instance from which to fetch data for rendering.

    Returns:
        str: The rendered template as a string. If an error occurs during rendering, it returns
             "Error in template rendering."

    Raises:
        Any exceptions raised by get_template_context_data(), Template(), Template.render(),
        get_parsed_stylesheet(), apply_template_customizations() or serialize_stylesheet()
        are handled within the function and logged but not propagated further.
        Render errors result in a "Error in template rendering." message return.
    """
    try:
        data = helpers.get_template_context_data(template=template, instance=instance)
        django_template = Template(template.content)
        rendered_content = django_template.render(Context(data))
    except TemplateSyntaxError as e:
        raise ValidationError(f"Template syntax error: {e}")

    if theme := getattr(template, "theme", None):
        stylesheet = helpers.get_parsed_stylesheet(theme=theme)
        helpers.apply_template_customizations(stylesheet=stylesheet, template=template)
        customized_css = helpers.serialize_stylesheet(stylesheet=stylesheet)

        rendered_content = f"<style>{customized_css}</style>\n{rendered_content}"

    return rendered_content
