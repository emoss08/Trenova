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
import uuid
from typing import Any, final

from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from utils.models import ChoiceField, GenericModel


class HazardousMaterial(GenericModel):  # type: ignore
    """
    A class representing a hazardous material.

    This class stores information about a hazardous material, including its name, description, hazard
    class, packing group, ERG number, proper shipping name, and UN number. It also defines two
    subclasses, `HazardousClassChoices` and `PackingGroupChoices`, which define the possible values
    for the `hazard_class` and `packing_group` fields.
    """

    @final
    class HazardousClassChoices(models.TextChoices):
        """
        A class representing the possible hazardous class choices.

        This class inherits from the `models.TextChoices` class and defines several constants
        representing the different hazardous classes defined in the United Nations' Recommendations
        on the Transport of Dangerous Goods.
        """

        CLASS_1_1 = "1.1", _("Division 1.1: Mass Explosive Hazard")
        CLASS_1_2 = "1.2", _("Division 1.2: Projection Hazard")
        CLASS_1_3 = "1.3", _(
            "Division 1.3: Fire and/or Minor Blast/Minor Projection Hazard"
        )
        CLASS_1_4 = "1.4", _("Division 1.4: Minor Explosion Hazard")
        CLASS_1_5 = "1.5", _(
            "Division 1.5: Very Insensitive With Mass Explosion Hazard"
        )
        CLASS_1_6 = "1.6", _(
            "Division 1.6: Extremely Insensitive; No Mass Explosion Hazard"
        )
        CLASS_2_1 = "2.1", _("Division 2.1: Flammable Gases")
        CLASS_2_2 = "2.2", _("Division 2.2: Non-Flammable Gases")
        CLASS_2_3 = "2.3", _("Division 2.3: Poisonous Gases")
        CLASS_3 = "3", _("Division 3: Flammable Liquids")
        CLASS_4_1 = "4.1", _("Division 4.1: Flammable Solids")
        CLASS_4_2 = "4.2", _("Division 4.2: Spontaneously Combustible Solids")
        CLASS_4_3 = "4.3", _("Division 4.3: Dangerous When Wet")
        CLASS_5_1 = "5.1", _("Division 5.1: Oxidizing Substances")
        CLASS_5_2 = "5.2", _("Division 5.2: Organic Peroxides")
        CLASS_6_1 = "6.1", _("Division 6.1: Toxic Substances")
        CLASS_6_2 = "6.2", _("Division 6.2: Infectious Substances")
        CLASS_7 = "7", _("Division 7: Radioactive Material")
        CLASS_8 = "8", _("Division 8: Corrosive Substances")
        CLASS_9 = "9", _("Division 9: Miscellaneous Hazardous Substances and Articles")

    @final
    class PackingGroupChoices(models.TextChoices):
        """
        A class representing the possible packing group choices.

        This class inherits from the `models.TextChoices` class and defines several constants representing
        the three possible packing groups defined in the United Nations' Recommendations on the Transport
        of Dangerous Goods.
        """

        ONE = "I", _("I")
        TWO = "II", _("II")
        THREE = "III", _("III")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    is_active = models.BooleanField(
        default=True,
        verbose_name=_("Is Active"),
        help_text=_("Whether or not the hazardous material is active."),
    )
    name = models.CharField(
        _("Name"),
        max_length=255,
        help_text=_("Name of the Hazardous Class"),
    )
    description = models.TextField(
        _("Description"),
        blank=True,
        help_text=_("Description of the Hazardous Class"),
    )
    hazard_class = ChoiceField(
        _("Hazard Class"),
        choices=HazardousClassChoices.choices,
        help_text=_("Hazard Class of the Hazardous Material"),
    )
    packing_group = ChoiceField(
        _("Packing Group"),
        choices=PackingGroupChoices.choices,
        help_text=_("Packing Group of the Hazardous Material"),
        blank=True,
    )
    erg_number = models.CharField(
        _("ERG Number"),
        max_length=255,
        blank=True,
    )
    proper_shipping_name = models.TextField(
        _("Proper Shipping Name"),
        help_text=_("Proper Shipping Name of the Hazardous Material"),
        blank=True,
    )

    class Meta:
        verbose_name = _("Hazardous Material")
        verbose_name_plural = _("Hazardous Materials")
        ordering = ["name"]
        db_table = "hazardous_material"

    def __str__(self) -> str:
        """Hazardous Material String Representation

        Returns:
            str: Hazardous Material Name
        """
        return textwrap.wrap(self.name, 50)[0]

    def update_hazmat(self, **kwargs: Any) -> None:
        """Update Hazardous Material

        Args:
            **kwargs (Any): Hazardous Material Fields
        """
        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()

    def get_absolute_url(self) -> str:
        """Hazardous Material Absolute URL

        Returns:
            str: Hazardous Material Absolute URL
        """
        return reverse("order:hazardousmaterial_detail", kwargs={"pk": self.pk})


