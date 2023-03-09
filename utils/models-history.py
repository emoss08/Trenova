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

# PREVIOUSLY:

#
# class AutoSelectRelatedQuerySetMixin:
#     """Mixin for automatically selecting related and prefetching objects in a queryset"""
#
#     RELATED_FIELDS_TO_INCLUDE = {
#         "OneToOneField": "__all__",
#         "ForeignKey": "__all__",
#         "ManyToManyField": "__all__",
#     }
#
#     def get_queryset(self):
#         queryset = super().get_queryset()
#
#         # Get the model associated with the queryset
#         model = queryset.model
#
#         # Get the list of fields being requested in the queryset
#         fields = set(queryset.only_fields)
#
#         # Build up a dictionary of related objects to include in select_related and their respective fields
#         related_objects = {}
#         for field in fields:
#             if field != "id":
#                 # Split the field name into its components
#                 components = field.split("__")
#
#                 # Iterate through the components to determine which related objects to include
#                 for i in range(len(components) - 1):
#                     related_field_name = "__".join(components[:i + 1])
#                     related_field_class = model._meta.get_field(related_field_name).__class__.__name__
#                     if related_field_class in self.RELATED_FIELDS_TO_INCLUDE:
#                         if related_field_name not in related_objects:
#                             related_objects[related_field_name] = set()
#                         related_objects[related_field_name].add(components[i + 1])
#
#         # Build up a list of select_related arguments from the related_objects dictionary
#         select_related_args = []
#         for related_object, related_fields in related_objects.items():
#             select_related_args.append(related_object)
#
#         # Call select_related with the related objects and fields from the related_objects dictionary
#         queryset = queryset.select_related(*select_related_args).only(
#             *fields
#         )
#
#         # Determine which related objects to prefetch based on the related_objects dictionary
#         prefetch_objects = []
#         for related_object, related_fields in related_objects.items():
#             if len(related_fields) > 1:
#                 prefetch_objects.append(
#                     Prefetch(
#                         related_object,
#                         queryset=model._meta.get_field(related_object).remote_field.model.objects.only(
#                             *related_fields
#                         )
#                     )
#                 )
#
#         queryset = queryset.prefetch_related(*prefetch_objects)
#
#         return queryset
