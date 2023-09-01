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
import textwrap
import typing
import uuid

from django.contrib.contenttypes.models import ContentType
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _
from djmoney.money import Money

from utils.models import GenericModel


@typing.final
class FieldTypes(models.TextChoices):
    """Choices for the type of field."""

    STRING = "string", _("String")
    NUMBER = "number", _("Number")
    DATE = "date", _("Date")
    DATETIME = "datetime", _("Datetime")
    BOOLEAN = "boolean", _("Boolean")
    MONEY = "money", _("Money")


class DocumentTemplate(GenericModel):
    """
    Stores Template field information related to a :model:`billing.DocumentClassification`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    name = models.CharField(
        verbose_name=_("Name"),
        max_length=255,
        help_text=_(
            "Name of the Template (e.g: 'Invoice', 'Credit Memo', 'Purchase Order', etc.)"
        ),
    )
    content = models.TextField(
        verbose_name=_("Content"),
        help_text="The template content with custom syntax (e.g., {invoice.order.pro_number}).",
    )
    document_classification = models.OneToOneField(
        to="billing.DocumentClassification",
        on_delete=models.CASCADE,
        related_name="document_templates",
        verbose_name=_("Document Classification"),
        help_text=_("The classification of the document (e.g., Invoice, Credit Memo)."),
    )
    theme = models.ForeignKey(
        to="DocumentTheme",
        on_delete=models.SET_NULL,
        related_name="document_templates",
        verbose_name=_("Theme"),
        blank=True,
        null=True,
        help_text=_("The theme associated with this template."),
    )
    current_version = models.ForeignKey(
        "DocumentTemplateVersion",
        on_delete=models.SET_NULL,
        related_name="current_for_template",
        null=True,
        blank=True,
    )
    user_id = models.ForeignKey(
        verbose_name=_("Created By"),
        to="accounts.User",
        on_delete=models.RESTRICT,
    )
    cloned_from = models.ForeignKey(
        verbose_name=_("Cloned From"),
        to="self",
        on_delete=models.SET_NULL,
        null=True,
        blank=True,
    )

    class Meta:
        """
        Meta class for Document Template Model
        """

        verbose_name = _("Document Template")
        verbose_name_plural = _("Document Templates")
        db_table = "document_template"

    def __str__(self) -> str:
        """String representation of the document template.

        Returns:
            str: A shortened version of the template name.
        """
        return textwrap.shorten(self.name, width=50, placeholder="...")

    def save(self, *args: typing.Any, **kwargs: typing.Any) -> None:
        """
        Saves the DocumentTemplate object. If this is a new template or if the content has changed,
        a new version will be created.

        Args:
            *args (typing.Any): Any positional arguments.
            **kwargs (typing.Any): Any keyword arguments.

        Returns:
            None: This function does not return anything.

        """
        super().save(*args, **kwargs)

        # Check if this is a new template or if the content has changed
        if not self.current_version or self.content != self.current_version.content:
            from document_generator.utils import save_template_version

            save_template_version(
                template=self,
                new_content=self.content,
                user=self.user_id,
                change_reason=None,
                organization=self.organization,
                business_unit=self.business_unit,
            )

    def get_absolute_url(self) -> str:
        """Get the absolute URL of the DocumentTemplate object.

        Returns:
            str: The absolute URL of the DocumentTemplate object.
        """
        return reverse("document-template-detail", kwargs={"pk": self.pk})


class DocumentTemplateVersion(GenericModel):
    """
    Stores Template version information related to a :model:`document_generator.DocumentTemplate`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    template = models.ForeignKey(
        verbose_name=_("Template"),
        to=DocumentTemplate,
        related_name="versions",
        help_text=_("The template to which this version belongs."),
        on_delete=models.CASCADE,
    )
    version_number = models.PositiveIntegerField(
        verbose_name=_("Version Number"),
        help_text=_("The version number of the template."),
    )
    content = models.TextField(
        verbose_name=_("Content"),
        help_text="The template content with custom syntax (e.g., {invoice.order.pro_number}).",
    )
    created_at = models.DateTimeField(
        verbose_name=_("Created At"),
        help_text=_("The date and time when the version was created."),
        auto_now_add=True,
    )
    created_by = models.ForeignKey(
        verbose_name=_("Created By"),
        to="accounts.User",
        on_delete=models.SET_NULL,
        null=True,
        blank=True,
    )
    change_reason = models.TextField(
        verbose_name=_("Change Reason"),
        help_text=_("The reason for the change in the template."),
        null=True,
        blank=True,
    )

    class Meta:
        """
        Meta class for Document Template version model.
        """

        verbose_name = _("Document Template Version")
        verbose_name_plural = _("Document Template Versions")
        db_table = "document_template_version"
        constraints = [
            models.UniqueConstraint(
                fields=["template", "version_number"],
                name="unique_document_template_version",
            )
        ]

    def __str__(self) -> str:
        """Returns a shortened version of the template name.

        Parameters:
            None.

        Returns:
            str: A shortened version of the template name.
        """
        return textwrap.shorten(self.template.name, width=50, placeholder="...")

    def get_absolute_url(self) -> str:
        """Get the absolute URL of the DocumentTemplateVersion object.

        Returns:
            str: The absolute URL of the DocumentTemplateVersion object.
        """
        return reverse("document-template-version-detail", kwargs={"pk": self.pk})


