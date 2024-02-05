# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
import hashlib
import json
import logging
import typing

from django.conf import settings
from django.core.cache import caches
from django.http import HttpRequest, JsonResponse
from django.utils.deprecation import MiddlewareMixin
from redis import Redis, RedisError
from rest_framework import status

logger = logging.getLogger(__name__)
debug, info, warning, exception, error = (
    logger.debug,
    logger.info,
    logger.warning,
    logger.exception,
    logger.error,
)

EXCLUDED_METHODS = ["GET", "HEAD", "OPTIONS", "TRACE"]
EXCLUDED_PATHS = {
    "/api/login/",
    "/api/logout/",
    "/api/reset_password/",
    "/api/change_email/",
}
IDEMPOTENCY_KEY_TTL = settings.IDEMPOTENCY_KEY_TTL


def encode_key(key: str) -> str:
    """Encodes a given key using the SHA-256 hashing algorithm.

    This function is utilized to standardize the format of the idempotency keys, ensuring they are uniformly hashed.
    It takes a raw key as input, encodes it to bytes, applies the SHA-256 hash function, and then returns the
    hexadecimal representation of the hash.

    Args:
        key: The original string key to be encoded.

    Returns:
        str: The SHA-256 hashed representation of the input key as a hexadecimal string.
    """
    return hashlib.sha256(key.encode("utf-8")).hexdigest()


def generate_composite_key(request: HttpRequest) -> str:
    """Generates a composite idempotency key based on the user's identity, the request path, and the raw idempotency key.

    This function is designed to create a unique idempotency key for each operation by a user, ensuring that
    subsequent operations with the same parameters can be identified and handled idempotent. It constructs the key
    by concatenating the user's identifier (or 'anonymous' if not authenticated), the request path, and the raw
    idempotency key provided in the request's headers. The composite key is then hashed using SHA-256 to maintain
    a consistent and secure format.

    Args:
        request: The HttpRequest object, which provides the user identity, request path, and headers.

    Returns:
        str: A SHA-256 hashed string representing the composite idempotency key.

    Raises:
        ValueError: If the 'X-Idempotency-Key' header is missing in the request.
    """
    user_identifier = str(
        request.user.id if request.user.is_authenticated else "anonymous"
    )
    operation_identifier = request.path
    raw_key = request.headers.get("X-Idempotency-Key")
    if not raw_key:
        raise ValueError("X-Idempotency-Key header is missing.")
    composite_key = f"{user_identifier}-{operation_identifier}-{raw_key}"
    return encode_key(composite_key)


def serialize_response(response: JsonResponse) -> str:
    """Serializes a JsonResponse object into a JSON-formatted string.

    This function converts a JsonResponse object into a JSON-formatted string, ensuring that the response's content,
    status code, and headers are preserved and accurately represented in the serialized format. This serialized form
    can be used for storage or transmission and later reconstructed into its original form.

    Args:
        response: The JsonResponse object to serialize.

    Returns:
        str: A JSON-formatted string representing the serialized content, status, and headers of the response.
    """
    return json.dumps(
        {
            "content": response.content.decode("utf-8"),
            "status": response.status_code,
            "headers": dict(response.items()),
        }
    )


def deserialize_response(data: str) -> JsonResponse:
    """Deserializes a JSON-formatted string into a JsonResponse object.

    This function reconstructs a JsonResponse object from a JSON-formatted string, ensuring that the original content,
    status code, and headers of the response are accurately restored. It's primarily used to convert responses
    retrieved from storage or transmission back into their usable JsonResponse format.

    Args:
        data: The JSON-formatted string representing the serialized JsonResponse.

    Returns:
        JsonResponse: A JsonResponse object reconstructed from the serialized data.
    """
    loaded_data = json.loads(data)
    response = JsonResponse(
        json.loads(loaded_data["content"]), status=loaded_data["status"], safe=False
    )
    for key, value in loaded_data["headers"].items():
        response[key] = value
    return response


