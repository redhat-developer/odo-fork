## UDO POC

### Current limitations:
- Spring project types only
- A RWX PV is currently required. We're working on RWO

The poc uses a sample repository of IDPs: https://github.com/maysunfaisal/iterative-dev-packs

### What the POC contains
1. Catalog  
   `udo catalog list idp`  

2. Create  
    `udo create <IDP name>`  

3. URL create  
    `udo url create myapp.<ingress domain> --port <port>`  
    
4. Push  
    `udo push --fullBuild`  

    Note: Use `udo push` (without the `--fullBuild`) for updates. `--fullBuild` is a temporary flag for the POC, an actual implementation would have built-in smarts to determine when a full build or update is required.

5. Delete
   `udo delete`  

### Developing IDPs  

You can develop your own IDPs locally using the `--local-repo` flag with udo.

1. Clone https://github.com/maysunfaisal/iterative-dev-packs  
2. Use the local version of the IDP  
   `udo create spring-dev-pack-build-tasks --local-repo /Users/maysun/dev/redhat/idp/spring-idp/index.json`  

### Try out the POC with these samples:  

#### Spring

1. Clone  
   https://github.com/spring-projects/spring-petclinic

2. Create
   - `udo create spring`  
   - `udo url create <ingress domain> --port 8080`  
   - `udo push --fullBuild`  

3. Update
   - `udo push`  

#### Microprofile

1. Clone  
   https://github.com/rajivnathan/microproj

2. Create  
   - `udo create microprofile`  
   - `udo url create <ingress domain> --port 9080`  
   - `udo push --fullBuild`  

3. Update
   - `udo push`  