class TemplateField(GenericModel):
    """
    Stores Template field information related to a :model:`document_generator.DocumentTemplate`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    name = models.CharField(
        max_length=255, help_text="Name of the field (e.g., order.pro_number)."
    )
    label = models.CharField(
        max_length=255, help_text="Display name (e.g., 'Order Pro Number')."
    )
    type = models.CharField(
        max_length=50,
        choices=FieldTypes.choices,
        help_text="Data type (e.g., string, number, date).",
    )
    template = models.ForeignKey(
        DocumentTemplate,
        on_delete=models.CASCADE,
        related_name="fields",
        help_text="ForeignKey to the DocumentTemplate model.",
    )

    class Meta:
        """
        Meta class for Template field Model
        """

        verbose_name = _("Template Field")
        verbose_name_plural = _("Template Fields")
        db_table = "template_field"

    def __str__(self) -> str:
        """String representation of the template field.

        Returns:
            str: A shortened version of the template field name.
        """
        return textwrap.shorten(self.name, width=50, placeholder="...")

    def get_absolute_url(self) -> str:
        """Get the absolute URL of the TemplateField object.

        Returns:
            str: The absolute URL of the TemplateField object.
        """
        return reverse("template-field-detail", kwargs={"pk": self.pk})


class DocumentTheme(GenericModel):
    """
    Stores Document Theme information.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    name = models.CharField(
        verbose_name=_("Theme Name"),
        max_length=255,
        help_text=_("Name of the predefined theme."),
    )
    css = models.TextField(
        verbose_name=_("CSS Content"), help_text=_("The CSS content for the theme.")
    )
    primary_color = models.CharField(
        verbose_name=_("Primary Color"),
        help_text=_("The primary color for the document."),
        max_length=7,
        default="#FFFFFF",
    )
    secondary_color = models.CharField(
        verbose_name=_("Secondary Color"),
        help_text=_("The secondary color for the document."),
        max_length=7,
        default="#000000",
    )
    font_family = models.CharField(
        verbose_name=_("Font Family"),
        help_text=_("The font family for the document."),
        max_length=50,
        default="Arial, sans-serif",
    )
    header_font_size = models.PositiveIntegerField(
        verbose_name=_("Header Font Size"),
        help_text=_("The font size for the header of the document."),
        default=16,
    )
    body_font_size = models.PositiveIntegerField(
        verbose_name=_("Body Font Size"),
        help_text=_("The font size for the body of the document."),
        default=14,
    )

    class Meta:
        """
        Meta Class for Document Theme Model
        """

        verbose_name = _("Document Theme")
        verbose_name_plural = _("Document Themes")
        db_table = "document_theme"
        constraints = [
            models.UniqueConstraint(
                fields=["name", "organization"],
                name="unique_document_theme_per_organization",
            )
        ]

    def __str__(self) -> str:
        """String representation of the document theme.

        Returns:
            str: A shortened version of the theme name.
        """
        return textwrap.shorten(self.name, width=50, placeholder="...")

    def get_absolute_url(self) -> str:
        """Get the absolute URL of the DocumentTheme object.

        Returns:
            str: The absolute URL of the DocumentTheme object.
        """
        return reverse("document-theme-detail", kwargs={"pk": self.pk})


