# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  Monta is free software: you can redistribute it and/or modify                                   -
#  it under the terms of the GNU General Public License as published by                            -
#  the Free Software Foundation, either version 3 of the License, or                               -
#  (at your option) any later version.                                                             -
#                                                                                                  -
#  Monta is distributed in the hope that it will be useful,                                        -
#  but WITHOUT ANY WARRANTY; without even the implied warranty of                                  -
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the                                   -
#  GNU General Public License for more details.                                                    -
#                                                                                                  -
#  You should have received a copy of the GNU General Public License                               -
#  along with Monta.  If not, see <https://www.gnu.org/licenses/>.                                 -
# --------------------------------------------------------------------------------------------------

from typing import Any, final

from django.core import checks
from django.core.checks import CheckMessage, Error
from django.db import models
from django.db.models import CharField
from django.utils.translation import gettext_lazy as _
from django_extensions.db.models import TimeStampedModel
from organization.models import Organization
from django.core.exceptions import FieldDoesNotExist
from django.db.models import F, Q, Prefetch


@final
class StatusChoices(models.TextChoices):
    """
    Status Choices for Order, Stop & Movement Statuses.
    """

    NEW = "N", _("New")
    IN_PROGRESS = "P", _("In Progress")
    COMPLETED = "C", _("Completed")
    HOLD = "H", _("Hold")
    BILLED = "B", _("Billed")
    VOIDED = "V", _("Voided")


@final
class RatingMethodChoices(models.TextChoices):
    """
    Rating Method choices for Order Model
    """

    FLAT = "F", _("Flat Fee")
    PER_MILE = "PM", _("Per Mile")
    PER_STOP = "PS", _("Per Stop")
    POUNDS = "PP", _("Per Pound")


@final
class StopChoices(models.TextChoices):
    """
    Status Choices for the Stop Model
    """

    PICKUP = "P", _("Pickup")
    SPLIT_PICKUP = "SP", _("Split Pickup")
    SPLIT_DROP = "SD", _("Split Drop Off")
    DELIVERY = "D", _("Delivery")
    DROP_OFF = "DO", _("Drop Off")


class GenericModel(TimeStampedModel):
    """
    Generic Model Fields
    """

    organization = models.ForeignKey(
        Organization,
        on_delete=models.CASCADE,
        related_name="%(class)ss",
        related_query_name="%(class)s",
        verbose_name=_("Organization"),
        help_text=_("Organization"),
    )

    class Meta:
        abstract = True

    def save(self, **kwargs: Any) -> None:
        """Save the model instance

        Args:
            **kwargs (Any):

        Returns:
            None
        """

        self.full_clean()
        super().save(**kwargs)


class ChoiceField(CharField):
    """
    A CharField that lets you use Django choices and provides a nice
    representation in the admin.
    """

    description = _("Choice Field")

    def __init__(
            self, *args: Any, db_collation: str | None = None, **kwargs: Any
    ) -> None:
        super().__init__(*args, **kwargs)
        self.db_collation = db_collation
        if self.choices:
            self.max_length = max(len(choice[0]) for choice in self.choices)

    def check(self, **kwargs: Any) -> list[CheckMessage | CheckMessage]:
        """Check the field for errors.

        Check the fields for errors and return a list of Error objects.

        Args:
            **kwargs (Any): Keyword Arguments

        Returns:
            list[CheckMessage | CheckMessage]: List of Error objects
        """
        return [
            *super().check(**kwargs),
            *self._validate_choices_attribute(**kwargs),
        ]

    def _validate_choices_attribute(self, **kwargs: Any) -> list[Error] | list:
        """Validate the choices attribute for the field.

        Validate the choices attribute is set in the field, if not return a list of
        Error objects.

        Args:
            **kwargs (Any): Keyword Arguments

        Returns:
            list{Error} | list: List of Error objects or an empty list
        """
        if self.choices is None:
            return [
                checks.Error(
                    "ChoiceField must define a `choice` attribute.",
                    hint="Add a `choice` attribute to the ChoiceField.",
                    obj=self,
                    id="fields.E120",
                )
            ]
        return []

class AutoSelectRelatedQuerySetMixin:
    """Mixin for automatically selecting related and prefetching objects in a queryset"""

    RELATED_FIELDS_TO_INCLUDE = {
        "OneToOneField": "__all__",
        "ForeignKey": "__all__",
        "ManyToManyField": "__all__",
    }

    def build_related_tree(self, fields, model):
        """
        Builds a tree structure that represents the related objects and fields in the queryset.
        Each node in the tree represents a related object, with its children representing the fields being requested
        for that related object.
        """
        root = {}
        for field in fields:
            if field != "id":
                components = field.split("__")
                current_node = root
                for component in components[:-1]:
                    related_field = model._meta.get_field(component)
                    related_field_class = related_field.__class__.__name__
                    if related_field_class in self.RELATED_FIELDS_TO_INCLUDE:
                        if component not in current_node:
                            current_node[component] = {}
                        current_node = current_node[component]
                current_node[components[-1]] = None
        return root

    def get_queryset(self):
        queryset = super().get_queryset()

        # Get the model associated with the queryset
        model = queryset.model

        # Get the list of fields being requested in the queryset
        fields = set(queryset.only_fields)

        # Build up a tree structure that represents the related objects and fields in the queryset
        related_tree = self.build_related_tree(fields, model)

        # Build up a list of select_related arguments and a set of related fields to include in the query
        select_related_args = []
        related_fields = set()
        self.traverse_related_tree(related_tree, select_related_args, related_fields, model)

        # Call select_related with the related objects and fields from the related_objects dictionary
        queryset = queryset.select_related(*select_related_args).only(
            *fields
        )

        # Determine which related objects to prefetch based on the related_objects dictionary
        prefetch_objects = []
        self.build_prefetch_objects(related_tree, prefetch_objects, model)
        queryset = queryset.prefetch_related(*prefetch_objects)

        return queryset

    def traverse_related_tree(self, related_tree, select_related_args, related_fields, model):
        """
        Traverses the related_tree in a depth-first manner, building up the select_related_args list and the
        related_fields set.
        """
        for related_object, children in related_tree.items():
            select_related_args.append(related_object)
            related_fields.add(related_object)
            if children is not None:
                self.traverse_related_tree(children, select_related_args, related_fields, model)

    def build_prefetch_objects(self, related_tree, prefetch_objects, model):
        """
        Builds up a list of Prefetch objects based on the related_tree.
        """
        for related_object, children in related_tree.items():
            if children is not None:
                related_model = model._meta.get_field(related_object).remote_field.model
                related_fields = [field for field in children.keys() if field != "id"]
                if related_fields:
                    prefetch_objects.append(
                        Prefetch(
                            related_object,
                            queryset=related_model.objects.only(*related_fields)
                        )
                    )
                self.build_prefetch_objects(children, prefetch_objects, related_model)
