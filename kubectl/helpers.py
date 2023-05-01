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
from typing import Any

from django.core.exceptions import ValidationError
from kubernetes import client

from kubectl.selectors import get_kube_config_by_organization
from organization.models import Organization


def get_node_info(*, node: client.V1Node) -> dict[str, Any]:
    """Returns a dictionary containing information about a Kubernetes node.

    Args:
        node: A `V1Node` instance representing the node for which information is to be extracted. Should be of
            type `client.V1Node`.

    Returns:
        A dictionary containing information about the specified node, including its CPU and memory usage and
        allocatable resources, as well as its status, type, reason, message, last heartbeat time, last transition
        time, machine ID, system UUID, boot ID, kernel version, operating system, architecture, and versions of
        container runtime, Kubelet, and Kube proxy. The dictionary should be of type `Dict[str, Any]`.
    """

    return {
        "cpu_usage": node.status.capacity["cpu"],
        "memory_usage": node.status.capacity["memory"],
        "pod_usage": node.status.capacity["pods"],
        "cpu_allocatable": node.status.allocatable["cpu"],
        "memory_allocatable": node.status.allocatable["memory"],
        "pod_allocatable": node.status.allocatable["pods"],
        "status": node.status.conditions[0].status,
        "type": node.status.conditions[0].type,
        "reason": node.status.conditions[0].reason,
        "message": node.status.conditions[0].message,
        "last_heartbeat_time": node.status.conditions[0].last_heartbeat_time,
        "last_transition_time": node.status.conditions[0].last_transition_time,
        "machine_id": node.status.node_info.machine_id,
        "system_uuid": node.status.node_info.system_uuid,
        "boot_id": node.status.node_info.boot_id,
        "kernel_version": node.status.node_info.kernel_version,
        "os_image": node.status.node_info.os_image,
        "container_runtime_version": node.status.node_info.container_runtime_version,
        "kubelet_version": node.status.node_info.kubelet_version,
        "kube_proxy_version": node.status.node_info.kube_proxy_version,
        "operating_system": node.status.node_info.operating_system,
        "architecture": node.status.node_info.architecture,
    }


def get_node_metadata(*, node: client.V1Node) -> dict[str, Any]:
    """Returns a dictionary of metadata for a Kubernetes node.

    Args:
        node: A `V1Node` instance representing the node for which metadata is to be extracted. Should be of type
            `client.V1Node`.

    Returns:
        A dictionary containing metadata for the specified node, including its name, namespace, self-link, UID,
        resource version, generation, creation timestamp, deletion timestamp, deletion grace period in seconds,
        labels, annotations, owner references, and finalizers. The dictionary should be of type `Dict[str, Any]`.
    """

    return {
        "name": node.metadata.name,
        "namespace": node.metadata.namespace,
        "self_link": node.metadata.self_link,
        "uid": node.metadata.uid,
        "resource_version": node.metadata.resource_version,
        "generation": node.metadata.generation,
        "creation_timestamp": node.metadata.creation_timestamp,
        "deletion_timestamp": node.metadata.deletion_timestamp,
        "deletion_grace_period_seconds": node.metadata.deletion_grace_period_seconds,
        "labels": node.metadata.labels,
        "annotations": node.metadata.annotations,
        "owner_references": node.metadata.owner_references,
        "finalizers": node.metadata.finalizers,
    }


def organization_kube_api_client(*, organization: Organization) -> client.CoreV1Api:
    """Returns a Kubernetes API client instance for the specified organization.

    Args:
        organization: An instance of the `Organization` model that identifies the organization whose Kubernetes
            API client instance is to be created. Should be of type `Organization`.

    Returns:
        A `CoreV1Api` instance that can be used to interact with the Kubernetes API for the specified organization.
        The instance should be of type `client.CoreV1Api`.

    """
    org_kube_config = get_kube_config_by_organization(organization=organization)

    if not org_kube_config:
        raise ValidationError("Organization does not have a Kubernetes configuration.")

    configuration = client.Configuration()

    # Set configuration based on the organization Kube Configuration
    configuration.verify_ssl = org_kube_config.verify_ssl
    configuration.host = org_kube_config.host
    configuration.DEBUG = org_kube_config.debug

    client.Configuration.set_default(configuration)
    return client.CoreV1Api(api_client=client.ApiClient(configuration))
