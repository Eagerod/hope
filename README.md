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

## Development

Hope uses Go 1.19, and requires staticcheck to be installed to run its linting/validation recipes:

```
go install honnef.co/go/tools/cmd/staticcheck@v0.4.2
```
