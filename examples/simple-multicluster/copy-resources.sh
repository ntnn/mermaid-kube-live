#!/usr/bin/env bash

export KUBECONFIG="$(realpath $1)"
shift 1

source_cluster="$1"
shift 1

target_clusters="$@"

echo "Starting to loop"
while sleep 5; do
    for target_cluster in $target_clusters; do
        echo "Copying secrets in default namespace from $source_cluster to $target_cluster"
        kubectl --context "$source_cluster" get secrets -o yaml | \
            kubectl --context "$target_cluster" apply -f -
        sleep 5
    done
done
