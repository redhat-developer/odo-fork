## UDO POC

The current POC requires the use of a local IDP repository using the `local-repo` flag.

Current limitations:
- Spring project types only
- A RWX volume is currently required. We're working on 

Try out the POC with the following steps:

Change directory to a project root directory and run the following:
1. Create
    `udo create spring`
    or
    `udo create spring-buildtasks`

2. URL create
    `udo url create <ingress domain> --port 9080`
    eg.
    `./udo url create myapp.<IP>.nip.io --port 9080`
    
3. Push    
    `./udo push`
