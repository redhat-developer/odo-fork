## UDO POC

Current limitations:
- Spring project types only
- A RWX PV is currently required. We're working on RWO

The poc uses a sample repository of IDPs: https://github.com/maysunfaisal/iterative-dev-packs

Try out the POC with the following steps:

Change directory to a project root directory and run the following:
1. Create
    Using an s2i-like IDP
    `udo create spring`
    or
    Using a build task IDP
    `udo create spring-buildtasks`

2. URL create
    `udo url create <ingress domain> --port 8080`
    eg.
    `udo url create myapp.<IP>.nip.io --port 8080`
    
3. Push    
    `udo push`
