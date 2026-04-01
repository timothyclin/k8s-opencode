# **Architectural Design for Multi-Tenant AI Coding Assistant Environments in Kubernetes**

## **Executive Summary**

The paradigm of software engineering is undergoing a fundamental transformation
driven by the integration of autonomous artificial intelligence agents directly
into the development workflow. As organizations transition from relying solely
on cloud-based Large Language Model (LLM) interfaces to utilizing deeply
integrated, agentic frameworks like OpenCode, the requirements for secure,
scalable, and isolated deployment environments have escalated significantly.1
OpenCode, a sophisticated terminal and web-based AI coding assistant, possesses
the capability to execute complex, multi-step operations across codebases,
utilizing specialized subagents such as the Build agent for file modifications
and system commands, and the Plan agent for read-only analytical tasks.1 The
deployment of such powerful tools within an enterprise demands strict isolation,
persistent state management, and seamless authentication mechanisms to ensure
that developers can leverage AI assistance without compromising data sovereignty
or cluster security.3

Historically, the architectural gold standard for provisioning multi-tenant,
interactive, and isolated computational workspaces has been JupyterHub.
JupyterHub pairs a centralized authentication interface with a dynamic container
spawning mechanism and an explicitly controlled reverse proxy, allowing it to
seamlessly map authenticated users to their uniquely provisioned backend
instances.5 However, replicating the JupyterHub architecture for arbitrary,
non-Jupyter workloads like OpenCode within a Kubernetes cluster introduces
profound architectural complexities. The primary challenge lies in the dynamic
routing of HTTP and WebSocket traffic. Conventional ingress controllers and
authentication middleware, such as the ubiquitous OAuth2-Proxy paired with
NGINX, are fundamentally designed for static upstream targeting. They excel at
validating OpenID Connect (OIDC) tokens from providers like Google Cloud
Platform (GCP) Identity or Google Workspace, but they inherently lack the
sophisticated internal mechanisms required to dynamically map an authenticated
user to a distinct, dynamically provisioned Kubernetes backend service on a
per-request basis.7

This comprehensive research report conducts an exhaustive technical analysis of
the mechanisms necessary to replicate the JupyterHub multi-user paradigm for
OpenCode instances deployed as Kubernetes StatefulSets. It deconstructs the
internal routing logic of JupyterHub's configurable-http-proxy, meticulously
evaluates the severe security and operational limitations of attempting to force
dynamic routing through standard ingress controllers utilizing Lua scripting,
and proposes two highly maintainable, enterprise-grade architectures. The
primary proposed solution leverages Authentik as an advanced Identity-Aware
Proxy (IAP) capable of executing secure, dynamic backend overrides based on the
real-time evaluation of user claims.9 The secondary architecture proposes a
Kubernetes-native, declarative GitOps pattern paired with Pomerium, effectively
shifting the complexity of dynamic routing away from the proxy layer and
entirely into the Kubernetes control plane.10 By implementing these
architectures, enterprises can achieve a secure, scalable, and fully isolated
multi-tenant AI coding environment that meets stringent security and operational
standards.

## **Deconstructing the JupyterHub Multi-Tenant Architecture**

To accurately replicate the functionality of JupyterHub's multi-user design for
an entirely different application stack, it is imperative to deeply understand
the complex interactions between its three core components: the central Hub, the
Spawner, and the Proxy.12 JupyterHub's success in managing multi-tenant
environments is not derived from a single monolithic application, but rather
from the orchestrated synchronization between these distinct processes.

### **The Mechanics of the Configurable HTTP Proxy**

Unlike traditional reverse proxies such as NGINX, HAProxy, or Envoy, which
typically rely on static configuration files (e.g., nginx.conf) or declarative
Kubernetes Custom Resource Definitions (CRDs) that must be loaded or watched by
a control loop, JupyterHub utilizes a bespoke, Node.js-based proxy known as
configurable-http-proxy (CHP).14 CHP operates as the primary data plane entry
point for all user traffic entering the JupyterHub ecosystem.

The architecture of CHP is uniquely suited for highly dynamic environments. It
concurrently runs two distinct HTTP servers. The first is a public-facing
interface, controlled by the \--ip and \--port parameters, which listens on all
external network interfaces by default and handles the actual routing of user
traffic to backend services.14 The second is an inward-facing REST API,
controlled by the \--api-ip and \--api-port parameters, which typically listens
exclusively on localhost or a highly restricted internal cluster IP.14 This REST
API is secured by a shared cryptographic token, the CONFIGPROXY\_AUTH\_TOKEN,
which is passed in the Authorization header of all administrative requests.14

When the configurable-http-proxy initializes, its internal routing table is
effectively empty, containing only a default target that points directly back to
the central JupyterHub process.14 Any request that does not match a specific
path prefix in the proxy's memory is automatically forwarded to the Hub,
allowing the Hub to handle the initial authentication flows and the management
of server lifecycles.16 The routing table itself is managed entirely in memory
via the REST API, enabling instantaneous updates without requiring a process
reload or dropping active WebSocket connections, a feature that is critical for
maintaining interactive coding sessions.

### **Control Plane Synchronization: Hub and KubeSpawner**

The lifecycle of a user session in JupyterHub demonstrates how dynamic routing
is achieved in practice. When an unauthenticated user attempts to access the
platform, CHP routes the request to the default target—the Hub.16 The Hub
intercepts the request and redirects the user to the configured Identity
Provider (IdP) for authentication, which could involve OAuth, LDAP, or PAM
integrations.12 Upon successful authentication, the Hub invokes a Spawner class
to provision the user's isolated environment.

In cloud-native environments, this is typically handled by the KubeSpawner, a
specialized Spawner that translates the user's request into a Kubernetes Pod
specification.18 The KubeSpawner is responsible for defining the entire
execution context of the user's container. It manages the scheduling of the Pod,
enforces resource guarantees and limits for CPU and RAM, and dynamically mounts
Persistent Volume Claims (PVCs) to ensure that the user's workspace data
survives Pod restarts.18 Furthermore, KubeSpawner allows administrators to
strictly control the security posture of the spawned containers, including the
configuration of Pod Security Policies, the manipulation of userid and groupid
parameters, and the disabling of potentially dangerous settings like privilege
escalation.18

Crucially, the interaction between KubeSpawner and the Hub does not end when the
Pod is scheduled. The Hub continuously monitors the status of the spawned Pod
via the Kubernetes API. The Spawner.start method polls the cluster until the Pod
achieves a Running state and the internal notebook server successfully binds to
its designated port.12 Once the backend service is fully operational, the Hub
executes a POST request to the CHP REST API, targeting the /api/routes
endpoint.20 This REST call dynamically injects a new route into the proxy's
in-memory routing table, mapping a specific URL path prefix (e.g., /user/alice/)
directly to the internal Kubernetes IP address and port of the newly spawned
Pod.21 From that moment forward, CHP routes all traffic destined for
/user/alice/ directly to Alice's dedicated container, bypassing the Hub entirely
for steady-state traffic.

### **The Limitations of Direct Replication for Arbitrary Workloads**