class DocTemplateCustomization(GenericModel):
    """
    Stores Document Template Customization information related to a
    :model:`document_generator.DocumentTemplate`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    doc_template = models.ForeignKey(
        to=DocumentTemplate,
        on_delete=models.CASCADE,
        related_name="customizations",
        verbose_name=_("Template"),
        help_text=_("The template to customize."),
    )
    css_selector = models.CharField(
        verbose_name=_("CSS Selector"),
        max_length=255,
        help_text=_("The CSS selector to customize."),
    )
    property_name = models.CharField(
        verbose_name=_("Property Name"),
        max_length=255,
        help_text=_("The property name to customize."),
    )
    property_value = models.CharField(
        verbose_name=_("Property Value"),
        max_length=255,
        help_text=_("The property value to customize."),
    )

    class Meta:
        """
        Meta class for Document Template Customization Model
        """

        verbose_name = _("Customization")
        verbose_name_plural = _("Customizations")
        db_table = "doc_template_customization"
        constraints = [
            models.UniqueConstraint(
                fields=["doc_template", "css_selector", "property_name"],
                name="unique_customization",
            )
        ]

    def __str__(self) -> str:
        """String representation of the document template customization.

        Returns:
            str: A shortened version of the template name.
        """
        return textwrap.shorten(self.doc_template.name, width=50, placeholder="...")

    def get_absolute_url(self) -> str:
        """Get the absolute URL of the DocTemplateCustomization object.

        Returns:
            str: The absolute URL of the DocTemplateCustomization object.
        """
        return reverse("doc-template-customization-detail", kwargs={"pk": self.pk})


class DocumentDataBinding(GenericModel):
    """
    Stores Document Data binding information related to a :model:`document_generator.DocumentTemplate`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    placeholder = models.CharField(
        verbose_name=_("Placeholder"),
        max_length=255,
        help_text=_("The placeholder used in the document (e.g., {invoice.total})."),
    )
    field_name = models.CharField(
        verbose_name=_("Field Name"),
        max_length=255,
        help_text="The field name in the model to fetch data from.",
    )
    content_type = models.ForeignKey(
        verbose_name=_("Content Type"),
        to=ContentType,
        on_delete=models.CASCADE,
        help_text=_("The content type of the model to fetch data from."),
    )
    template = models.ForeignKey(
        verbose_name=_("Template"),
        to=DocumentTemplate,
        on_delete=models.CASCADE,
        related_name="data_bindings",
        help_text="ForeignKey to the DocumentTemplate model.",
    )
    field = models.ForeignKey(
        verbose_name=_("Field"),
        to=TemplateField,
        on_delete=models.CASCADE,
        related_name="data_bindings",
        help_text="ForeignKey to the TemplateField model.",
        blank=True,
        null=True,
    )
    is_list = models.BooleanField(
        verbose_name=_("Is List"),
        default=False,
        help_text=_(
            "Whether the field is a list of values (e.g., invoice.line_items)."
        ),
    )
    related_model = models.ForeignKey(
        verbose_name=_("Related Model"),
        to=ContentType,
        on_delete=models.CASCADE,
        related_name="data_bindings",
        blank=True,
        null=True,
        help_text=_(
            "The related model to fetch data from (e.g., invoice.line_items.product)."
        ),
    )
    conditional_field = models.CharField(
        verbose_name=_("Conditional Field"),
        max_length=255,
        blank=True,
        null=True,
        help_text=_(
            "Name of the field to check for a condition. If set, it will be used to determine if the binding"
            " should be rendered."
        ),
    )

    conditional_value = models.CharField(
        verbose_name=_("Conditional Value"),
        max_length=255,
        blank=True,
        null=True,
        help_text=_(
            "Value to check against the conditional_field. If they match, the binding will be rendered."
        ),
    )

    class Meta:
        """
        Meta class for Document Data Binding Model
        """

        verbose_name = _("Document Data Binding")
        verbose_name_plural = _("Document Data Bindings")
        db_table = "document_data_binding"

    def __str__(self) -> str:
        """String representation of the document data binding.

        Returns:
            str: A shortened version of the placeholder.
        """
        return textwrap.shorten(self.placeholder, width=50, placeholder="...")

    def get_absolute_url(self) -> str:
        """Get the absolute URL of the DocumentDataBinding object.

        Returns:
            str: The absolute URL of the DocumentDataBinding object.
        """
        return reverse("document-data-binding-detail", kwargs={"pk": self.pk})

    def get_value(self, instance: models.Model) -> typing.Any:
        """Get the value of the field from the instance.

        Args:
            instance (models.Model): The instance of the model to fetch data from.

        Returns:
            typing.Any: The value of the field from the instance.
        """
        return getattr(instance, self.field_name)