class RedisIdempotencyKeyLock:
    def __init__(self) -> None:
        """Initializes a lock mechanism to manage access to idempotency keys in Redis.

        This constructor method sets up a RedisIdempotencyKeyLock instance by connecting to the Redis server specified
        in the settings and creating a lock object. The lock is used to synchronize access to idempotency keys, ensuring
        that concurrent requests don't interfere with each other's idempotency checks and storage operations.

        Raises:
            ValueError: If the 'IDEMPOTENCY_LOCATION' setting is not properly configured in the settings file.
        """
        location = settings.IDEMPOTENCY_LOCATION
        if location is None or location == "":
            raise ValueError("IDEMPOTENCY_LOCATION must be set in the settings file")

        self.redis_obj = Redis.from_url(location)
        self.storage_lock = self.redis_obj.lock(
            name="idempotency_lock",
            timeout=300,  # 5 minutes
            blocking_timeout=0.1,  # 100ms
        )

    def acquire(self, *args: typing.Any, **kwargs: typing.Any) -> bool:
        """Attempts to acquire the lock.

        This method is used to obtain control over the idempotency key resource. If the lock is acquired successfully,
        it returns True, allowing the caller to proceed with operations that require exclusive access. If the lock
        cannot be acquired immediately, it returns False.

        Args:
            *args: Variable length argument list.
            **kwargs: Arbitrary keyword arguments.

        Returns:
            A boolean indicating whether the lock was successfully acquired.
        """
        return self.storage_lock.acquire()

    def release(self, *args: typing.Any, **kwargs: typing.Any) -> None:
        """Releases the lock.

        This method is called to release control over the idempotency key resource, allowing other operations to
        acquire the lock and proceed. It's crucial to ensure that the lock is released after the operation requiring
        exclusive access is completed.

        Args:
            *args: Variable length argument list.
            **kwargs: Arbitrary keyword arguments.

        Returns:
            None: This function does not return anything.
        """
        self.storage_lock.release()


class RedisIdempotencyKeyStorage:
    def __init__(self) -> None:
        """Initializes a storage object to manage idempotency keys using Redis.

        This constructor connects to a Redis instance specified in the settings. It's used to store and retrieve
        idempotency keys along with their corresponding responses, allowing for idempotent operations.

        Raises:
            ValueError: If the 'IDEMPOTENCY_LOCATION' is not properly configured in the settings.

        Returns:
            None: This function does not return anything.
        """
        if location := settings.IDEMPOTENCY_LOCATION:
            self.redis_obj = Redis.from_url(location)
        else:
            raise ValueError(
                "IDEMPOTENCY_LOCATION must be configured in the settings file"
            )

    @staticmethod
    def store_data(
        *, cache_name: str, encoded_key: str, response: JsonResponse
    ) -> None:
        """Stores the response data in Redis cache associated with the given idempotency key.

        The function serializes the response and stores it in the specified cache with a set expiration time. This
        allows subsequent requests with the same idempotency key to retrieve the stored response instead of
        re-executing the operation.

        Args:
            cache_name: The name of the cache to use.
            encoded_key: The idempotency key after encoding.
            response: The JsonResponse object to be stored.

        Returns:
            None: This function does not return anything.
        """
        str_response = serialize_response(response)
        caches[cache_name].set(encoded_key, str_response, timeout=IDEMPOTENCY_KEY_TTL)
        debug(f"Response stored in cache for idempotency key: {encoded_key}")

    @staticmethod
    def retrieve_data(
        *, cache_name: str, encoded_key: str
    ) -> tuple[bool, JsonResponse | None]:
        """Retrieves the stored response data from Redis cache associated with the given idempotency key.

        The function checks the cache for the given idempotency key and, if found, returns the stored response. If the
        key is not found, it returns None, indicating that the operation can proceed.

        Args:
            cache_name(str): The name of the cache to use.
            encoded_key(str): The idempotency key after encoding.

        Returns:
            tuple[bool, JsonResponse | None]: A tuple containing a boolean indicating whether the key was found and the
            deserialized JsonResponse object if the key was found, or None if not.
        """
        if encoded_key in caches[cache_name]:
            str_response = caches[cache_name].get(encoded_key)
            debug(f"Response retrieved from cache for idempotency key: {encoded_key}")
            return True, deserialize_response(str_response)

        debug(f"No cached response found for idempotency key: {encoded_key}")
        return False, None

    @staticmethod
    def validate_storage(name: str) -> bool:
        """Validates if the specified cache is properly configured and available.

        This function is used to ensure that the cache specified for storing idempotency keys is properly configured
        before attempting to store or retrieve data.

        Args:
            name: The name of the cache to validate.

        Returns:
            A boolean indicating whether the specified cache name is configured and available.

        """
        if name not in caches:
            error(f"Specified cache name '{name}' is not configured.")
            return False
        return True


