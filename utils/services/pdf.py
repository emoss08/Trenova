"""
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""
from io import BytesIO

from django.http import HttpRequest, HttpResponse
from django.template.loader import get_template
from weasyprint import CSS, HTML


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
    response[
        "Content-Disposition"
    ] = 'attachment; filename="generated_document.pdf"'
    return response
