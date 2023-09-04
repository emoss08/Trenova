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
import typing

from rest_framework import status
from rest_framework.response import Response
from rest_framework.views import APIView

if typing.TYPE_CHECKING:
    from rest_framework.request import Request


class ValidateView(APIView):
    """A basic class to validate different steps of a multistep form/model using Django REST framework's APIView.

    The main responsibility of this class is to take in the provided request data and perform validation checks
    based on the specified step. The class could contain custom step validation methods, which will be used for complex validation.

    Each view using this class is expected to define the `serializer_class` and `step_to_fields` class variables.

    Attributes:
        serializer_class (Serializer): A Django REST framework Serializer class that is used for data validation.
        step_to_fields (Dict[str, List[str]]): A dictionary mapping each step to its relevant fields.

    Methods:
        post: Standard HTTP POST method. Perform validation depending upon the step provided in request data.
        validate_custom_step: Example method for a custom step called "custom_step".
            Can be used as a template for other custom step validation methods.
    """

    serializer_class = None
    step_to_fields: dict[str, list[str]] = None

    def post(self, request: "Request") -> "Response":
        """Handle HTTP POST request.

        Validates the data in the request based on the step provided in request data.
        The step is used as a key in `step_to_fields` to get the relevant fields for validation.
        If a method `validate_{step}` exists, it is called for custom validation logic.

        Args:
            request (Request): The request object.

        Returns:
            Response: A response object indicating whether the validation succeeded or failed,
                      and containing any error information in case of failure.

        Raises:
            NotImplementedError: If `serializer_class` or `step_to_fields` is not defined.
        """

        # Check if the required class variables are defined
        if not self.serializer_class:
            raise NotImplementedError(
                f"{self.__class__.__name__} must define serializer_class"
            )
        if not self.step_to_fields:
            raise NotImplementedError(
                f"{self.__class__.__name__} must define step_to_fields"
            )

        step = request.data.get("step")  # The frontend should send the step name
        if not step or step not in self.step_to_fields:
            return Response(
                {"valid": False, "errors": "Invalid step"},
                status=status.HTTP_400_BAD_REQUEST,
            )

        # Only validate the fields relevant to the step
        relevant_fields = self.step_to_fields[step]
        partial_data = {field: request.data.get(field) for field in relevant_fields}

        # Custom logic for steps that require more than simple field validation
        if hasattr(self, f"validate_{step}"):
            custom_validate = getattr(self, f"validate_{step}")
            response = custom_validate(request, partial_data)
            if response:
                return response

        serializer = self.serializer_class(data=partial_data)
        if serializer.is_valid():
            return Response({"valid": True}, status=status.HTTP_200_OK)

        return Response(
            {"valid": False, "errors": serializer.errors},
            status=status.HTTP_400_BAD_REQUEST,
        )

    def validate_custom_step(
        self, request: "Request", partial_data: dict
    ) -> None:
        """Example method for a custom step called "custom_step".

        This is a template for methods that can be used for custom validation of steps.
        To use, define a new method in the same format for the step that requires custom validation.

        Args:
            request (Request): The request object.
            partial_data (Dict): The relevant fields and their values for this step.

        Returns:
            None: Always returns None, actual response should be returned in real implementations.
        """
        # Implement any custom validation logic here
        pass  # Example: return Response(...)
