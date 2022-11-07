# -*- coding: utf-8 -*-
"""
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""
import textwrap
from typing import final

from django.conf import settings
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _
from localflavor.us.models import USStateField, USZipCodeField  # type: ignore

from control_file.models import CommentType
from core.models import GenericModel
from dispatch.models import DispatchControl
from organization.models import Depot

User = settings.AUTH_USER_MODEL


class Customer(GenericModel):
    """
    Stores customer information
    """
    is_active = models.BooleanField(
        _('Active'),
        default=True,
        help_text=_(
            'Designates whether this customer should be treated as active. '
            'Unselect this instead of deleting customers.'
        ),
    )
    code = models.CharField(
        _('Code'),
        max_length=10,
        unique=True,
        editable=False,
        primary_key=True,
        help_text=_('Customer code'),
    )
    name = models.CharField(
        _('Name'),
        max_length=150,
        help_text=_('Customer name'),
    )
    address_line_1 = models.CharField(
        _('Address Line 1'),
        max_length=150,
        help_text=_('Address line 1'),
    )
    address_line_2 = models.CharField(
        _('Address Line 2'),
        max_length=150,
        blank=True,
        help_text=_('Address line 2'),
    )
    city = models.CharField(
        _('City'),
        max_length=150,
        help_text=_('City'),
    )
    state = USStateField(
        _('State'),
        help_text=_('State'),
    )
    zip_code = USZipCodeField(
        _('Zip Code'),
        help_text=_('Zip code'),
    )

    class Meta:
        verbose_name = _('Customer')
        verbose_name_plural = _('Customers')
        ordering: list[str] = ['code']

    def __str__(self) -> str:
        """Customer string representation

        Returns:
            str: Customer string representation
        """
        return textwrap.wrap(f"{self.code} - {self.name}", 50)[0]

    def generate_customer_code(self) -> str:
        """Generate a unique code for the customer

        Returns:
            str: Customer code
        """
        code: str = self.name[:8].upper()
        new_code: str = f"{code}{Customer.objects.count()}"

        return code if not Customer.objects.filter(code=code).exists() else new_code

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular customer instance

        Returns:
            str: Customer url
        """
        return reverse('billing:customer-detail', kwargs={'pk': self.pk})


class CustomerBillingProfile(GenericModel):
    """
    Stores Billing Criteria related to the `Customer` model.
    """
    customer = models.OneToOneField(
        Customer,
        on_delete=models.CASCADE,
        related_name='billing_profile',
        related_query_name='billing_profiles',
        help_text=_('Customer'),
        verbose_name=_('Customer'),
    )
    is_active = models.BooleanField(
        _('Active'),
        default=True,
        help_text=_(
            'Designates whether this customer billing profile should be treated as active. '
            'Unselect this instead of deleting customer billing profiles.'
        ),
    )
    document_class = models.ManyToManyField(
        "DocumentClassification",
        related_name="billing_profiles",
        related_query_name="billing_profile",
        verbose_name=_('Document Class'),
        help_text=_('Document class'),
    )


class DocumentClassification(GenericModel):
    """
    Stores Document Classification information.
    """
    name = models.CharField(
        _('Name'),
        max_length=150,
        help_text=_('Document classification name'),
    )
    description = models.TextField(
        _('Description'),
        blank=True,
        help_text=_('Document classification description'),
    )

    class Meta:
        verbose_name = _('Document Classification')
        verbose_name_plural = _('Document Classifications')
        ordering: list[str] = ['name']

    def __str__(self) -> str:
        """Document classification string representation

        Returns:
            str: Document classification string representation
        """
        return textwrap.wrap(f"{self.name}", 50)[0]

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular document classification instance

        Returns:
            str: Document classification url
        """
        return reverse('billing:document-classification-detail', kwargs={'pk': self.pk})
