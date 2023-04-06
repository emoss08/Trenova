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

from kubernetes import client
from kubernetes.client import Configuration
from rest_framework.decorators import api_view
from rest_framework.request import Request
from rest_framework.response import Response
from silk.profiling.profiler import silk_profile

@silk_profile(name="Get Active Clusters")
@api_view(["GET"])
def get_active_clusters(request: Request) -> Response:
    configuration = Configuration()
    configuration.verify_ssl = False
    configuration.host = "http://localhost:8080"
    configuration.DEBUG = True
    Configuration.set_default(configuration)
    api = client.CoreV1Api(api_client=client.ApiClient(configuration))
    node = api.list_node()
    response = [
        {
            "name": node.metadata.name,
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
            "node_info": {
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
            },
            "metadata" : {
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
        }
        for node in node.items
    ]
    return Response(response)
