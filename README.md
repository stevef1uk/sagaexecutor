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
kubectl create -f components/.
```
(the following files need to be used: : cron.yaml, observability.yaml, statestore.yaml & pubsub.yaml)

First deploy & run the Subscriber & Poller components (tilt up and tilt down to undeploy)

Then the test clients can be run (mock_server & mock_client) to demonstrate (or see) if it is working (again tilt up)

If the mock_client is run the output should look like this:

```
apr client initializing for: 127.0.0.1:50001
2023/12/19 14:43:15 setting up handler
2023/12/19 14:43:15 About to send a couple of messages
2023/12/19 14:43:15 Sleeping for a bit
2023/12/19 14:43:20 Finished sleeping
2023/12/19 14:43:20 Successfully published first start message
2023/12/19 14:43:20 Successfully published first stop message
2023/12/19 14:43:20 Checking no records left
2023/12/19 14:43:20 Returned 0 records
2023/12/19 14:43:20 Sending a Start without a Stop & waiting for the call-back
2023/12/19 14:43:20 Successfully published second start message
2023/12/19 14:43:20 Returned 0 records
2023/12/19 14:43:20 Sleeping for a bit for the Poller to call us back
Yay callback invoked!
transaction callback invoked {mock-client test2 abcdefg1235 callback {"Param1":France} 30 false 2023-12-19 14:43:20 +0000 UTC}
2023/12/19 14:44:00 Sending a group of starts & stops
2023/12/19 14:44:01 Finished sending starts & stops
2023/12/19 14:44:01 Sleeping for quite a bit to allow time to receive any callbacks
```

What I have found is that if the system is loaded with too many messages unwanted call-backs occur. This needs some investigation, but it seems that the Dapr messaging has 'pauses' with the Redis back-end.

    








