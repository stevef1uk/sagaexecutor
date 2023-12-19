This project has been created to demonstrate the use of Dapr Building Blocks There is a solution architecture picture and description in the README.pdf file.
I have performed basic testing and it seems to work as I expect, however, I have not performed extensive load testing and on my litte cluster it is easy to max-out the message queues.
An enhancemnt would be to run one one Subscriber per Go client service deployed.

There is actually very little code required:
```
% gocloc *          
-------------------------------------------------------------------------------
Language                     files          blank        comment           code
-------------------------------------------------------------------------------
Go                               8            144             61            508
YAML                            10              7              0            278
Makefile                         4              1              0             25
-------------------------------------------------------------------------------
TOTAL                           22            152             61            811
-------------------------------------------------------------------------------
```

To get started with running this proejct, there are some prerequisites:

1. A kubernetes cluster is required
2. Redis & Postgres must be installed on the cluster
3. Tilt is is used to deply the components (see: https://tilt.dev). However, manual deployment is possible.

I used a personal hosted k3s cluster running on RPi4s, with k3s depolyed, this seesm fairly solid but a Cloud SaaS version is expected to be used for real use cases of this software.

To install Postgres on my home cluster I used the Postgres Operator, which configures a HA set-up by default. See:  https://github.com/zalando/postgres-operator/tree/master

As I am using an arm system I needed to change the image being deployed: Change: image: registry.opensource.zalan.do/acid/postgres-operator:v1.10.1 in manifests/postgres-operator.yaml to: ghcr.io/zalando/postgres-operator:v1.10.1

Then I created a DB for this project, which I called hasura - on mac/Linux):
```
  export POSTGRES=$(kubectl get secret postgres.acid-minimal-cluster.credentials.postgresql.acid.zalan.do -n postgres -o 'jsonpath={.data.password}' | base64 -d)
  kubectl port-forward acid-minimal-cluster-0 -n postgres 5432:5432
  psql --host localhost --username postgres
  create database hasura with owner postgres;
```
The postgres password is required to create a kubernetes secret as the deploymnet manifests expect this e.g
```
create secret generic postgres-url --from-literal="postgres-url=postgresql://postgres:$POSTGRES@acid-minimal-cluster.postgres.svc.cluster.local:5432/hasura"
```
To install Redis I used this Helm script: 
```
helm install my-release oci://registry-1.docker.io/bitnamicharts/redis
export REDIS_PASSWORD=$(kubectl get secret --namespace default my-release-redis -o jsonpath="{.data.redis-password}" | base64 -d)
kubectl create secret generic redis --from-literal="redis-password=$REDIS_PASSWORD"
```
The structure of the projects is:
```
components
cmd 
    poller
    subscriber
database
service
test_clients
    mock_server
    mock_client
```
Before running the core Subscriber & Postgres componnets the config files in components need to be applied to the cluster e.g
```
kubectl create -f components
```
(the following files need to be used: : cron.yaml, observability.yaml, statestore.yaml & pubsub.yaml)

First deploy & run the Subscriber & Poller components (tilt up or kubectl create -f deploymnets/kubernetes.yaml)

Then the test clinets can be run (mock_server & mock_client) to demonstrate (or see) if it is working (again tilt up)


    








