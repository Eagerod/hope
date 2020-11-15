# Home OPErations

>> Hope is not a strategy
>>
>> -- Unknown
>
> But it can be your cluster management command line tool!
>
> -- Me

Hope is my solution to my 1000 line Makefile that ended up being written in my [home-network](https://github.com/eagerod/home-network) repo.
Since there's very little value to having everything wrapped up into a Makefile, considering nearly nothing is _made_, I decided to start the migration into a more robust tool.

This repo exists for a few different reasons:
- Clean up the massive Makefile I used to have
- Experiment more with Golang, especially with passing around stdin/out between subprocesses
- Eliminate other odd scripts that posed minimal value, but were tons of code to deal with
- Allow me to just pull this binary, and a couple credentials, and have a fully functioning management layer for my network
- Have a single binary I could bake into a CI pipeline that could be used to automate even more
- Keep things consolidated.

# Procedure to set up a freshly deployed node
Note: At some point, this process will include actually deploying a node to ESXi, but for now it makes the assumption that a node has been started on ESXi.

Make sure the host has enough memory and CPU.
Kubernetes does require a minimum of 2 CPUs.

Add the host to the appropriate place in `hope.yaml`.

Set up passwordless SSH, so that all the remaining commands are a lot less cumbersome to run:

    hope ssh <host>

Set its hostname, so when it's added to the cluster:

    hope hostname <host> <new-hostname>

Initialize the node.
Since it's already (hopefully) in the right section, it will initialize using the right procedure.

    hope init <host>
