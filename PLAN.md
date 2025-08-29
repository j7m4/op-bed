
### Phase 1: Solidify the Fundamentals (Enhance `HelloWorld`)

Before creating a new operator, let's make your existing `HelloWorld` operator more robust. This will introduce three
critical concepts you'll use in every operator you write.

1. **Implement the Status Subresource:** Your operator currently *acts* on the world but doesn't *report* on what it has
   done. The `.status` subresource is how an operator communicates the state of the resources it manages back to the
   user.
    * **Goal:** Update the `HelloWorld` resource's status with the name of the Deployment it created and the number of
      available pods.
    * **Key Concepts:**
        * **Observability:** Users can run `kubectl get helloworld -o yaml` and see the current state without having to
          find the Deployment themselves.
        * **State Reporting:** The operator provides feedback on its actions.
    * **Action Steps:**
        1. Add a `HelloWorldStatus` struct in your `api/v1/helloworld_types.go` file.
        2. Add the `//+kubebuilder:subresource:status` marker above your `HelloWorld` type definition.
        3. In your `Reconcile` loop, after creating or getting the Deployment, read its `.status.availableReplicas`.
        4. Update the `HelloWorld` object's status field and use `r.Status().Update(ctx, helloWorld)` to write it back
           to the API server.

2. **Add Finalizers for Graceful Deletion:** What happens if you run `kubectl delete helloworld my-app`? Kubernetes will
   delete the `HelloWorld` resource, and its ownership of the Deployment will cause the Deployment to be garbage
   collected. But what if you needed to perform a cleanup action first, like exporting data or notifying an external
   system? This is what **finalizers** are for.
    * **Goal:** Prevent the `HelloWorld` resource from being deleted until its associated Deployment is successfully
      removed by the operator.
    * **Key Concepts:**
        * **Lifecycle Hooks:** Finalizers are a way to hook into the deletion process.
        * **Pre-delete Logic:** Ensuring a clean shutdown and resource cleanup.
    * **Action Steps:**
        1. In your reconciler, check if the resource `metadata.deletionTimestamp` is zero. If it is, and the resource
           doesn't have your finalizer string (e.g., `helloworld.my.domain/finalizer`), add it and update the resource.
        2. If `deletionTimestamp` is *not* zero, it means the user has requested deletion.
        3. Perform your cleanup logic (e.g., delete the Deployment you created).
        4. Once cleanup is complete, remove your finalizer string from the resource's metadata and update it. Kubernetes
           will now complete the deletion.

3. **Watch Secondary Resources:** Right now, your operator only reacts to changes in the `HelloWorld` resource. What if
   someone manually deletes the `busybox` Deployment? Your operator won't know until the next scheduled reconciliation (
   which could be minutes!). You need to watch the resources you create.
    * **Goal:** Trigger a `HelloWorld` reconciliation automatically if its child Deployment is modified or deleted.
    * **Key Concepts:**
        * **Responsive Reconciliation:** Making the operator react immediately to changes in the cluster state.
        * **Owner-Child Relationships:** Understanding how controllers can watch resources they create.
    * **Action Steps:**
        1. In your controller's `SetupWithManager` function, add a call to `builder.Owns(&appsv1.Deployment{})`.
        2. This tells the controller manager: "In addition to watching `HelloWorld` resources, also watch for any
           changes to Deployments. If a Deployment changes, find the `HelloWorld` resource that owns it and trigger a
           reconcile for that parent resource."

***

### Phase 2: The Stateful Application Operator (`SimpleDB`)

Now, let's create a new operator that manages a stateful application, like a simple database. This introduces the
challenge of orchestrating multiple, dependent resources.

* **Project:** A `SimpleDB` operator that deploys a single-node PostgreSQL instance.
* **CRD Spec:** The `SimpleDB` spec could include fields like `storageSize` (e.g., `1Gi`), `databaseName`, and
  `postgresVersion`.
* **Operator Logic:** For each `SimpleDB` resource, the operator must create and manage:
    1. A `Secret` to store the generated PostgreSQL password.
    2. A `PersistentVolumeClaim` (PVC) to provide stable storage.
    3. A `Deployment` that mounts the PVC and uses the password from the Secret.
    4. A `Service` to expose the PostgreSQL port within the cluster.
