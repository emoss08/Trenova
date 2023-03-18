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
import io
from openpyxl import Workbook
from openpyxl.utils import get_column_letter

from reports.views import get_model_by_table_name


def generate_excel_report_as_file(report):
    # Get the model and columns
    model = get_model_by_table_name(report.table)
    columns = report.columns.all().order_by("column_order")

    # Create a workbook and sheet
    wb = Workbook()
    ws = wb.active

    # Write the headers
    for index, column in enumerate(columns):
        col_letter = get_column_letter(index + 1)
        ws[f"{col_letter}1"] = column.column_name

    # Write the data
    row = 2
    for obj in model.objects.all():
        for index, column in enumerate(columns):
            col_letter = get_column_letter(index + 1)
            ws[f"{col_letter}{row}"] = getattr(obj, column.column_name)
        row += 1

    # Save the workbook to a BytesIO object
    file_obj = io.BytesIO()
    wb.save(file_obj)
    file_obj.seek(0)

    return file_obj