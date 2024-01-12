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

import typing

from django_filters import FilterSet
from graphene import Field, ObjectType, Schema, String, relay
from graphene_django import DjangoObjectType
from graphene_django.filter import DjangoFilterConnectionField

from accounts import models

if typing.TYPE_CHECKING:
    from django.db.models import QuerySet
    from graphql import GraphQLResolveInfo


class JobTitleFilterSet(FilterSet):
    """
    FilterSet for JobTitle
    """

    class Meta:
        model = models.JobTitle
        fields = ["name"]


class JobTitleNode(DjangoObjectType):
    """
    Job Title Node for GraphQL
    """

    class Meta:
        model = models.JobTitle
        interfaces = (relay.Node,)
        filterset_class = JobTitleFilterSet
        exclude = ["profile"]

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.JobTitle]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.JobTitle]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class UserProfileFilterSet(FilterSet):
    """
    FilterSet for UserProfile
    """

    class Meta:
        model = models.UserProfile
        fields = ["first_name", "last_name"]


class UserProfileNode(DjangoObjectType):
    """
    User Profile Node for GraphQL
    """

    job_title = Field(JobTitleNode)

    class Meta:
        model = models.UserProfile
        interfaces = (relay.Node,)
        fiterset_class = UserProfileFilterSet
        exclude = ["user"]

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.UserProfile]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.UserProfile]":
        return queryset.filter(organization_id=info.context.user.organization_id)

    def resolve_job_title(self, info: "GraphQLResolveInfo") -> models.JobTitle:
        return self.job_title


class UserFilterSet(FilterSet):
    """
    FilterSet for User
    """

    class Meta:
        model = models.User
        fields = ["is_active", "department", "is_staff", "username"]


class UserNode(DjangoObjectType):
    """
    User Node for GraphQL
    """

    profile = Field(UserProfileNode)
    full_name = String()

    class Meta:
        model = models.User
        interfaces = (relay.Node,)
        filterset_class = UserFilterSet
        fields = [
            "id",
            "username",
            "is_active",
            "department",
            "is_staff",
            "email",
            "profile",
            "date_joined",
            "timezone",
            "session_key",
            "last_login",
        ]

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.User]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.User]":
        return queryset.filter(
            organization_id=info.context.user.organization_id
        ).select_related("profiles", "profiles__job_title")

    def resolve_full_name(self, info: "GraphQLResolveInfo") -> str:
        return f"{self.profile.first_name} {self.profile.last_name}"

    def resolve_profile(self, info: "GraphQLResolveInfo") -> models.UserProfile:
        return self.profile


class Query(ObjectType):
    """
    The Query class defines the GraphQL queries that can be made to the server
    """

    user = relay.Node.Field(UserNode)
    users = DjangoFilterConnectionField(UserNode)
    job_title = relay.Node.Field(JobTitleNode)
    job_titles = DjangoFilterConnectionField(JobTitleNode)


schema = Schema(query=Query)
