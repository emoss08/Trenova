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

import time

from kubernetes import client, config, watch
from kubernetes.client import Configuration
from pick import pick


def delete_pod(api_instance: client, namespace: str, pod_name: str) -> None:
    try:
        api_instance.delete_namespaced_pod(
            name=pod_name, namespace=namespace, body=client.V1DeleteOptions()
        )
        print(f"Pod {pod_name} in namespace {namespace} has been deleted")
    except client.exceptions.ApiException as e:
        print(f"Exception when calling CoreV1Api->delete_namespaced_pod: {e}")


def wait_for_pod_recreation(
    api_instance: client,
    namespace: str,
    original_pod_name: str,
    original_pod_labels: str,
) -> None:
    print(f"Waiting for pod {original_pod_name} to be recreated...")
    while True:
        pods = api_instance.list_namespaced_pod(namespace)
        for pod in pods.items:
            if (
                pod.metadata.labels == original_pod_labels
                and pod.metadata.name != original_pod_name
                and pod.status.phase == "Running"
            ):
                print(
                    f"Pod {pod.metadata.name} in namespace {namespace} has been recreated and is running"
                )
                print(f"Pod IP: {pod.status.pod_ip}")
                return
        time.sleep(5)


def labels_to_selector(labels):
    label_list = [f"{key}={value}" for key, value in labels.items()]
    return ",".join(label_list)


def main() -> None:
    kubeconfig_path = "updated_kubeconfig.yaml"  # Path to the updated kubeconfig file
    contexts, active_context = config.list_kube_config_contexts(
        config_file=kubeconfig_path
    )
    if not contexts:
        print("Cannot find any context in kube-config file.")
        return
    contexts = [context["name"] for context in contexts]
    active_index: int = contexts.index(active_context["name"])
    option, _ = pick(
        contexts, title="Pick the context to load", default_index=active_index
    )
    config.load_kube_config(context=option, config_file=kubeconfig_path)
    configuration = Configuration()
    configuration.verify_ssl = False
    configuration.host = "http://localhost:8080"
    configuration.DEBUG = True
    Configuration.set_default(configuration)

    print(f"Active host is {configuration.host}")
    v1 = client.CoreV1Api(api_client=client.ApiClient(configuration))

    watch.Watch()

    print("Listing pods with their IPs:")
    ret = v1.list_pod_for_all_namespaces(watch=False)
    pod_list = [(i.metadata.name, i.metadata.namespace) for i in ret.items]
    pod_to_restart, _ = pick(pod_list, title="Select the pod to restart")  # type: ignore
    pod_name, namespace = pod_to_restart

    original_pod = v1.read_namespaced_pod(namespace=namespace, name=pod_name)
    original_pod_labels = original_pod.metadata.labels

    delete_pod(v1, namespace, pod_name)
    wait_for_pod_recreation(v1, namespace, pod_name, original_pod_labels)

    recreated_pod_name = ""
    while not recreated_pod_name:
        try:
            label_selector = labels_to_selector(original_pod_labels)
            recreated_pod = v1.list_namespaced_pod(
                namespace, label_selector=label_selector
            )
            recreated_pod_name = recreated_pod.items[0].metadata.name
        except IndexError:
            continue
    # watch_events(v1, namespace, recreated_pod_name, log_file="my-pod-events.txt")
    # stream_logs(v1, "default", "nginx-748c667d99-6hc75", log_file="my-pod-logs.txt")


if __name__ == "__main__":
    main()
