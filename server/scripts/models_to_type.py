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
import typing
from pathlib import Path

import rich
from rich.progress import Progress

type ModelReturnType = dict[str, list[tuple[str, str]]]

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


def snake_to_camel(snake_str: str) -> str:
    """Converts a snake_case string to camelCase.

    This function takes a string formatted in snake_case (with underscores between words)
    and converts it to camelCase, where the first word is in lowercase and each subsequent
    word starts with an uppercase letter, with no intervening spaces or underscores.

    Args:
        snake_str (str): The snake_case string to convert.

    Returns:
        str: The converted camelCase string.

    Examples:
        >>> snake_to_camel("this_is_a_test")
        'thisIsATest'
    """
    components = snake_str.split("_")
    return components[0] + "".join(x.title() for x in components[1:])


class ModelVisitor(ast.NodeVisitor):
    """Visitor class to traverse through the AST of a Django model file.

    This class extends `ast.NodeVisitor` to process class definitions (models) in
    a Django models.py file. It extracts field names, types, and optionalities, and
    stores them in a dictionary for later processing.

    Attributes:
        models (dict): A dictionary storing the model fields and their corresponding types.
    """
    def __init__(self):
        self.models = {}

    def visit_ClassDef(self, node: ast.ClassDef) -> None:
        """Visits a ClassDef node and processes the Django model fields.

        Extracts the field names and types from the class definition, handling
        Django-specific field attributes like `null` and `blank`. Updates the `models`
        attribute with the processed information.

        Args:
            node (ast.ClassDef): The ClassDef node representing a Django model.
        """
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


def parse_model_file(file_path: Path) -> ModelReturnType:
    """Parses a Django model file and extracts model definitions as AST.

    Reads a Django model file, parses it into an AST, and then uses `ModelVisitor`
    to extract model field information.

    Args:
        file_path (Path): The path to the Django model file.

    Returns:
        Dict[str, List[Tuple[str, str]]]: A dictionary where keys are model names and
        values are lists of tuples containing field names and their types.

    Raises:
        Exception: If there is an error in reading or parsing the file.
    """
    try:
        with file_path.open() as file:
            tree = ast.parse(file.read())
    except Exception as e:
        rich.print(f"[red]Error parsing {file_path}: {e}")
        return {}

    visitor = ModelVisitor()
    visitor.visit(tree)
    return visitor.models


def write_ts_interface(
    models: ModelReturnType, output_file: Path
) -> None:
    """Writes TypeScript interfaces for Django models to a file.

    Takes the extracted model information and generates TypeScript interfaces, writing
    them to the specified output file.

    Args:
        models (Dict[str, List[Tuple[str, str]]]): The dictionary containing model data.
        output_file (Path): The path where the TypeScript file will be written.

    Returns:
        None: This function does not return anything.

    Raises:
        Exception: If there is an error in writing to the file.
    """
    try:
        with output_file.open("w") as file:
            for model_name, fields in models.items():
                file.write(f"export type {model_name} = BaseModel & {{\n")
                for field_name, ts_type in fields:
                    file.write(f"  {field_name}: {ts_type};\n")
                file.write("}\n\n")
    except Exception as e:
        rich.print(f"[red]Error writing to {output_file}: {e}")


def process_directory(
    root: Path, ignore_dirs: list[str], progress: Progress, task
) -> None:
    """Processes each Django model file in a directory recursively.

    Walks through the given directory and its subdirectories, ignores specified directories,
    and processes each 'models.py' file found. For each model file, it generates TypeScript
    interfaces and writes them to an output file.

    Args:
        root (Path): The root directory to start processing from.
        ignore_dirs (List[str]): A list of directory names to ignore.
        progress (Progress): Rich library's progress display object.
        task: The current progress task.

    Returns:
        None: This function does not return anything.
    """
    for path in root.glob("**/models.py"):
        if any(ignored_dir in path.parts for ignored_dir in ignore_dirs):
            continue

        if models := parse_model_file(path):
            output_file = Path("ts_models") / f"{path.parent.name}-models.ts"
            write_ts_interface(models, output_file)

            for model_name in models.keys():
                progress.update(
                    task,
                    advance=10,
                    description=f"[cyan]Generating interface for {model_name}...",
                )


def main(ignore_dirs: list[str] | None) -> None:
    ignore_dirs = ignore_dirs or []
    os.makedirs("ts_models", exist_ok=True)

    with Progress() as progress:
        task = progress.add_task("[cyan]Scanning for Django models...", total=100)
        process_directory(Path(".."), ignore_dirs, progress, task)

    rich.print("[green]Conversion complete!")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Convert Django models to TypeScript interfaces."
    )
    parser.add_argument("--ignore_dirs", nargs="+", help="directories to ignore")
    args = parser.parse_args()
    main(args.ignore_dirs)