class DocumentTableColumnBinding(GenericModel):
    """
    Stores Data Table Column Binding information for a :model:`document_generator.DocumentDataBinding`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    table_binding = models.ForeignKey(
        verbose_name=_("Table Binding"),
        to=DocumentDataBinding,
        on_delete=models.CASCADE,
        related_name="column_bindings",
        help_text=_(
            "ForeignKey to the DocumentDataBinding model representing the table."
        ),
    )
    column_name = models.CharField(
        verbose_name=_("Column Name"),
        max_length=255,
        help_text=_("The name or header of the column in the table."),
    )
    field_name = models.CharField(
        verbose_name=_("Field Name"),
        max_length=255,
        help_text=_("The field name in the related model to fetch data from."),
    )
    field = models.ForeignKey(
        verbose_name=_("Field"),
        to=TemplateField,
        on_delete=models.CASCADE,
        related_name="table_column_bindings",
        help_text="ForeignKey to the TemplateField model.",
        blank=True,
        null=True,
    )

    class Meta:
        """
        Meta class for Document Table Column Binding Model
        """

        verbose_name = _("Document Table Column Binding")
        verbose_name_plural = _("Document Table Column Bindings")
        db_table = "document_table_column_binding"
        constraints = [
            models.UniqueConstraint(
                fields=["table_binding", "column_name"],
                name="unique_document_table_column_binding",
            )
        ]

    def __str__(self) -> str:
        """String representation of the document table column binding.

        Returns:
            str: A shortened version of the placeholder.
        """
        return textwrap.shorten(
            f"{self.table_binding.placeholder} - {self.column_name}",
            width=50,
            placeholder="...",
        )

    def get_absolute_url(self) -> str:
        """Get the absolute URL of the DocumentTableColumnBinding object.

        Returns:
            str: The absolute URL of the DocumentTableColumnBinding object.
        """
        return reverse("document-table-column-binding-detail", kwargs={"pk": self.pk})

    def get_column_value(self, instance: models.Model) -> typing.Any:
        """Get the value of the field from the instance.

        Args:
            instance(models.Model): The instance of the model to fetch data from.

        Returns:
            typing.Any: The value of the field from the instance.
        """
        from document_generator.helpers import get_nested_attr

        value = get_nested_attr(obj=instance, attr=self.field_name)
        if isinstance(value, Money):
            return f"{value.amount} {value.currency}"
        return value