While the JupyterHub architecture is highly effective for its intended use case,
attempting to co-opt this exact mechanism for deploying arbitrary, non-Jupyter
workloads like OpenCode introduces severe architectural friction and operational
overhead.

It is technically possible to override the KubeSpawner configuration to spawn
non-Jupyter containers. Administrators can modify the c.KubeSpawner.cmd
parameter to override the default Docker image's entrypoint, preventing the
injection of the standard jupyterhub-singleuser process.19 Alternatively, tools
like jupyter-server-proxy can be utilized to wrap arbitrary web applications,
such as VS Code or Code-Server, allowing them to run alongside the Jupyter
server and proxying the traffic through the Jupyter interface.22

However, utilizing JupyterHub purely as a process manager for OpenCode is an
anti-pattern. This approach forces the deployment of the entire Jupyter Python
stack into environments where it serves no purpose, unnecessarily inflating
container image sizes and significantly expanding the attack surface.23
Furthermore, it tightly couples the infrastructure to JupyterHub's bespoke
configurable-http-proxy, forcing administrators to rely on a proxy that lacks
the broad ecosystem support, advanced traffic management capabilities, and
extensive observability integrations of modern, cloud-native ingress controllers
like NGINX, Traefik, or Envoy.23 A more elegant, maintainable approach involves
extracting the core _concept_ of dynamic user routing and implementing it using
modern identity providers and Kubernetes-native ingress controllers, completely
eliminating the need for a bespoke intermediate Hub process.

## **Architectural Requirements for Stateful AI Coding Assistants**

Deploying an AI coding assistant like OpenCode in a multi-tenant enterprise
environment requires an architecture tailored to the unique behavioral
characteristics and security requirements of autonomous agents. Unlike stateless
microservices, which can be easily horizontally scaled and destroyed without
consequence, developer workspaces demand strict isolation, deterministic
identity, and robust state management.

### **OpenCode Execution Mechanics and Security Implications**

OpenCode operates as a sophisticated AI coding agent that can function within a
terminal interface, a desktop application, or a web-based UI.1 It is designed to
interface directly with the developer's codebase, supporting multiple
programming languages and integrating with over 75 different LLM providers,
ranging from cloud-based models like OpenAI and Anthropic to locally hosted
models managed by the Docker Model Runner or Ollama.2

The architecture of OpenCode is inherently multi-agentic. It utilizes primary
agents, such as the Build agent, which possesses full access to file operations
and system commands for active development work, and the Plan agent, which
operates in a restricted mode for read-only analysis and architectural
planning.1 Furthermore, it relies on specialized subagents to handle complex,
multi-step tasks or rapid codebase exploration.1 Because OpenCode is designed to
actively edit files, execute bash scripts, and interact with the underlying
system environment to compile code and run tests, it must execute within a
highly restricted, heavily sandboxed execution context.1 In a multi-tenant
Kubernetes cluster, the security parameters dictate an absolute boundary between
users: User A's OpenCode instance must have zero network visibility into User
B's instance, and the underlying filesystem must be strictly partitioned to
prevent accidental or malicious cross-contamination of source code or API
credentials.

### **Kubernetes StatefulSets for Deterministic Network Identity**

The deployment of developer workspaces fundamentally differs from the deployment
of typical web services. Workspaces require stable network identities and
persistent storage that maps consistently to the same user across Pod restarts.
Kubernetes Deployments, which manage stateless ReplicaSets and assign random,
ephemeral hashes to Pod names, are entirely unsuitable for this requirement. The
Kubernetes StatefulSet is the optimal workload API for managing these
environments.25

A StatefulSet guarantees the strict ordering and uniqueness of its Pods. When a
StatefulSet is deployed in conjunction with a Headless Service—a Kubernetes
Service explicitly configured with clusterIP: None—the Kubernetes control plane
provisions highly deterministic DNS records within the cluster's internal
network.25 This mechanism circumvents the standard Kubernetes service proxy
layer, allowing direct, predictable access to individual Pods.

If an operator or an automated pipeline provisions a StatefulSet named
opencode-alice with a replica count of 1, the Kubernetes scheduler will create a
Pod strictly named opencode-alice-0. The CoreDNS component of the cluster will
then automatically generate an A-record that resolves directly to the IP address
of that specific Pod. The Fully Qualified Domain Name (FQDN) follows a strict
mathematical format:

opencode-alice-0.\<headless-service-name\>.\<namespace\>.svc.cluster.local.

This deterministic naming convention is the foundational element required for
dynamic routing without a centralized state database. If the reverse proxy or
ingress controller can extract the authenticated user's identity (e.g., alice)
from a verified token, it can dynamically construct the exact backend DNS
address of that user's dedicated OpenCode instance on the fly, eliminating the
need for complex REST API route injections or external lookup tables.

| Workload Type   | Pod Naming Convention                                                   | Network Identity                                           | Storage Persistence                                                                    | Suitability for User Workspaces                                                    |
| :-------------- | :---------------------------------------------------------------------- | :--------------------------------------------------------- | :------------------------------------------------------------------------------------- | :--------------------------------------------------------------------------------- |
| **Deployment**  | app-name-\<replicaset-hash\>-\<pod-hash\> (e.g., opencode-7b89c4f-zxt2) | Ephemeral; behind a shared ClusterIP Service.              | Ephemeral; PVCs cannot easily track specific pods across rescheduling.                 | Unsuitable. User mapping is impossible to predict mathematically.                  |
| **StatefulSet** | statefulset-name-\<ordinal\> (e.g., opencode-alice-0)                   | Deterministic; directly routable via Headless Service DNS. | Persistent; VolumeClaimTemplates ensure the same PVC follows the specific ordinal pod. | Highly Suitable. Enables direct dynamic routing based on predictable user handles. |

### **The Central Authentication and Redirection Challenge**

Establishing the backend workloads is only half the architectural challenge. The
ingress and proxy layer must be explicitly designed to orchestrate the
authentication and routing flow seamlessly. The system must fulfill the
following lifecycle:

1. Intercept all unauthenticated HTTP and WebSocket traffic arriving at a
   central, organizational domain (e.g., opencode.enterprise.com).
2. Redirect the unauthenticated user to a centralized SSO Identity Provider
   (e.g., GCP Identity or Google Workspace via OIDC).
3. Upon successful authentication, intercept the redirect callback and
   mathematically evaluate the resulting JSON Web Token (JWT) to extract the
   unique user identifier.
4. Dynamically route the incoming request to the corresponding StatefulSet
   backend (e.g., opencode-username-0) based solely on the extracted identifier,
   preserving all required paths and headers.

## **Evaluating Conventional Routing and Authentication Proxies**

The user query correctly identifies a critical gap in standard ingress
architectures: deploying a ubiquitous reverse proxy alongside an authentication
middleware like OAuth2-Proxy is insufficient out-of-the-box because these tools
lack native mechanisms for dynamic upstream user mapping.7

### **The Constraints of OAuth2-Proxy**

OAuth2-Proxy is a highly popular, open-source reverse proxy and static file
server designed to provide authentication using providers such as Google,
GitHub, and generic OpenID Connect.26 In Kubernetes environments, it is
typically deployed as an external authentication middleware for an Ingress
Controller, such as NGINX, via specific annotations like
nginx.ingress.kubernetes.io/auth-url and
nginx.ingress.kubernetes.io/auth-signin.8

