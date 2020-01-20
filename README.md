## UDO POC

### Current limitations:
- A RWX PV is currently required. We're working on RWO

### Prereqs
May require privileged and root containers depending on the selected IDP.

If your cluster is running OpenShift, run the following commands where <namespace> is the namespace you'll be using for the component:

    To enable privileged containers, enter oc adm policy add-scc-to-group privileged system:serviceaccounts:<namespace>.
    To enable containers to run as root, enter oc adm policy add-scc-to-group anyuid system:serviceaccounts:<namespace>.

### IDP repository
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

#### NodeJS

1. Clone  
   https://github.com/openshift/nodejs-ex

2. Create  
   - `udo create nodejs`  
   - `udo url create <ingress domain> --port 8080`  
   - `udo push --fullBuild`  

3. Update
   - `udo push`  
