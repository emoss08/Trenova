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
import typing

import cssutils
from django.db.models import Model

from document_generator import models

logger = logging.getLogger(__name__)


def get_nested_attr(*, obj: typing.Any, attr: str) -> typing.Any:
    """Extracts and returns the nested attribute of an object.

    This function allows you to access nested attributes using a string, with each attribute
    separated by a period. For example, if you have an object `a` with attribute `b`
    (which is an object itself), and `b` has an attribute `c`, you can use get_nested_attr(a, 'b.c')
    to access the value of `c`.

    Args:
        obj (typing.Any): The object to extract attribute from.
        attr (str): The attribute key to extract. This can be nested. For instance, 'attr1.attr2.attr3'

    Returns:
        typing.Any: Value of the attribute if it exists. If attribute does not exist, returns None.

    Raises:
        Does not raise any exceptions.
    """
    try:
        attributes = attr.replace(".", "__").split("__")
        for a in attributes:
            obj = getattr(obj, a)
        return obj
    except AttributeError:
        return None


def format_value(*, value: typing.Any, field_type: str) -> str:
    """Formats the input value based on the specified field_type.

    If field_type is `DATE`, it formats the value as YYYY-MM-DD.
    If field_type is `DATETIME`, it formats the value as YYYY-MM-DD HH:MM:SS.
    If field_type is `NUMBER`, it formats the value with two decimal places.
    If field_type is `BOOLEAN`, it returns 'Yes' if the value is truthy, 'No' otherwise.
    For any other field_type, it returns the string representation of the value.

    Args:
        value (typing.Any): The value to be formatted.
        field_type (str): The type of field that determines the formatting of the value.

    Returns:
        str: The formatted value as a string.
    """
    if field_type == models.FieldTypes.DATE:
        return value.strftime("%Y-%m-%d")
    elif field_type == models.FieldTypes.DATETIME:
        return value.strftime("%Y-%m-%d %H:%M:%S")
    elif field_type == models.FieldTypes.NUMBER:
        return f"{value:,.2f}"
    elif field_type == models.FieldTypes.BOOLEAN:
        return "Yes" if value else "No"
    else:
        return str(value)


def apply_template_customizations(
    *, stylesheet: cssutils.css.CSSStyleSheet, template: models.DocumentTemplate
) -> None:
    """Applies customizations to a CSS stylesheet based on the provided DocumentTemplate.

    This function applies each DocTemplateCustomization related to the provided template to the provided stylesheet.
    If a stylesheet rule associated with a customization's css selector is found, the function modifies
    that rule with the customization. If no associated rule is found, the function adds a new rule to the stylesheet.

    Args:
        stylesheet (cssutils.css.CSSStyleSheet): The CSS stylesheet to be customized.
        template (DocumentTemplate): The DocumentTemplate which customizations need to be applied to the stylesheet.

    Returns:
        None: This function does not return anything.

    Raises:
        Any exceptions raised by cssutils.css.CSSStyleSheet.insertRule() or cssutils.css.CSSStyleSheet.cssRules
        will be propagated further.
    """
    customizations = models.DocTemplateCustomization.objects.filter(
        doc_template=template
    )

    for customization in customizations:
        found = False
        for rule in stylesheet.cssRules:
            if (
                rule.type == rule.STYLE_RULE
                and customization.css_selector in rule.selectorText
            ):
                rule.style.setProperty(
                    customization.property_name, customization.property_value
                )
                found = True
                break

        # If the selector is not found in the existing stylesheet, append a new rule
        if not found:
            new_rule_css = f"{customization.css_selector} {{ {customization.property_name}: {customization.property_value}; }}"
            stylesheet.insertRule(new_rule_css)


def serialize_stylesheet(*, stylesheet: cssutils.css.CSSStyleSheet) -> str:
    """Serializes a CSS stylesheet object into a Unicode string.

    Args:
        stylesheet (cssutils.css.CSSStyleSheet): The CSS stylesheet to be serialized.

    Returns:
        str: The serialized stylesheet as a UTF-8 string.

    Raises:
        Any exceptions raised by cssutils.css.CSSStyleSheet.cssText.decode() will be
        propagated further.
    """
    return stylesheet.cssText.decode("utf-8")


