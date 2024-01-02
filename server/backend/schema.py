# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 MONTA                                                                         -
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
import typing

from django.contrib.auth.mixins import LoginRequiredMixin
from graphene import relay, ObjectType, Schema, Field
from graphene_django.filter import DjangoFilterConnectionField
from graphene_django.views import GraphQLView

from backend.graphql.types.dispatch_types import DelayCodeNode
from backend.graphql.types.shipment_types import ShipmentTypeNode, ShipmentControlNode
from shipment.models import ShipmentControl

if typing.TYPE_CHECKING:
    from graphql import GraphQLResolveInfo


class PrivateGraphQLView(LoginRequiredMixin, GraphQLView):
    pass


class Query(ObjectType):
    """
    The Query class defines the GraphQL queries that can be made to the server
    """

    delay_code = relay.Node.Field(DelayCodeNode)
    all_delay_codes = DjangoFilterConnectionField(DelayCodeNode)
    shipment_type = relay.Node.Field(ShipmentTypeNode)
    all_shipment_types = DjangoFilterConnectionField(ShipmentTypeNode)
    shipment_control = Field(ShipmentControlNode)

    def resolve_shipment_control(
        self, info: "GraphQLResolveInfo", **kwargs: typing.Any
    ):
        if not ShipmentControlNode.has_read_permission(info=info):
            raise PermissionError("You do not have permission to view ShipmentControl")

        return ShipmentControl.objects.get(organization=info.context.user.organization)


schema = Schema(query=Query)
