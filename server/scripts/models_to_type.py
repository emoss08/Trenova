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
import argparse
import ast
import os

import rich
from rich.progress import Progress

# Define the mapping from Django fields to TypeScript types
TYPE_MAP = {
    "UUIDField": "string",
    "OneToOneField": "string",
    "PositiveIntegerField": "number",
    "DecimalField": "string",
    "BooleanField": "boolean",
    "CharField": "string",
    "TextField": "string",
    "DateField": "Date",
    "ChoiceField": "string",
    "ForeignKey": "string",
    "ManyToManyField": "string[]",
    "JSONField": "any",
    "AutoField": "number",
    "BigIntegerField": "number",
    "BinaryField": "any",
    "DateTimeField": "Date",
    "DurationField": "any",
    "EmailField": "string",
    "FileField": "any",
    "FilePathField": "any",
    "FloatField": "number",
    "GenericIPAddressField": "string",
    "ImageField": "any",
    "IntegerField": "number",
    "IPAddressField": "string",
    "NullBooleanField": "boolean",
    "PositiveSmallIntegerField": "number",
    "MoneyField": "number",
    "USZipCodeField": "string",
    "USStateField": "string",
}


# Function to convert snake_case to camelCase
def snake_to_camel(snake_str: str) -> str:
    components = snake_str.split("_")
    return components[0] + "".join(x.title() for x in components[1:])


class ModelVisitor(ast.NodeVisitor):
    def __init__(self):
        self.models = {}

    def visit_ClassDef(self, node: ast.ClassDef) -> None:
        fields = []

        for stmt in node.body:
            if isinstance(stmt, ast.Assign):
                for target in stmt.targets:
                    if isinstance(target, ast.Name):
                        field_name = target.id
                        field_type = None
                        is_nullable = False
                        is_blankable = False

                        if isinstance(stmt.value, ast.Call):
                            if isinstance(stmt.value.func, ast.Attribute):
                                field_type = stmt.value.func.attr
                            elif isinstance(stmt.value.func, ast.Name):
                                field_type = stmt.value.func.id

                            for keyword in stmt.value.keywords:
                                if (
                                    keyword.arg == "null"
                                    and isinstance(keyword.value, ast.Constant)
                                    and keyword.value.value
                                ):
                                    is_nullable = True
                                if (
                                    keyword.arg == "blank"
                                    and isinstance(keyword.value, ast.Constant)
                                    and keyword.value.value
                                ):
                                    is_blankable = True

                            if ts_type := TYPE_MAP.get(field_type):  # type: ignore
                                if is_nullable:
                                    ts_type += " | null"
                                if is_blankable:
                                    field_name += "?"
                                camel_case_name = snake_to_camel(
                                    field_name
                                )  # Convert to camelCase
                                fields.append(
                                    (camel_case_name, ts_type)
                                )  # Use camelCase name

        if fields:
            self.models[node.name] = fields


def parse_model_file(path: str) -> dict[str, list[tuple[str, str]]]:
    with open(path) as file:
        tree = ast.parse(file.read())

    visitor = ModelVisitor()
    visitor.visit(tree)
    return visitor.models


def write_ts_interface(
    models: dict[str, list[tuple[str, str]]], output_path: str
) -> None:
    with open(output_path, "w") as file:
        for model_name, fields in models.items():
            file.write(f"export interface {model_name} extends BaseModel {{\n")
            for field_name, ts_type in fields:
                file.write(f"  {field_name}: {ts_type};\n")
            file.write("}\n\n")


def main(ignore_dirs: list[str]) -> None:
    os.makedirs("ts_models", exist_ok=True)

    with Progress() as progress:
        task = progress.add_task("[cyan]Starting...", total=100)

        for root, dirs, files in os.walk(".."):
            dirs[:] = [d for d in dirs if d not in ignore_dirs]

            for name in files:
                if name == "models.py":
                    file_path = os.path.join(root, name)
                    models = parse_model_file(file_path)
                    dir_name = os.path.basename(root)
                    output_file = os.path.join("ts_models", f"{dir_name}-models.ts")
                    write_ts_interface(models, output_file)

                    for model_name in models.keys():
                        progress.update(
                            task,
                            advance=10,
                            description=f"[cyan]Generating interface for {model_name}...",
                        )

        progress.stop()
        rich.print("[green]Done!")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Convert Django models to TypeScript interfaces."
    )
    parser.add_argument("--ignore_dirs", nargs="+", help="directories to ignore")
    args = parser.parse_args()
    main(args.ignore_dirs or [])