While OAuth2-Proxy excels at its primary function—validating OIDC tokens,
managing session cookies, and forwarding user claims to the backend via headers
like X-Forwarded-User or X-Auth-Request-Email—it is fundamentally designed as a
static router.7 Traffic that successfully passes the authentication phase is
routed to a static upstream target defined in the proxy's configuration (via the
\--upstream flag or the upstreams configuration block).7 OAuth2-Proxy possesses
no internal mechanism to read an authenticated email address, parse the user
handle, and dynamically alter its upstream proxy target on a per-request basis.
Therefore, it cannot natively route alice@company.com to pod-alice and
bob@company.com to pod-bob.

### **The NGINX Ingress and Lua Scripting Approach**

When faced with the limitations of static upstream configuration, engineers
frequently resort to leveraging the NGINX Ingress Controller's advanced support
for Lua scripting to implement dynamic routing logic.31 Because the Kubernetes
ingress-nginx controller is built upon the OpenResty framework, it possesses the
capability to execute embedded Lua code at various phases of the request
lifecycle, allowing for the manipulation of routing variables and upstream
targets on the fly.31

In this theoretical architecture, the NGINX Ingress is configured with an
auth-url annotation pointing to the OAuth2-Proxy deployment.34 When a request
arrives, NGINX pauses processing and sends a subrequest to OAuth2-Proxy. Once
OAuth2-Proxy verifies the session, it returns a 200 OK status code, alongside
the injected X-Auth-Request-Email header containing the user's identity.34 The
NGINX Ingress then utilizes the
nginx.ingress.kubernetes.io/configuration-snippet annotation to execute a block
of Lua code. This script intercepts the header, extracts the prefix of the email
address, formats the specific Kubernetes DNS string for the target StatefulSet,
and dynamically updates the proxy\_pass variable.33

**Conceptual Lua Dynamic Upstream Logic:**

Nginx

auth\_request\_set $user\_email $upstream\_http\_x\_auth\_request\_email;\
set\_by\_lua\_block $dynamic\_upstream {\
local email \= ngx.var.user\_email\
if email then\
\-- Extract the username prefix from the email address\
local user \= string.match(email, "(\[^@\]+)@")\
\-- Construct the deterministic Headless Service DNS address\
return "http://opencode-".. user..
"-0.opencode-headless.opencode.svc.cluster.local:8080"\
end\
\-- Fallback target for unmapped traffic\
return "http://default-fallback.opencode.svc.cluster.local:8080"\
}\
proxy\_pass $dynamic\_upstream;

#### **Severe Limitations and Security Risks of the Lua Approach**

While the Lua scripting approach is functionally capable of resolving the user
mapping problem, it introduces severe operational, performance, and security
liabilities that make it unsuitable for an enterprise-grade platform requiring
high maintainability.

1. **Critical Security Vulnerabilities:** The integration of Lua scripting via
   user-supplied annotations in ingress-nginx has historically exposed
   Kubernetes clusters to catastrophic Remote Code Execution (RCE) and injection
   vulnerabilities.33 Vulnerabilities such as CVE-2021-25742 and CVE-2025-1097
   stem from the fact that the ingress controller's Lua-based validation logic
   can often be tricked by maliciously crafted annotations or manipulated
   headers, allowing attackers to inject arbitrary directives directly into the
   NGINX configuration template or execute code within the ingress pod.33
2. **Operational Complexity and Configuration Sprawl:** Managing complex,
   multi-line Lua logic within YAML annotations (configuration-snippet) leads to
   significant configuration sprawl.33 Testing and debugging embedded Lua
   scripts within a Kubernetes manifest is notoriously difficult, as syntax
   errors can cause the entire NGINX controller to fail to reload its
   configuration, potentially causing cluster-wide routing outages.33 This
   approach heavily violates the principle of "easy to maintain."
3. **Performance Degradation and Resource Leaks:** Dynamic reconfiguration via
   Lua forces the NGINX master process to execute script blocks for every single
   incoming request, bypassing the highly optimized static upstream routing
   engines that make NGINX fast.33 Furthermore, errors in Lua shared dictionary
   management (ngx.shared.DICT) are a known cause of gradual, indefinite memory
   consumption increases, leading to Out-Of-Memory (OOM) crashes.33 The frequent
   dynamic endpoint updates driven by Lua can also trigger the accumulation of
   "zombie processes," where the NGINX master process fails to properly reap
   worker child processes, severely impacting the stability of the ingress
   layer.33

### **Analyzing Traefik and ForwardAuth Constraints**

An alternative ingress controller frequently evaluated for this pattern is
Traefik, which utilizes a dedicated ForwardAuth middleware to delegate
authentication decisions to external services.37 Similar to the NGINX flow, if
the external service (such as OAuth2-Proxy or a custom authentication container)
answers with a 2xx HTTP status code, Traefik grants access and seamlessly copies
specified headers (e.g., X-Forwarded-User or custom authorization tokens) from
the authentication response into the original request before forwarding it to
the backend.38

Traefik boasts robust support for dynamic routing based on HTTP headers,
utilizing matchers such as Header and HeaderRegexp to route traffic based on the
presence or value of specific headers.41 However, a critical limitation prevents
this from being a viable solution for large-scale multi-tenancy: Traefik's
routing rules must be explicitly and individually defined for every single
backend service.42

To route the user alice to the pod opencode-alice and the user bob to the pod
opencode-bob, an administrator must define two distinct IngressRoute Custom
Resource Definitions (CRDs). Each IngressRoute must contain a specific
Headers("X-Forwarded-User", "alice") matcher pointing to Alice's Kubernetes
Service, and a separate matcher for Bob pointing to Bob's Service.43 Traefik
does not natively support the ability to capture a value from a header injected
by the ForwardAuth middleware and dynamically interpolate that captured string
into the backend Service hostname within a single, unified route definition.43

Therefore, relying purely on Traefik for dynamic routing requires the deployment
of external automation—such as a custom Kubernetes controller or complex CI/CD
scripts—to generate, apply, and manage an individual IngressRoute object for
every single user on the platform, significantly increasing operational
overhead.

## ---

**Proposed Architecture 1: The Identity-Aware Dynamic Proxy Model (Authentik)**

To achieve a seamless, low-maintenance replication of JupyterHub's dynamic
routing without resorting to the security risks of Lua scripting or the
configuration sprawl of static ingress generation, the optimal architecture
relies on deploying a specialized Identity-Aware Proxy (IAP). **Authentik**, an
open-source Identity Provider, features a highly capable built-in proxy
mechanism specifically designed to handle dynamic backend selection based on the
programmatic evaluation of authenticated user attributes.9

Authentik elegantly unifies the Identity Provider (IdP) and the Proxy into a
single, cohesive control plane.46 This consolidation eliminates the complex,
multi-hop handoffs between NGINX, OAuth2-Proxy, and custom Lua scripts,
providing a highly secure, centrally managed authentication and routing layer.

### **Architecture Topology**

