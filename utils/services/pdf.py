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
import json
from io import BytesIO
from uuid import UUID

from django.http import HttpRequest, HttpResponse
from django.template.loader import get_template
from weasyprint import CSS, HTML


class UUIDEncoder(json.JSONEncoder):
    def default(self, obj):
        if isinstance(obj, UUID):
            # if the obj is uuid, we simply return the value of uuid
            return obj.hex
        return json.JSONEncoder.default(self, obj)


def render_to_pdf(
    template_src: str, request: HttpRequest, context_dict: dict | None = None
) -> HttpResponse:
    """Render an HTML template as a PDF response.

    Args:
        template_src (str): The source path of the template file to render.
        request (HttpRequest): The HTTP request object associated with the view.
        context_dict (dict, optional): The context dictionary to pass to the template.
            Defaults to an empty dictionary.

    Returns:
        HttpResponse or None: The PDF response object if the rendering succeeds. Returns None if there is an error.

    Raises:
        TemplateDoesNotExist: If the specified template does not exist.
        WeasyPrintError: If there is an error generating the PDF from the rendered HTML.

    Notes:
        This function uses the Django template loader to load and render the specified template,
        and the WeasyPrint library to generate the PDF from the rendered HTML.
        It sets the PDF content type to "application/pdf".
    """

    if not isinstance(template_src, str):
        raise ValueError("Invalid template name")

    if context_dict is None:
        context_dict = {}

    template = get_template(template_src)
    html = template.render(context_dict)

    css = CSS(string="@page { margin-top: 0; }")
    pdf_file = BytesIO()
    HTML(string=html, base_url=request.build_absolute_uri()).write_pdf(
        target=pdf_file, stylesheets=[css], presentational_hints=True
    )

    response = HttpResponse(pdf_file.getvalue(), content_type="application/pdf")
    response["Content-Disposition"] = 'attachment; filename="generated_document.pdf"'
    return response
