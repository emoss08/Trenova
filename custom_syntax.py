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

import re
from typing import Optional, List, Tuple

model_template = """
class {class_name}(models.Model):
    {fields}
"""

field_template = "{field_name} = models.{field_type}({field_args})"

def parse_create_model(line) -> Optional[str]:
    return match[1] if (match := re.search(r"CREATE MODEL (\w+)", line)) else None


def parse_app(line) -> Optional[str]:
    return app_name[1] if (app_name := re.search(r"APP (\w+)", line)) else None


def parse_field(lines: str) -> List[Tuple[str, str, str]]:
    fields_data: List = re.findall(
        r"(\w+): (\w+)Field ((?:\w+=\w+(?:\.\w+)* )*\w+=\w+(?:\.\w+)*)", lines
    )

    fields = []
    for field_data in fields_data:
        field_name, field_type, field_args_str = field_data
        field_args_list = [
            arg.strip() for arg in field_args_str.split(" ") if arg.strip()
        ]
        field_args = ", ".join(field_args_list)
        fields.append((field_name.strip(), field_type, field_args))
    return fields

def generate_code(custom_syntax: str) -> str:
    lines: List[str] = custom_syntax.strip().split("\n")
    class_name: Optional[str] = parse_create_model(lines[0])
    app_name: Optional[str] = parse_app(lines[1])

    field_lines = []
    i = 2
    while not lines[i].startswith("END"):
        field_lines.append(lines[i])
        i += 1

    fields_data: List[Tuple[str, str, str]] = parse_field("\n".join(field_lines))

    field_codes = []
    for field_name, field_type, field_args in fields_data:
        field_code = f'{field_name} = models.{field_type}Field(\n'
        for arg in field_args.split(", "):
            key, value = [x.strip() for x in arg.split("=")]
            field_code += f'    {key.lower()}={value},\n'
        field_code += ')'
        field_codes.append(field_code)

    return model_template.format(
        class_name=class_name, fields="\n    ".join(field_codes)
    )

custom_syntax = """
CREATE MODEL User
APP Accounts
FIELDS
  id: UUIDField PRIMARY_KEY=True DEFAULT=uuid.uuid4 EDITABLE=False UNIQUE=True
  username: CharField MAX_LENGTH=150 UNIQUE=True
END
"""

generated_code = generate_code(custom_syntax)
print(generated_code)

"""
Output:

class User(models.Model):
    id = models.UUIDField(
    primary_key=True,
    default=uuid.uuid4,
    editable=False,
    unique=True,
)
    username = models.CharField(
    max_length=150,
    unique=True,
)
"""