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
import re
import sys
from textwrap import wrap


def is_properly_formatted(docstring: str) -> bool:
    """Checks if a docstring is properly formatted.

    Args:
        docstring (str): The docstring to check.

    Returns:
        bool: True if the docstring is properly formatted, False otherwise.
    """
    lines = docstring.split("\n")
    indent_next = False
    for i, line in enumerate(lines):
        if indent_next:
            if not line.strip() and (i + 1 == len(lines) or not lines[i + 1].strip()):
                break
            if not line.startswith("        "):
                return False
            indent_next = False
        elif line.strip().startswith(("Args:", "Returns:", "Raises:")):
            indent_next = True
        elif len(line) > 100:
            return False
    return True


def wrap_docstring(docstring: str) -> str:
    """Wraps a docstring to 100 characters per line.

    Args:
        docstring (str): The docstring to wrap.

    Returns:
        str: The wrapped docstring.
    """
    if is_properly_formatted(docstring):
        return docstring

    wrapped_lines = []
    indent = False
    for line in docstring.split("\n"):
        if line.strip().startswith(("Args:", "Returns:", "Raises:")):
            wrapped_lines.append(line)
            indent = True
        elif indent and line.strip():
            if len(line) <= 100:
                wrapped_lines.append(f"    {line}")
            else:
                first_line, *other_lines = wrap(line, 100)
                wrapped_lines.append(f"    {first_line}")
                wrapped_lines.extend("    " * 2 + line for line in other_lines)
        else:
            indent = False
            if len(line) <= 100:
                wrapped_lines.append(line)
            else:
                first_line, *other_lines = wrap(line, 100)
                wrapped_lines.append(first_line)
                wrapped_lines.extend(f"    {line}" for line in other_lines)
    return "\n".join(wrapped_lines)


def reformat_docstrings(file_path: str) -> None:
    """Reformats docstrings in a file.

    Args:
        file_path (str): Path to the file to reformat.

    Returns:
        None: This function does not return anything.
    """

    with open(file_path) as f:
        content = f.read()

    docstrings = re.findall(r'("""[\s\S]*?""")', content)

    for docstring in docstrings:
        wrapped_docstring = wrap_docstring(docstring)
        content = content.replace(docstring, wrapped_docstring)

    with open(file_path, "w") as f:
        f.write(content)


def process_directory(direc: str, file_n: str) -> None:
    """Processes a directory and reformat docstrings in all files with the given name.

    Args:
        direc (str): Path to the directory to process.
        file_n (str): Name of the files to process.

    Returns:
        None: This function does not return anything.
    """
    for root, _, files in os.walk(direc):
        for file in files:
            if file.endswith(".py") and file == file_n:
                file_path = os.path.join(root, file)
                reformat_docstrings(file_path)


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python format_docstrings.py <directory> <file_name>")
        sys.exit(1)

    directory = sys.argv[1]
    file_name = sys.argv[2]
    process_directory(directory, file_name)