* **New Concepts Learned:**
    * **Managing Multiple Resources:** You're now orchestrating a set of interdependent resources, not just one.
    * **Handling State:** The operator must generate a password once and store it in a Secret, ensuring it doesn't
      change on every reconciliation. The PVC ensures data survives pod restarts.
    * **Orchestration:** The reconciler must create resources in the correct order (Secret and PVC before the
      Deployment) and ensure they all conform to the spec.
    * **Status Reporting:** The `.status` field should now report the name of the `Service`, the name of the `Secret`
      holding the password, and whether the database is ready.

***

### Phase 3: The External Service Operator (`DomainManager`)

This operator steps outside the cluster to manage resources in an external system via an API. This is a very common and
powerful pattern.

* **Project:** A `DomainManager` operator that manages DNS records with a cloud provider (e.g., AWS Route 53,
  Cloudflare).
* **CRD Spec:** The `DomainManager` spec could include `domainName`, `recordType` (A, CNAME), `value` (like an IP
  address), and `ttl`.
* **Operator Logic:**
    1. The operator needs credentials for the external API, which should be stored in a Kubernetes `Secret`.
    2. When a `DomainManager` resource is created, the operator uses an API client (e.g., the AWS Go SDK) to create a
       DNS record.
    3. The reconcile loop must constantly check if the external resource exists and matches the spec. If the spec
       changes in Kubernetes, the operator must update the record via the API.
    4. When the resource is deleted, the operator (using a **finalizer**) must call the API to delete the DNS record.
* **New Concepts Learned:**
    * **Interacting with External APIs:** Integrating third-party Go clients into your operator.
    * **Credential Management:** Securely loading and using API keys from Kubernetes Secrets.
    * **Managing External State:** The reconciliation logic now has to compare the "desired state" in the cluster with
      the "actual state" in an external system.
    * **Advanced Error Handling:** API calls can fail due to network issues, rate limiting, or invalid credentials. Your
      reconciler must handle these errors gracefully, likely by returning an error with `RequeueAfter` to retry after a
      delay.

***

### Phase 4: Integrating with Your Observability Stack 🚀

This final phase ties everything together. You'll instrument your operators to take full advantage of the monitoring
stack you already have set up.

* **Project:** Add observability features to all your operators (`HelloWorld`, `SimpleDB`, `DomainManager`).
* **Action Steps & Concepts Learned:**
    1. **Custom Metrics (Mimir):** Use the `controller-runtime/pkg/metrics` package to expose custom Prometheus metrics.
        * **Example:** For the `SimpleDB` operator, create a Gauge metric to track the total number of `SimpleDB`
          instances managed (`simpledb_managed_instances`). For the `DomainManager`, create a Counter for successful and
          failed API calls (`domainmanager_api_calls_total{status="success"}`).
        * **Benefit:** In Grafana, you can build dashboards to visualize your operators' health and activity.
    2. **Structured Logging (Loki/Mimir):** Ensure all your logging statements are structured using
       `log.WithValues("resource_name", req.Name, "namespace", req.Namespace)`.
        * **Benefit:** Your logs, collected by Alloy, become searchable and filterable in Grafana. You can easily find
          all log entries for a specific reconciliation of a specific resource, which is invaluable for debugging.
    3. **Distributed Tracing (Tempo/Mimir):** Instrument your `Reconcile` function with OpenTelemetry spans. You can
       create a parent span for the entire reconciliation and child spans for major operations (e.g., "
       get_deployment", "call_external_api", "update_status").
        * **Benefit:** In Grafana, you can visualize the entire lifecycle of a single reconcile request. This helps you
          pinpoint performance bottlenecks—for example, you might discover that a specific API call is consistently
          slow.
    4. **Profiling (Pyroscope):** With Alloy and Pyroscope already running, your operator's Go runtime is likely already
       being profiled. Go into the Pyroscope UI and inspect the CPU and memory profiles for your controller-manager.
        * **Benefit:** You can identify "hot spots" in your code—functions that are consuming a lot of CPU or allocating
          too much memory—and optimize them.
    5. **Alerting:** Set up Prometheus alerts based on your custom metrics. For example, alert if the number of failed
       API calls in the `DomainManager` exceeds a threshold within a time window.
        * **Benefit:** You get proactive notifications about issues with your operators before they impact users.