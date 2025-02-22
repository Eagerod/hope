# Home OPErations

>> Hope is not a strategy
>>
>> -- Unknown
>
> But it can be your cluster management command line tool!
>
> -- Me

Hope is a command line tool to manage small (and generally static) Kubernetes clusters.
Hope offers tooling to manage the virtual machines that Kubernetes nodes run on, as well as managing their membership in the cluster.
It also enables managing the resources in the cluster, with some simple parameter expansion that can be managed through a yaml file.

## Topology Resources

Hope provides a somewhat pluggable interface for managing different hypervisors.
Although pluggable, the only system implemented today is VMWare 6.7.

With these hypervisors, VMs can be created and destroyed, and generally be managed up to the point where they can be SSHed into.

Once SSH is available, VMs can be configured and added to clusters, or used to create fresh clusters.

## Cluster Resources

Hope offers a simple templatable YAML structure that allows for different resources within the cluster to be managed.

Resources that can be managed include:
- Kubernetes YAML resources, either by filename, or by inline YAML content within the configuration file.
- The building and pushing of Docker images to a registry.
- Tasks that attach to running jobs within the cluster, and wait for completion.
- Tasks that execute commands within already running pods within the cluster.

## Development

Hope uses Go 1.23, and requires staticcheck to be installed to run its linting/validation recipes:

```
go install honnef.co/go/tools/cmd/staticcheck@v0.6.0
```
