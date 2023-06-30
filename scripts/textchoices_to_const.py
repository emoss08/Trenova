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

import os
import ast
from collections.abc import Generator
from typing import Any


def find_files(directory: str, extension: str) -> Generator[bytes | str, Any, None]:
    """Generate file paths that match a given extension in a directory.

    This generator walks a directory tree, starting at `directory`, and yields
    paths to files that have the specified `extension`.

    Args:
        directory (str): The root directory from which the search starts.
        extension (str): The file extension to match.

    Yields:
        str: Full file path for each file that matches the given extension.
    """
    for root, dirs, files in os.walk(directory):
        for file in files:
            if file.endswith(extension):
                yield os.path.join(root, file)


def find_and_convert_choices(directory: str) -> None:
    """Find Django TextChoices in Python files and print their TypeScript equivalent.

    This function uses the `ast` module to parse Python source code and look for
    Django TextChoices declarations. For each TextChoices found, it prints a TypeScript
    equivalent.

    Args:
        directory (str): The root directory where the search for Python files starts.

    Returns
        None: This function does not return anything.
    """
    for file in find_files(directory, ".py"):
        with open(file) as f:
            content = f.read()
            try:
                module = ast.parse(content)
            except SyntaxError as e:
                print(f"Skipping file {file} due to syntax error: {e}")
                continue
            for class_node in [
                n for n in ast.walk(module) if isinstance(n, ast.ClassDef)
            ]:
                for base in class_node.bases:
                    if isinstance(base, ast.Attribute) and base.attr == "TextChoices":
                        ts_conversion = []
                        for assign_node in [
                            n for n in ast.walk(class_node) if isinstance(n, ast.Assign)
                        ]:
                            for _ in assign_node.targets:
                                if (
                                    isinstance(assign_node.value, ast.Tuple)
                                    and len(assign_node.value.elts) == 2
                                ):
                                    value = assign_node.value.elts[0].s
                                    label = assign_node.value.elts[1].args[0].s
                                    ts_conversion.append(
                                        f'  {{ value: "{value}", label: "{label}" }},'
                                    )
                        ts_string = (
                            f"export const {class_node.name} = [\n"
                            + "\n".join(ts_conversion)
                            + "\n];"
                        )
                        print(ts_string)
            f.close()


find_and_convert_choices("accounts")