| Component              | Technology                             | Function                                                                                                                           |
| :--------------------- | :------------------------------------- | :--------------------------------------------------------------------------------------------------------------------------------- |
| **External DNS**       | Cloud DNS / Route53                    | Points a wildcard domain (e.g., \*.opencode.enterprise.com) or a root domain to the cluster Ingress.                               |
| **Ingress Controller** | Standard Ingress (e.g., NGINX/Traefik) | Handles primary SSL termination and unconditionally forwards all traffic for the target domain to the Authentik Outpost.           |
| **Identity / Proxy**   | Authentik Embedded Outpost             | Authenticates users via GCP Workspace (OIDC), evaluates user claims, and executes a dynamic override of the upstream backend URL.9 |
| **Stateful Workloads** | Kubernetes StatefulSets                | Runs individual OpenCode instances (opencode-user-0) with mounted Persistent Volume Claims (PVCs) for workspace persistence.25     |

### **Detailed Implementation Strategy**

#### **1\. Central Authentication and Identity Federation**

Authentik is deployed directly within the Kubernetes cluster, utilizing its
official Helm chart to manage the core worker pods, Redis cache, and PostgreSQL
database.48 Within the comprehensive Authentik Admin UI, an administrator
configures an **OAuth2/OpenID Connect Source** and federates it with Google
Cloud Platform (GCP) Identity or Google Workspace.27 This configuration fulfills
the requirement for a central SSO sign-on. When developers attempt to access the
proxy, they are presented with an Authentik login screen, which can be
configured to immediately redirect them to the familiar Google authentication
flow, ensuring a frictionless login experience.46

#### **2\. The Proxy Provider and Dynamic Backend Override**

Authentik manages access by connecting "Applications" to "Providers." For this
specific architecture, an administrator creates a **Proxy Provider**.27 The
proxy provider is designed to intercept network traffic, ensure the user
possesses a valid authentication session, and then forward the traffic to a
protected backend service.27

The critical feature that establishes Authentik as the ideal replacement for
JupyterHub's configurable-http-proxy is its native support for **Dynamic Backend
Selection** via Scope and Property Mappings.9 Authentik allows administrators to
write highly secure, sandboxed Python expressions that evaluate _during the
login flow_ to dynamically modify the proxy's routing behavior for that specific
user's session.49

To implement the routing logic, an administrator creates a new **Scope Mapping**
in the Authentik UI with the following Python expression:

Python

\# Extract the full email address from the authenticated GCP Workspace profile\
email \= request.user.email

\# Split the email to isolate the organizational username (e.g.,
alice@company.com \-\> alice)\
username \= email.split('@')

\# Construct the exact, deterministic Kubernetes DNS name for the user's
StatefulSet Headless Service\
\# Format:
\<pod-name\>.\<headless-service-name\>.\<namespace\>.svc.cluster.local:\<port\>\
dynamic\_upstream \=
f"http://opencode-{username}\-0.opencode-headless-svc.opencode.svc.cluster.local:8080"

\# Return the backend\_override directive to dynamically configure the Authentik
Proxy target\
return {\
"ak\_proxy": {\
"backend\_override": dynamic\_upstream\
}\
}

This Scope Mapping is then attached directly to the Proxy Provider
configuration.9 The elegance of this approach lies in its execution timing: when
the user successfully authenticates via Google Workspace, Authentik safely
executes this Python snippet within its secure control plane, calculates the
deterministic backend URL, and overrides the proxy target for the duration of
that user's session.9 Unlike NGINX Lua scripts, which must execute blindly on
every single HTTP request based on easily spoofed headers, the Authentik
expression evaluates securely during the token issuance phase, providing a
significantly higher degree of security and performance.9

#### **3\. Deterministic Kubernetes Resource Provisioning**

For the dynamic routing expression to resolve correctly, the backend OpenCode
instances must be deployed following a strict, predictable Kubernetes naming
convention.

First, a headless Kubernetes Service must be created to govern the internal
network domain for all OpenCode instances:

YAML

apiVersion: v1\
kind: Service\
metadata:\
name: opencode-headless-svc\
namespace: opencode\
spec:\
\# clusterIP: None defines a Headless Service for direct Pod routing\
clusterIP: None\
selector:\
app: opencode-instance

When a new user requires access to the platform, an administrator (or an
automated self-service pipeline) deploys a StatefulSet specifically named for
that user:

YAML

apiVersion: apps/v1\
kind: StatefulSet\
metadata:\
name: opencode-alice\
namespace: opencode\
spec:\
serviceName: "opencode-headless-svc"\
replicas: 1\
selector:\
matchLabels:\
app: opencode-instance\
user: alice\
template:\
metadata:\
labels:\
app: opencode-instance\
user: alice\
spec:\
containers:\
\- name: opencode\
image: ghcr.io/timothyclin/k8s-omo/opencode-workspace:0.1.0\
ports:\
\- containerPort: 8080\
\# Dynamically provision persistent storage for the AI agent's workspace\
volumeClaimTemplates:\
\- metadata:\
name: opencode-workspace\
spec:\
accessModes:\
resources:\
requests:\
storage: 20Gi

Because the StatefulSet is explicitly named opencode-alice, the Kubernetes
scheduler automatically assigns the resulting pod the specific hostname
opencode-alice-0. Consequently, the cluster's internal CoreDNS resolver will
route opencode-alice-0.opencode-headless-svc.opencode.svc.cluster.local directly
to Alice's Pod IP address, perfectly aligning with the URL generated by the
Authentik Python expression.25

#### **4\. Execution of the Traffic Flow**

1. The developer navigates to the central access URL,
   https://opencode.enterprise.com.
2. The cluster's primary Ingress Controller receives the connection and routes
   the traffic unconditionally to the Authentik Proxy Outpost deployment.51
3. The Authentik Outpost detects an unauthenticated session and redirects the
   user's browser to the GCP Workspace SSO portal.27
4. The developer completes the authentication challenge. Authentik receives the
   secure OIDC token and extracts the user's profile information, yielding
   alice@enterprise.com.
5. Authentik evaluates the assigned Scope Mapping policy, executing the Python
   script to set the internal proxy target to
   http://opencode-alice-0.opencode-headless-svc.opencode.svc.cluster.local:8080.9
6. All subsequent HTTP and WebSocket traffic generated by the OpenCode web
   interface is seamlessly and securely proxied directly to Alice's isolated
   StatefulSet.

### **Advantages in Maintainability and Security Posture**

The Authentik architecture represents a highly maintainable and secure solution.
It completely removes the requirement for custom Go or Python controller code
that must continuously watch Kubernetes pod states and inject routes via fragile
REST APIs, a major pain point in scaling JupyterHub.12 It avoids the operational
nightmares and security vulnerabilities associated with complex Lua scripts
operating on raw HTTP headers within the ingress layer.33 The logic relies
entirely on the mathematical predictability of Kubernetes StatefulSet DNS and
Authentik's native, purpose-built Identity-Aware Proxy capabilities.
Furthermore, because Authentik manages the entire authentication perimeter,
unauthenticated network packets are dropped at the edge, ensuring they never
reach the internal Kubernetes network space where the OpenCode agents reside.

## ---

**Proposed Architecture 2: The Kubernetes-Native GitOps Pattern (Pomerium)**