class Commodity(GenericModel):  # type: ignore
    """A class representing a commodity.

    This class inherits from the `GenericModel` class and defines several fields that are used to store
    information about a commodity. It also contains a nested `UnitOfMeasureChoices` class that defines
    the possible unit of measure choices for the `Commodity` model.

    Attributes:
        id: A UUIDField that represents the unique identifier of a commodity.
        name: A CharField that stores the name of a commodity.
        description: A TextField that stores the description of a commodity.
        min_temp: A DecimalField that stores the minimum temperature of a commodity.
        max_temp: A DecimalField that stores the maximum temperature of a commodity.
        set_point_temp: A DecimalField that stores the set point temperature of a commodity.
        unit_of_measure: A ChoiceField that stores the unit of measure of a commodity.
        hazmat: A ForeignKey that links a commodity to its hazardous material.
        is_hazmat: A BooleanField that indicates whether a commodity is hazardous.
    """

    @final
    class UnitOfMeasureChoices(models.TextChoices):
        """A class representing the possible unit of measure choices.

        This class inherits from the `models.TextChoices` class and defines several constants
        representing the different units of measure that can be used in the `Commodity` model.
        """

        # Constants representing the different units of measure
        PALLET = "PALLET", _("Pallet")
        TOTE = "TOTE", _("Tote")
        DRUM = "DRUM", _("Drum")
        CYLINDER = "CYLINDER", _("Cylinder")
        CASE = "CASE", _("Case")
        AMPULE = "AMPULE", _("Ampule")
        BAG = "BAG", _("Bag")
        BOTTLE = "BOTTLE", _("Bottle")
        PAIL = "PAIL", _("Pail")
        PIECES = "PIECES", _("Pieces")
        ISO_TANK = "ISO_TANK", _("ISO Tank")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    name = models.CharField(
        _("Name"),
        max_length=255,
        help_text=_("Name of the Commodity"),
    )
    description = models.TextField(
        _("Description"),
        blank=True,
        help_text=_("Description of the Commodity"),
    )
    min_temp = models.DecimalField(
        _("Minimum Temperature"),
        max_digits=10,
        decimal_places=2,
        help_text=_("Minimum Temperature of the Commodity"),
        null=True,
        blank=True,
    )
    max_temp = models.DecimalField(
        _("Maximum Temperature"),
        max_digits=10,
        decimal_places=2,
        help_text=_("Maximum Temperature of the Commodity"),
        null=True,
        blank=True,
    )
    set_point_temp = models.DecimalField(
        _("Set Point Temperature"),
        max_digits=10,
        decimal_places=2,
        help_text=_("Set Point Temperature of the Commodity"),
        null=True,
        blank=True,
    )
    unit_of_measure = ChoiceField(
        _("Unit of Measure"),
        choices=UnitOfMeasureChoices.choices,
        help_text=_("Unit of Measure of the Commodity"),
        blank=True,
    )
    hazmat = models.ForeignKey(
        "commodities.HazardousMaterial",
        on_delete=models.PROTECT,
        verbose_name=_("Hazardous Material"),
        help_text=_("Hazardous Material of the Commodity"),
        null=True,
        blank=True,
    )
    is_hazmat = models.BooleanField(
        _("Is Hazardous Material"),
        default=False,
        help_text=_("Is the Commodity a Hazardous Material"),
    )

    class Meta:
        """
        Commodity Metaclass
        """

        verbose_name = _("Commodity")
        verbose_name_plural = _("Commodities")
        ordering = ["name"]
        db_table = "commodity"
        constraints = [
            models.UniqueConstraint(
                fields=["name", "organization"],
                name="unique_commodity_name_organization",
            )
        ]

    def __str__(self) -> str:
        """Commodity String Representation

        Returns:
            str: Commodity Name
        """
        return textwrap.wrap(self.name, 50)[0]

    def update_commodity(self, **kwargs: Any) -> None:
        """Update Commodity

        Args:
            **kwargs: Keyword arguments that are used to update the commodity.
        """

        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()

    def save(self, **kwargs: Any) -> None:
        """Save Commodity

        Args:
            **kwargs (Any): Keyword Arguments

        Returns:
            None
        """

        self.full_clean()

        if self.hazmat:
            self.is_hazmat = True
        super().save(**kwargs)

    def get_absolute_url(self) -> str:
        """Commodity Absolute URL

        Returns:
            str: Commodity Absolute URL
        """
        return reverse("commodity:commodity_detail", kwargs={"pk": self.pk})
