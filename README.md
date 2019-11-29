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

    You can also develop your own IDPs locally and use the `--local-repo` flag with udo to try it out
    eg. `udo create spring-dev-pack-build-tasks --local-repo /Users/maysun/dev/redhat/idp/spring-idp/index.json`

2. URL create
    `udo url create <ingress domain> --port 8080`
    eg.
    `udo url create myapp.<IP>.nip.io --port 8080`
    
3. Push    
    `udo push`

    Note: `udo push` can also be used for updates. To force a full build use `udo push --fullBuild`