While the dynamic proxy approach leveraging Authentik is highly efficient and
minimizes the number of required Kubernetes resources, certain enterprise
environments adhere to strict, declarative GitOps methodologies. In these
environments, every single routing path and access policy must be explicitly
defined, reviewed, and version-controlled in source control repositories. If
dynamic string evaluation at the proxy layer is deemed undesirable by security
or architecture review boards, the architecture must shift the burden of routing
logic away from the proxy and firmly into the Kubernetes control plane.

This objective can be achieved by pairing **Pomerium**, an advanced, open-source
Identity-Aware Proxy based on the high-performance Envoy proxy engine, with a
Kubernetes GitOps operator such as ArgoCD or FluxCD.52

### **Shifting from Dynamic Proxying to Dynamic Provisioning**

Instead of configuring a single proxy endpoint that intelligently guesses the
backend target based on parsed user headers, the Pomerium architecture relies on
provisioning explicit, individual Ingress routes for every user automatically.

Pomerium integrates seamlessly and natively with Kubernetes via the Pomerium
Ingress Controller.11 Operating as a first-class Kubernetes component, it
enforces fine-grained, Zero Trust access control policies based on user identity
(evaluated via OIDC claims) and routes traffic directly to backend endpoints,
bypassing the standard Kubernetes service proxy for improved performance.11

### **Architecture Topology**

| Component              | Technology                  | Function                                                                                                                                  |
| :--------------------- | :-------------------------- | :---------------------------------------------------------------------------------------------------------------------------------------- |
| **GitOps Controller**  | ArgoCD / FluxCD             | Continuously monitors a Git repository for user environment definitions and automatically stamps out the required Kubernetes manifests.53 |
| **Ingress Controller** | Pomerium Ingress Controller | Authenticates traffic via GCP Workspace, evaluates OIDC claims against declarative policies, and routes traffic.11                        |
| **Stateful Workloads** | Kubernetes StatefulSets     | Isolated OpenCode instances, deployed identically to Architecture 1\.                                                                     |

### **Detailed Implementation Strategy**

#### **1\. Central Authentication and Context-Aware Evaluation**

The Pomerium Ingress Controller is deployed to the cluster and centrally
configured with the organization's GCP Workspace OIDC credentials within its
global CRD settings (utilizing the authenticate.idp.provider: google
specification).10 Pomerium operates under a strict Zero Trust model; it does not
merely authenticate the user once at the edge. Instead, it continuously verifies
the user's identity, contextual state, and OIDC claims for every single HTTP
request before proxying traffic to the upstream service.55

#### **2\. Declarative User Workspaces via GitOps Automation**

When a new developer is onboarded to the platform, an administrator commits a
simple declarative definition to a central Git repository. Alternatively, an
internal developer portal (such as Backstage) can trigger an API call to
generate a Helm release payload.

A standardized Helm chart is designed to deploy a complete "Developer
Workspace." For a user named bob, the GitOps controller detects the commit and
automatically applies a package containing the following resources:

1. The OpenCode StatefulSet (opencode-bob).
2. A standard ClusterIP Service mapping directly to the StatefulSet
   (opencode-bob-svc).
3. An Ingress resource uniquely configured for Bob's workspace.

The Ingress resource utilizes specific Pomerium annotations to enforce a strict
cryptographic policy: it dictates that _only_ the user possessing an OIDC claim
proving ownership of the email address bob@enterprise.com is permitted to access
this specific routing path.11

YAML

apiVersion: networking.k8s.io/v1\
kind: Ingress\
metadata:\
name: opencode-bob-ingress\
namespace: opencode\
annotations:\
\# Explicitly enforce authorization based on GCP Workspace email claim\
ingress.pomerium.io/policy: |\
\- allow:\
and:\
\- claim/email: bob@enterprise.com\
\# Forward identity context to the backend if the OpenCode agent requires it\
ingress.pomerium.io/pass\_identity\_headers: "true"\
spec:\
ingressClassName: pomerium\
rules:\
\# Define a unique, user-specific subdomain for direct access\
\- host: bob.opencode.enterprise.com\
http:\
paths:\
\- path: /\
pathType: Prefix\
backend:\
service:\
name: opencode-bob-svc\
port:\
number: 8080

#### **3\. Execution of the Traffic Flow**

1. The developer attempts to access their dedicated workspace URL,
   https://bob.opencode.enterprise.com.
2. The Pomerium Ingress Controller intercepts the network request at the cluster
   edge.
3. Detecting that the session lacks a valid identity token, Pomerium redirects
   the developer to the GCP Workspace central SSO UI.10
4. Upon successful authentication, Pomerium receives the token and evaluates the
   encapsulated OIDC claims against the strict policy defined in the Ingress
   annotation (claim/email: bob@enterprise.com).53
5. Because the claims mathematically match the policy requirements, Pomerium
   authorizes the request and proxies the traffic directly to the
   opencode-bob-svc backend.11
6. If the pass\_identity\_headers annotation is active, Pomerium signs the
   user's JWT and injects it into the x-pomerium-jwt-assertion HTTP header,
   passing it downstream. This allows the OpenCode agent, or any sidecar running
   alongside it, to maintain deep cryptographic awareness of exactly who is
   issuing commands within the container.57

### **Advantages in Maintainability and Declarative Security**

The Pomerium architecture represents the absolute pinnacle of Kubernetes
operational best practices. By completely abandoning the concept of a "dynamic
proxy" executing logic at runtime, the architecture entirely avoids complex
scripting, header extraction manipulations, or the management of fragile,
memory-based routing tables.11

Every developer's access policy, routing path, and stateful workload definition
is explicitly documented in human-readable YAML within source control, rendering
disaster recovery and compliance auditing utterly trivial.55 If the Pomerium
controller experiences a catastrophic crash, it simply rebuilds its internal
routing table instantly by reading the declarative state of the Kubernetes
Ingress objects via the native API server.

## **Managing Stateful Workloads for Autonomous AI Agents**

Regardless of whether an organization selects the Dynamic Proxy (Authentik) or
the GitOps (Pomerium) architectural pattern, managing the underlying state of AI
coding assistants like OpenCode requires the meticulous configuration of
Kubernetes primitives.

OpenCode functions as an agentic framework capable of profound infrastructure
interaction. It executes subagent processes (such as the Build and Plan modules)
that require high-performance, persistent access to source code repositories,
deeply nested configuration files (e.g., opencode.json, and
\~/.local/share/opencode/auth.json which stores highly sensitive API keys for
LLM providers), and local SQLite databases utilized for maintaining context logs
and session history across restarts.1

### **Persistent Volume Claims (PVCs) for Workspace Continuity**

JupyterHub's KubeSpawner dynamically generates Persistent Volume Claims (PVCs)
for notebook instances to guarantee that user data and environments survive the
inevitable pod restarts inherent in distributed systems.18 In both of the
proposed architectures, the volumeClaimTemplates block nested within the
StatefulSet specification achieves the exact same continuity natively, without
requiring external management logic.25

