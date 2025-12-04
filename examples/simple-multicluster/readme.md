# simple multicluster

This example shows how to use mermaid-kube-live with multiple clusters.

## Pre-requisites

- kind
- kubectl
- go

## Setup

Start two kind clusters:

    kind create cluster --name cluster1 --kubeconfig ./kubeconfig.yaml
    kind create cluster --name cluster2 --kubeconfig ./kubeconfig.yaml

Next start mermaid-kube-live:

    go run ../../cmd/mermaid-kube-live serve \
        --config ./mkl.yaml \
        --diagram ./mkl.mermaid \
        --kubeconfig ./kubeconfig.yaml

Then open the browser at http://localhost:8080 and you should see the
diagram with three clusters - ignore the third one for now:

![Image](https://github.com/user-attachments/assets/f3fd073a-015f-4aee-ac60-a08cc878be83)

To simulate multicluster interactivity run the script
`copy-resources.sh` in a new terminal:

    ./copy-resources.sh ./kubeconfig.yaml kind-cluster1 kind-cluster2 kind-cluster3

The script sleeps for 5 seconds between iterations and between clusters
to make it easier to see the changes in the browser.

## Creating resources

Now create a secret in cluster1. After the secret is created it will
turn green in the diagram.

    KUBECONFIG=./kubeconfig.yaml kubectl --context kind-cluster1 create secret generic our-first-secret

![Image](https://github.com/user-attachments/assets/61ea6c66-faee-4134-9543-29d49c18b0e1)

The script will copy the seret to cluster2, after which it will turn
green in the second cluster as well:

![Image](https://github.com/user-attachments/assets/b8c9882f-963e-46c4-8679-de069d7650d7)

Now create a new kind cluster:

    kind create cluster --name cluster3 --kubeconfig ./kubeconfig.yaml

Kind will start the cluster cluster and add it to the kubeconfig file.
The script copying the resources already knows about this cluster and
will automatically copy the secret to it on the next iteration.

mermaid-kube-live has been configured to watch for this cluster as well,
so it will automatically detect the new cluster in the kubeconfig and
update its node in the diagram:

![Image](https://github.com/user-attachments/assets/f2f05b1c-b6fd-4336-8299-b2bce4af32d6)

## Cleanup

Delete the clusters:

    kind delete cluster --name cluster1
    kind delete cluster --name cluster2
    kind delete cluster --name cluster3

And the kubeconfig file:

    rm ./kubeconfig.yaml