class IdempotencyMiddleware(MiddlewareMixin):
    def __init__(self, get_response) -> None:
        """Initializes the IdempotencyMiddleware.

        The middleware is initialized with a storage mechanism for idempotency keys and a lock to handle concurrent
        requests. It ensures that requests with the same idempotency key do not perform the operation multiple times.

        Args:
            get_response: A callable which takes a request and returns a response.

        Returns:
            None: This function does not return anything.
        """
        self.storage = RedisIdempotencyKeyStorage()
        self.storage_lock = RedisIdempotencyKeyLock()
        super().__init__(get_response)

    @staticmethod
    def bad_request() -> JsonResponse:
        """Constructs a JsonResponse indicating a bad request error due to idempotency key issues.

        Returns:
            JsonResponse: A JsonResponse object with a status code of 400 (Bad Request) and a message indicating that
            the idempotency key validation failed.
        """
        return JsonResponse(
            {
                "message": "Idempotency key validation failed. Please provide a valid idempotency key in the request "
                "header.",
                "code": "idempotency_key_validation_failed",
            },
            status=status.HTTP_400_BAD_REQUEST,
        )

    @staticmethod
    def resource_locked() -> JsonResponse:
        """Constructs a JsonResponse indicating that the resource is temporarily locked due to concurrent access.

        Returns:
            JsonResponse: A JsonResponse object with a status code of 423 (Locked) and a message indicating that the
            resource is temporarily unavailable due to concurrent access.
        """
        return JsonResponse(
            {
                "message": "Resource temporarily unavailable due to concurrent access. Please try again later.",
                "code": "resource_locked",
            },
            status=status.HTTP_423_LOCKED,
        )

    def _reject(self, request: HttpRequest, reason: str) -> JsonResponse:
        """Handles the rejection of a request due to idempotency or other related issues.

        This method logs the rejection reason and constructs an appropriate JsonResponse to be returned to the client.

        Args:
            request: The HttpRequest that is being rejected.
            reason: A string indicating the reason for rejection.

        Returns:
            JsonResponse: A JsonResponse indicating a bad request or a locked resource, based on the reason for
            rejection.
        """
        response = self.bad_request()
        warning(
            f"Idempotency key validation failed for {request.method} {request.path}: {reason}"
        )
        return response

    def process_request(self, request: HttpRequest) -> JsonResponse | None:
        """Processes the incoming request, applying idempotency checks and handling.

        This method checks if the request path and method are excluded from idempotency checks. If not, it performs
        idempotency checks using the stored idempotency keys and handles the request accordingly.

        Args:
            request: The HttpRequest to be processed.

        Returns:
            JsonResponse | None: None if the request is to be allowed to proceed normally, or a JsonResponse if the
            request is rejected due to idempotency key issues or if the resource is locked.
        """
        if request.path in EXCLUDED_PATHS or request.method in EXCLUDED_METHODS:
            return None

        try:
            idempotency_key = generate_composite_key(request)
        except ValueError as e:
            return self._reject(request, str(e))

        if not self.storage_lock.acquire():
            return self.resource_locked()

        try:
            key_exists, response = self.storage.retrieve_data(
                cache_name=settings.IDEMPOTENCY_CACHE_NAME,
                encoded_key=idempotency_key,
            )
            if key_exists:
                info(f"Idempotent request for key: {idempotency_key}")
                return response

        except RedisError as redis_exc:
            exception(f"Redis error occurred: {redis_exc}")
            return self._reject(
                request, "Failed to process idempotency key due to a Redis error"
            )

        except Exception as exc:
            exception(f"Failed to retrieve idempotency key {idempotency_key}: {exc}")
            return self._reject(request, "Failed to retrieve idempotency key")

        finally:
            self.storage_lock.release()

    def process_response(
        self, request: HttpRequest, response: JsonResponse
    ) -> JsonResponse:
        """Processes the outgoing response, storing the response data for idempotent requests.

        Args:
            request: The HttpRequest that was processed.
            response: The JsonResponse generated from processing the request.

        Returns:
            JsonResponse: The unmodified JsonResponse if the request method is excluded from idempotency checks, or the
            JsonResponse after storing the response data for idempotent requests.

        This method checks if the request is subject to idempotency handling. If so, it stores the response data
        associated with the request's idempotency key for future requests with the same key.
        """
        if request.method in EXCLUDED_METHODS:
            return response

        idempotency_key = request.headers.get("X-Idempotency-Key")
        if not idempotency_key:
            return response

        if not self.storage_lock.acquire():
            return response

        try:
            self.storage.store_data(
                cache_name=settings.IDEMPOTENCY_CACHE_NAME,
                encoded_key=idempotency_key,
                response=response,
            )
            info(f"Stored response for idempotency key: {idempotency_key}")

        except Exception as exc:
            exception(
                f"Unexpected error occurred while storing idempotency key {idempotency_key}: {exc}"
            )

        finally:
            self.storage_lock.release()

        return response