When the opencode-alice-0 pod is initially scheduled onto a worker node, the
Kubernetes storage controller dynamically provisions a PersistentVolume (PV)
from the default storage class and irrevocably binds it to a claim named
opencode-workspace-opencode-alice-0. If the physical node underlying the pod
suffers a catastrophic hardware failure, the Kubernetes control plane will
seamlessly reschedule the opencode-alice-0 pod to a healthy worker node and
remount the exact same PVC. This ensures that Alice's uncommitted source code,
configured API tokens, and AI context history remain perfectly intact with
absolute zero data loss.

### **Environment Security and Mitigation of Privilege Escalation**

OpenCode essentially provides an interactive, AI-driven shell directly into the
underlying container environment. Granting an autonomous AI agent—or a developer
utilizing the agent—full access to a container inherently means that the
container must be strictly isolated from the host node and the broader
Kubernetes cluster.

To mitigate the severe risk of lateral movement—whether originating from a
malicious developer attempting to break out of their workspace or a hijacked LLM
executing unintended, malicious shell code—the StatefulSet deployments must
enforce the strictest possible security contexts.19

1. **Disable Automount Service Account Token:** The automountServiceAccountToken
   field in the Pod specification must be explicitly set to false.19 By default,
   Kubernetes mounts a service account token into every pod, granting the
   processes inside potential access to the Kubernetes API server. If an
   OpenCode agent discovers this token, it could theoretically be manipulated
   into querying cluster state or modifying resources. Disabling this mount
   entirely prevents the AI from possessing any awareness of the Kubernetes
   control plane.
2. **Prevent Privilege Escalation:** The allowPrivilegeEscalation flag within
   the container's security context must be set to false.19 This critical
   setting ensures that any process spawned inside the container cannot gain
   more privileges than its parent process, effectively neutralizing the threat
   of exploited setuid binaries (such as sudo) that might be present in the base
   image.
3. **Strict Network Policies:** Organizations must implement default-deny
   NetworkPolicy objects within the opencode namespace. The OpenCode pods should
   only be permitted to receive ingress HTTP/TCP traffic from the specific
   namespaces housing the Authentik or Pomerium proxy deployments. Furthermore,
   egress traffic should be strictly limited to the public internet (to allow
   the agent to reach external LLM APIs like OpenAI or Anthropic) and explicitly
   authorized internal Git repositories.59 Critically, Pod-to-Pod communication
   within the opencode namespace must be categorically blocked to prevent one
   developer's compromised agent from scanning or attacking another developer's
   isolated workspace.

## **Strategic Recommendations**

Replicating JupyterHub's elegant multi-user design for customized, stateful
workloads like the OpenCode AI assistant requires decoupling the concept of
dynamic network routing from monolithic application controllers. While
JupyterHub effectively manages this through a custom polling process and a
REST-controlled in-memory proxy, modern cloud-native infrastructure offers
significantly more robust, secure, and maintainable architectural pathways.

Attempting to force traditional reverse proxies, such as the widely deployed
OAuth2-Proxy and NGINX combination, to perform dynamic backend mapping via
embedded Lua scripting introduces unacceptable security vulnerabilities,
operational fragility, and configuration sprawl.

Instead, enterprises should evaluate their internal operational maturity and
implement one of the two following architectures:

1. **The Authentik Dynamic Proxy Model:** This architecture is optimal for
   environments prioritizing a single, centralized entry point (e.g.,
   opencode.company.com/workspace) and minimizing the total number of Kubernetes
   objects. Authentik seamlessly handles GCP SSO integration and evaluates
   secure, sandboxed Python snippets during the initial login flow. This
   dynamically overwrites the proxy targets to deterministic StatefulSet
   headless DNS addresses, providing a direct, highly maintainable replacement
   for JupyterHub's dynamic proxying mechanism while keeping all routing logic
   centralized within the Identity Provider.
2. **The Pomerium Declarative GitOps Model:** This architecture is the gold
   standard for organizations with mature automation pipelines that prioritize
   explicit infrastructure-as-code methodologies. By utilizing Pomerium as an
   Identity-Aware Ingress Controller, routing and authorization are dictated
   explicitly via Kubernetes Ingress annotations. This eliminates dynamic proxy
   evaluation entirely, replacing it with automated, declarative provisioning of
   unique subdomains and strict OIDC claim enforcement, yielding an
   infrastructure that is infinitely auditable and effortlessly reproducible.

Both proposed architectures successfully eliminate the need for bespoke
middleware controllers, ensure strict stateful isolation for autonomous AI
agents, integrate flawlessly with centralized Google Cloud identity providers,
and provide a secure, low-maintenance environment that empowers enterprise
development teams to leverage the next generation of AI coding assistants
safely.

#### **Works cited**

