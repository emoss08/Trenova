# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
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

from django.contrib.auth.mixins import LoginRequiredMixin
from graphene import ObjectType, Schema
from graphene_django.views import GraphQLView

import accounting.schema
import accounts.schema
import billing.schema
import commodities.schema
import customer.schema
import dispatch.schema
import equipment.schema
import shipment.schema


class PrivateGraphQLView(LoginRequiredMixin, GraphQLView):
    pass


class Query(
    dispatch.schema.Query,
    shipment.schema.Query,
    accounting.schema.Query,
    accounts.schema.Query,
    billing.schema.Query,
    commodities.schema.Query,
    customer.schema.Query,
    equipment.schema.Query,
    ObjectType,
):
    """
    The Query class defines the GraphQL queries that can be made to the server
    """


schema = Schema(query=Query)