def get_parsed_stylesheet(*, theme: models.DocumentTheme) -> cssutils.css.CSSStyleSheet:
    """Parses a CSS stylesheet from the given theme's CSS attribute using cssutils.

    Args:
        theme (models.DocumentTheme, keyword only): The DocumentTheme instance, whose css attribute is
        to be parsed.

    Returns:
        cssutils.css.CSSStyleSheet: The parsed CSS stylesheet.

    Raises:
        Any exceptions raised by cssutils.CSSParser().parseString() will be propagated further.
    """
    parser = cssutils.CSSParser()
    return parser.parseString(theme.css)


def get_template_context_data(
    *, template: models.DocumentTemplate, instance: Model
) -> dict[str, typing.Any]:
    """Constructs and returns the context data for a template given an instance of a model.

    Function retrieves each field bound to the DocumentTemplate object, retrieves its value out of the instance object,
    and adds to the data dictionary to use in the template context.

    If value is a list, function handles it as table data - creates a dictionary of column values for each item in
    the list, and then appends a list of these dictionaries to the data dictionary.

    If conditional rendering is enabled for a field, function checks the corresponding condition
    and skips the processing of the field if the condition fails.

    Args:
        template (models.DocumentTemplate, keyword only): The DocumentTemplate instance whose data bindings to process.
        instance (Model, keyword only): The Model instance from which to fetch field values.

    Returns:
        dict: The context data to use when rendering the template, represented as a dictionary.

    Raises:
        Any exceptions raised by models.DocumentDataBinding.filter(), models.DocumentDataBinding.objects.all()
        or get_nested_attr() will be propagated further.
        Only errors are logged for missing or None values in the instance.
    """
    bindings = models.DocumentDataBinding.objects.filter(template=template)

    data = {}
    for binding in bindings:
        raw_value = get_nested_attr(obj=instance, attr=binding.field_name)

        # Check for conditional rendering
        if binding.conditional_field:
            conditional_value = get_nested_attr(
                obj=instance, attr=binding.conditional_field
            )
            # If the conditional value does not match, skip processing this binding
            if str(conditional_value) != binding.conditional_value:
                continue

        if binding.is_list:
            table_data = []
            for item_instance in raw_value.all():
                row_data = {}
                for column_binding in binding.column_bindings.all():
                    column_value = column_binding.get_column_value(item_instance)
                    key = column_binding.column_name.strip("{}").replace(".", "_")
                    row_data[key] = column_value
                table_data.append(row_data)
            data[binding.placeholder.strip("{}")] = table_data
        else:
            if raw_value or binding.field is None:
                # TODO(Wolfred): Probably will want to raise a ValidationError here that will send back to the user.
                logger.error(
                    f"Failed to fetch '{binding.field_name}' from instance of '{type(instance).__name__}'. Placeholder '{binding.placeholder}' will be left unchanged."
                )
                continue

            value = format_value(value=raw_value, field_type=binding.field.type)
            key = binding.placeholder.strip("{}").replace(".", "_")
            data[key] = value  # type: ignore
    return data


def clone_template(*, template_id: str) -> models.DocumentTemplate:
    """Creates a clone of a DocumentTemplate including its related DocumentDataBinding objects.

    This function fetches the original DocumentTemplate, duplicates it (including related entries),
    updates the pk, name, and cloned_from fields accordingly, and saves the new DocumentTemplate.
    The function then duplicates each DocumentDataBinding related to the original template,
    and changes the template to the new duplicate template.

    Args:
        template_id (str): The ID of the DocumentTemplate to clone.

    Returns:
        models.DocumentTemplate: The cloned DocumentTemplate object.

    Raises:
        models.DocumentTemplate.DoesNotExist: If no DocumentTemplate exists for the given template_id.
        Any exceptions raised by DocumentTemplate.save() or DocumentDataBinding.save() will be
        propagated further.
    """
    # Fetch the original template
    original = models.DocumentTemplate.objects.get(pk=template_id)

    # Duplicate the original template
    original.pk = None
    original.name = f"Copy of {original.name}"
    original.cloned_from = models.DocumentTemplate.objects.get(pk=template_id)
    original.save()

    # Clone related data bindings
    for binding in models.DocumentDataBinding.objects.filter(template=template_id):
        binding.pk = None
        binding.template = original
        binding.save()

    return original