1. Agents \- OpenCode, accessed April 1, 2026,
   [https://opencode.ai/docs/agents/](https://opencode.ai/docs/agents/)
2. OpenCode Tutorial 2026: Complete Install, Setup & Configuration Guide \-
   NxCode, accessed April 1, 2026,
   [https://www.nxcode.io/resources/news/opencode-tutorial-2026](https://www.nxcode.io/resources/news/opencode-tutorial-2026)
3. OpenCode | The open source AI coding agent, accessed April 1, 2026,
   [https://opencode.ai/](https://opencode.ai/)
4. OpenCode with Docker Model Runner for Private AI Coding, accessed April 1,
   2026,
   [https://www.docker.com/blog/opencode-docker-model-runner-private-ai-coding/](https://www.docker.com/blog/opencode-docker-model-runner-private-ai-coding/)
5. jupyterhub/jupyterhub: Multi-user server for Jupyter notebooks \- GitHub,
   accessed April 1, 2026,
   [https://github.com/jupyterhub/jupyterhub](https://github.com/jupyterhub/jupyterhub)
6. JupyterHub documentation, accessed April 1, 2026,
   [https://jupyterhub.readthedocs.io/\_/downloads/en/1.4.2/epub/](https://jupyterhub.readthedocs.io/_/downloads/en/1.4.2/epub/)
7. Using OAuth2 proxy for Kubernetes Dashboard \- Funky Penguin's Geek Cookbook,
   accessed April 1, 2026,
   [https://geek-cookbook.funkypenguin.co.nz/recipes/kubernetes/oauth2-proxy/](https://geek-cookbook.funkypenguin.co.nz/recipes/kubernetes/oauth2-proxy/)
8. How to get oauth2\_proxy running in kubernetes under one domain to redirect
   back to original domain that required authentication? \- Stack Overflow,
   accessed April 1, 2026,
   [https://stackoverflow.com/questions/55770138/how-to-get-oauth2-proxy-running-in-kubernetes-under-one-domain-to-redirect-back](https://stackoverflow.com/questions/55770138/how-to-get-oauth2-proxy-running-in-kubernetes-under-one-domain-to-redirect-back)
9. Proxy Provider \- authentik, accessed April 1, 2026,
   [https://docs.goauthentik.io/add-secure-apps/providers/proxy/](https://docs.goauthentik.io/add-secure-apps/providers/proxy/)
10. Global Configuration \- Pomerium, accessed April 1, 2026,
    [https://www.pomerium.com/docs/deploy/k8s/configure](https://www.pomerium.com/docs/deploy/k8s/configure)
11. Introducing Pomerium Ingress Controller for Kubernetes, accessed April 1,
    2026,
    [https://www.pomerium.com/blog/introducing-pomerium-ingress-controller-for-kubernetes](https://www.pomerium.com/blog/introducing-pomerium-ingress-controller-for-kubernetes)
12. Spawners \- JupyterHub documentation \- Read the Docs, accessed April 1,
    2026,
    [https://jupyterhub.readthedocs.io/en/latest/reference/spawners.html](https://jupyterhub.readthedocs.io/en/latest/reference/spawners.html)
13. Architecture — Jupyter Documentation 4.1.1 alpha documentation, accessed
    April 1, 2026,
    [https://docs.jupyter.org/en/stable/projects/architecture/content-architecture.html](https://docs.jupyter.org/en/stable/projects/architecture/content-architecture.html)
14. jupyterhub/configurable-http-proxy: node-http-proxy plus a REST API \-
    GitHub, accessed April 1, 2026,
    [https://github.com/jupyterhub/configurable-http-proxy](https://github.com/jupyterhub/configurable-http-proxy)
15. Configuration Reference \- Zero to JupyterHub with Kubernetes, accessed
    April 1, 2026,
    [https://z2jh.jupyter.org/en/0.10.x/resources/reference.html](https://z2jh.jupyter.org/en/0.10.x/resources/reference.html)
16. JupyterHub documentation, accessed April 1, 2026,
    [https://jupyterhub.readthedocs.io/\_/downloads/en/0.9.1/epub/](https://jupyterhub.readthedocs.io/_/downloads/en/0.9.1/epub/)
17. Running proxy separately from the hub \- JupyterHub documentation, accessed
    April 1, 2026,
    [https://jupyterhub.readthedocs.io/en/latest/howto/separate-proxy.html](https://jupyterhub.readthedocs.io/en/latest/howto/separate-proxy.html)
18. jupyterhub/kubespawner: Kubernetes spawner for JupyterHub \- GitHub,
    accessed April 1, 2026,
    [https://github.com/jupyterhub/kubespawner](https://github.com/jupyterhub/kubespawner)
19. JupyterHub Spawner to spawn user notebooks on a Kubernetes cluster. \-
    Kubespawner, accessed April 1, 2026,
    [https://jupyterhub-kubespawner.readthedocs.io/en/6.2.0/spawner.html](https://jupyterhub-kubespawner.readthedocs.io/en/6.2.0/spawner.html)
20. rest-api.yml \- jupyterhub/configurable-http-proxy \- GitHub, accessed April
    1, 2026,
    [https://github.com/jupyterhub/configurable-http-proxy/blob/master/doc/rest-api.yml](https://github.com/jupyterhub/configurable-http-proxy/blob/master/doc/rest-api.yml)
21. Proxies \- JupyterHub documentation, accessed April 1, 2026,
    [https://jupyterhub.readthedocs.io/en/latest/reference/api/proxy.html](https://jupyterhub.readthedocs.io/en/latest/reference/api/proxy.html)
22. How to configure Jupyterhub to run code-server? \- Jupyter Community Forum,
    accessed April 1, 2026,
    [https://discourse.jupyter.org/t/how-to-configure-jupyterhub-to-run-code-server/11578](https://discourse.jupyter.org/t/how-to-configure-jupyterhub-to-run-code-server/11578)
23. \[Feature Request\] Allow Spawners to respect Container ENTRYPOINT/CMD for
    robust init (tini) and optimized jupyter-server deployments · Issue \#5296
    \- GitHub, accessed April 1, 2026,
    [https://github.com/jupyterhub/jupyterhub/issues/5296](https://github.com/jupyterhub/jupyterhub/issues/5296)
24. Intro | AI coding agent built for the terminal \- OpenCode, accessed April
    1, 2026, [https://opencode.ai/docs/](https://opencode.ai/docs/)
25. StatefulSets \- Kubernetes, accessed April 1, 2026,
    [https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/)
26. charts/bitnami/oauth2-proxy/README.md at main \- GitHub, accessed April 1,
    2026,
    [https://github.com/bitnami/charts/blob/main/bitnami/oauth2-proxy/README.md](https://github.com/bitnami/charts/blob/main/bitnami/oauth2-proxy/README.md)
27. OAuth 2.0 provider \- authentik, accessed April 1, 2026,
    [https://docs.goauthentik.io/add-secure-apps/providers/oauth2/](https://docs.goauthentik.io/add-secure-apps/providers/oauth2/)
28. Deploying an OAuth Proxy for Internal Kubernetes Applications \- Manish
    Kumar, accessed April 1, 2026,
    [https://www.manishk.dev/blogs/oauth2-proxy/](https://www.manishk.dev/blogs/oauth2-proxy/)
29. Lock down your Kubernetes services with OAuth2 Proxy \- DEV Community,
    accessed April 1, 2026,
    [https://dev.to/styren/lock-down-your-kubernetes-services-with-oauth2-proxy-28d9](https://dev.to/styren/lock-down-your-kubernetes-services-with-oauth2-proxy-28d9)
30. Protecting Kubernetes Ingress Resources with Traefik ForwardAuth and
    oauth2-proxy | by Brendan Dalpe | Medium, accessed April 1, 2026,
    [https://medium.com/@bdalpe/protecting-kubernetes-ingress-resources-with-traefik-forwardauth-and-oauth2-proxy-a7b3d330f276](https://medium.com/@bdalpe/protecting-kubernetes-ingress-resources-with-traefik-forwardauth-and-oauth2-proxy-a7b3d330f276)
31. How to Set Up Nginx with Lua for Dynamic Routing on Ubuntu \- OneUptime,
    accessed April 1, 2026,
    [https://oneuptime.com/blog/post/2026-03-02-setup-nginx-lua-dynamic-routing-ubuntu/view](https://oneuptime.com/blog/post/2026-03-02-setup-nginx-lua-dynamic-routing-ubuntu/view)
32. Mastering Advanced Ingress-Nginx Techniques: Unleash the Power of Lua
    Scripts \- Medium, accessed April 1, 2026,
    [https://medium.com/@wadexu007/mastering-advanced-ingress-nginx-techniques-unleash-the-power-of-lua-scripts-8e827d5f8cbe](https://medium.com/@wadexu007/mastering-advanced-ingress-nginx-techniques-unleash-the-power-of-lua-scripts-8e827d5f8cbe)
33. The Complex Dance of Lua and NGINX: Power, Pitfalls, and Performance
    Challenges, accessed April 1, 2026,
    [https://blog.nginx.org/blog/the-complex-dance-of-lua-and-nginx-power-pitfalls-and-performance-challenges](https://blog.nginx.org/blog/the-complex-dance-of-lua-and-nginx-power-pitfalls-and-performance-challenges)
34. ingress-nginx/docs/user-guide/nginx-configuration/annotations.md at main \-
    GitHub, accessed April 1, 2026,
    [https://github.com/kubernetes/ingress-nginx/blob/main/docs/user-guide/nginx-configuration/annotations.md](https://github.com/kubernetes/ingress-nginx/blob/main/docs/user-guide/nginx-configuration/annotations.md)
35. Header-based backend selection with nginx ingress controller | Blog \-
    viesure, accessed April 1, 2026,
    [https://viesure.io/nginx-ingress-controller/tech/](https://viesure.io/nginx-ingress-controller/tech/)
36. A Guide to Choosing an Ingress Controller, Part 2: Risks and Future-Proofing
    | F5, accessed April 1, 2026,
    [https://www.f5.com/company/blog/nginx/guide-to-choosing-ingress-controller-part-2-risks-future-proofing](https://www.f5.com/company/blog/nginx/guide-to-choosing-ingress-controller-part-2-risks-future-proofing)
37. How to Implement Forward Authentication in Traefik \- OneUptime, accessed
    April 1, 2026,
    [https://oneuptime.com/blog/post/2026-01-23-traefik-forward-authentication/view](https://oneuptime.com/blog/post/2026-01-23-traefik-forward-authentication/view)
38. Traefik ForwardAuth Documentation, accessed April 1, 2026,
    [https://doc.traefik.io/traefik/middlewares/http/forwardauth/](https://doc.traefik.io/traefik/middlewares/http/forwardauth/)
39. ForwardAuth | Traefik Hub Documentation, accessed April 1, 2026,
    [https://doc.traefik.io/traefik-hub/api-gateway/reference/routing/http/middlewares/ref-forward-auth](https://doc.traefik.io/traefik-hub/api-gateway/reference/routing/http/middlewares/ref-forward-auth)
40. Setting an Authorization header after a ForwardAuth in Traefik \- Stack
    Overflow, accessed April 1, 2026,
    [https://stackoverflow.com/questions/62016654/setting-an-authorization-header-after-a-forwardauth-in-traefik](https://stackoverflow.com/questions/62016654/setting-an-authorization-header-after-a-forwardauth-in-traefik)
41. Traefik HTTP Routers Rules & Priority Documentation, accessed April 1, 2026,
    [https://doc.traefik.io/traefik/routing/routers/](https://doc.traefik.io/traefik/routing/routers/)
42. Dynamic routing based on pod labels \- Traefik Labs Community Forum,
    accessed April 1, 2026,
    [https://community.traefik.io/t/dynamic-routing-based-on-pod-labels/2305](https://community.traefik.io/t/dynamic-routing-based-on-pod-labels/2305)
43. Middleware forwardauth does not apply \- Traefik Labs Community Forum,
    accessed April 1, 2026,
    [https://community.traefik.io/t/middleware-forwardauth-does-not-apply/1839](https://community.traefik.io/t/middleware-forwardauth-does-not-apply/1839)
44. Header based routing to a specific Service in Kubernetes : r/Traefik \-
    Reddit, accessed April 1, 2026,
    [https://www.reddit.com/r/Traefik/comments/ur0ld9/header\_based\_routing\_to\_a\_specific\_service\_in/](https://www.reddit.com/r/Traefik/comments/ur0ld9/header_based_routing_to_a_specific_service_in/)
45. Multi-Layer Routing \- Traefik Labs, accessed April 1, 2026,
    [https://doc.traefik.io/traefik/reference/routing-configuration/http/routing/multi-layer-routing/](https://doc.traefik.io/traefik/reference/routing-configuration/http/routing/multi-layer-routing/)
46. Flows, stages, and policies: customizing your authentication with authentik,
    accessed April 1, 2026,
    [https://goauthentik.io/blog/2024-08-27-flows-stages-and-policies/](https://goauthentik.io/blog/2024-08-27-flows-stages-and-policies/)
47. Providers \- authentik, accessed April 1, 2026,
    [https://docs.goauthentik.io/providers/](https://docs.goauthentik.io/providers/)
48. Configuration \- authentik, accessed April 1, 2026,
    [https://docs.goauthentik.io/install-config/configuration/](https://docs.goauthentik.io/install-config/configuration/)
49. Source property mappings \- authentik, accessed April 1, 2026,
    [https://docs.goauthentik.io/users-sources/sources/property-mappings/](https://docs.goauthentik.io/users-sources/sources/property-mappings/)
50. Kubernetes Stateful Sets \- Mapping existing IDs to persistent/stateful pods
    \- Stack Overflow, accessed April 1, 2026,
    [https://stackoverflow.com/questions/60346568/kubernetes-stateful-sets-mapping-existing-ids-to-persistent-stateful-pods](https://stackoverflow.com/questions/60346568/kubernetes-stateful-sets-mapping-existing-ids-to-persistent-stateful-pods)
51. Kubernetes \- authentik, accessed April 1, 2026,
    [https://docs.goauthentik.io/add-secure-apps/outposts/integrations/kubernetes/](https://docs.goauthentik.io/add-secure-apps/outposts/integrations/kubernetes/)
52. Kubernetes Security | Pomerium, accessed April 1, 2026,
    [https://www.pomerium.com/secure-service-access/kubernetes-security](https://www.pomerium.com/secure-service-access/kubernetes-security)
53. How to Deploy Pomerium Access Proxy with Flux CD \- OneUptime, accessed
    April 1, 2026,
    [https://oneuptime.com/blog/post/2026-03-13-deploy-pomerium-access-proxy-with-flux-cd/view](https://oneuptime.com/blog/post/2026-03-13-deploy-pomerium-access-proxy-with-flux-cd/view)
54. Setting up Authentik with Kubernetes and FluxCD \- Tim Van Wassenhove,
    accessed April 1, 2026,
    [https://timvw.be/2025/03/17/setting-up-authentik-with-kubernetes-and-fluxcd/](https://timvw.be/2025/03/17/setting-up-authentik-with-kubernetes-and-fluxcd/)
55. Scoped Access for Multi-tenant Environments \- Pomerium, accessed April 1,
    2026,
    [https://www.pomerium.com/secure-service-access/scoped-access-for-multi-tenant-environments](https://www.pomerium.com/secure-service-access/scoped-access-for-multi-tenant-environments)
56. How Pomerium Enforces Real-Time, Context-Based Access, accessed April 1,
    2026,
    [https://www.pomerium.com/blog/real-time-context-based-access](https://www.pomerium.com/blog/real-time-context-based-access)
57. Zero Fundamentals: Build Advanced Routes \- Pomerium, accessed April 1,
    2026,
    [https://www.pomerium.com/docs/get-started/fundamentals/zero/zero-advanced-routes](https://www.pomerium.com/docs/get-started/fundamentals/zero/zero-advanced-routes)
58. Authenticate user with bearer token for kubernetes-dashboard · Issue \#1638
    \- GitHub, accessed April 1, 2026,
    [https://github.com/pomerium/pomerium/issues/1638](https://github.com/pomerium/pomerium/issues/1638)
59. From NGINX to Pomerium: A Practical Migration Guide for Internal Kubernetes
    Applications, accessed April 1, 2026,
    [https://www.pomerium.com/blog/from-nginx-to-pomerium-a-practical-migration-guide-for-internal-kubernetes-applications](https://www.pomerium.com/blog/from-nginx-to-pomerium-a-practical-migration-guide-for-internal-kubernetes-applications)
60. Providers \- OpenCode, accessed April 1, 2026,
    [https://opencode.ai/docs/providers/](https://opencode.ai/docs/providers/)
