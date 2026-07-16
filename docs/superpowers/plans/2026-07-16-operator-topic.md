# Operator Topic (controller-runtime) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a kind-backed `operator` topic with three controller-runtime exercises (op1-3) teaching the Reconcile contract, Owns/ownerRefs, and CreateOrUpdate idempotency.

**Architecture:** Each exercise is a single `main.go` following the existing exkit pattern: `exkit.Begin` provisions a fresh `clx-<name>` namespace, the exercise builds a real controller-runtime Manager scoped to that namespace, starts it in a goroutine, drives objects with the plain client-go clientset, and asserts the reconciler's side effects with `exkit.WaitFor`/`Assert*`. Broken exercise versions carry the `I AM NOT DONE` marker; solutions are the fixed twins at the same relative path (the e2e task overlays `solutions/` onto `exercises/`).

**Tech Stack:** Go 1.26, k8s.io/* v0.36.2, sigs.k8s.io/controller-runtime (v0.24.x line — must not bump the k8s.io pins), kind cluster `kind-kubeclientlings`.

## Global Constraints

- k8s.io/api, apimachinery, client-go, apiextensions-apiserver stay at `v0.36.2` in go.mod.
- Go module is `github.com/madhank93/kubeclientlings`; exkit import is `github.com/madhank93/kubeclientlings/internal/exkit`.
- exkit is NOT modified — manager wiring lives visibly in each exercise file.
- Exercise names: `op1`, `op2`, `op3`. Paths: `exercises/operator/opN/main.go` + `solutions/operator/opN/main.go` (identical relative paths, required by `mise run e2e`'s overlay).
- Broken versions MUST compile (`mode = "compile"`) and carry `// I AM NOT DONE` in the header comment; solutions must NOT contain that marker.
- info.toml entries are inserted after the `ctrl3` block and before the `wq1` block (order = curriculum order).
- All verification of exercise behavior runs against the kind cluster: bring it up with `mise run up` if `mise run doctor` complains.
- Commit after every task. Work on branch `operator-topic`.

---

### Task 1: Branch + controller-runtime dependency

**Files:**
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Produces: `sigs.k8s.io/controller-runtime` importable by all later tasks (packages used later: the root `ctrl` alias, `pkg/cache`, `pkg/client`, `pkg/controller/controllerutil`, `pkg/metrics/server`).

- [ ] **Step 1: Create the branch**

```bash
git checkout -b operator-topic
```

- [ ] **Step 2: Add the dependency**

```bash
go get sigs.k8s.io/controller-runtime@latest
go mod tidy
```

- [ ] **Step 3: Verify the k8s.io pins did not move**

Run: `grep -E 'k8s.io/(api|apimachinery|client-go|apiextensions-apiserver) v' go.mod`
Expected: all four still `v0.36.2`.

If `@latest` bumped them, pick the controller-runtime minor that pairs with k8s 1.36 instead (the pairing rule is CR minor = k8s minor − 12, i.e. v0.24.x):

```bash
git checkout -- go.mod go.sum
go get sigs.k8s.io/controller-runtime@v0.24.0
go mod tidy
```

then re-run the grep and confirm `v0.36.2`.

- [ ] **Step 4: Confirm the tool still builds and tests pass**

Run: `mise run build && mise run test`
Expected: build succeeds, all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum
git commit -m "build: add sigs.k8s.io/controller-runtime for the operator topic"
```

---

### Task 2: op1 — the Reconcile contract

**Files:**
- Create: `solutions/operator/op1/main.go`
- Create: `exercises/operator/op1/main.go`
- Modify: `info.toml` (insert after the `ctrl3` entry, before `wq1`)

**Interfaces:**
- Consumes: `exkit.Begin(name) (ctx, cancel, cs kubernetes.Interface, ns string)`, `exkit.MustRESTConfig() *rest.Config`, `exkit.WaitFor(ctx, desc, cond)`, `exkit.AssertEqual`, `exkit.Failf`, `exkit.Successf` — all existing, unchanged.
- Produces: the manager-boilerplate shape (SetLogger → NewManager with Scheme/Metrics off/Cache scoped to ns → NewControllerManagedBy → `go mgr.Start(ctx)` → `mgr.GetCache().WaitForCacheSync(ctx)`) that op2/op3 repeat verbatim.

- [ ] **Step 1: Ensure the cluster is up**

Run: `mise run doctor`
If it reports the cluster missing: `mise run up`

- [ ] **Step 2: Write the solution**

Create `solutions/operator/op1/main.go`:

```go
// op1
//
// Your first controller-runtime Reconciler. The manager owns all the
// machinery you built by hand in ctrl2 (informer → queue → worker); you only
// write Reconcile. The contract: a Request names one object, you return a
// Result plus error. NotFound is NORMAL — the object was deleted and the
// cache already forgot it. Returning an error there makes the manager retry
// a reconcile that can never succeed.
//
// Fetch the object the request names, and treat NotFound as "done".
package main

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

// labeler stamps reconciled=true on every ConfigMap it sees, and counts what
// happened so the exercise can assert on it.
type labeler struct {
	client.Client
	seen atomic.Int64 // reconciles entered
	errs atomic.Int64 // unexpected failures
}

func (r *labeler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.seen.Add(1)

	// The request already carries the object's full namespace/name key.
	var cm corev1.ConfigMap
	if err := r.Get(ctx, req.NamespacedName, &cm); err != nil {
		if apierrors.IsNotFound(err) {
			// The object is gone; there is nothing left to reconcile.
			return ctrl.Result{}, nil
		}
		r.errs.Add(1)
		return ctrl.Result{}, err
	}
	if cm.Labels["reconciled"] == "true" {
		return ctrl.Result{}, nil
	}

	// MergeFrom-patch instead of Update: idempotent and immune to the
	// stale-cache conflict a second event would otherwise cause.
	patch := client.MergeFrom(cm.DeepCopy())
	if cm.Labels == nil {
		cm.Labels = map[string]string{}
	}
	cm.Labels["reconciled"] = "true"
	if err := r.Patch(ctx, &cm, patch); err != nil {
		r.errs.Add(1)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func main() {
	ctx, cancel, cs, ns := exkit.Begin("op1")
	defer cancel()

	ctrl.SetLogger(logr.Discard()) // keep exercise output clean

	mgr, err := ctrl.NewManager(exkit.MustRESTConfig(), ctrl.Options{
		Scheme:  scheme.Scheme,                              // client-go's built-in types
		Metrics: metricsserver.Options{BindAddress: "0"},    // no metrics port
		Cache:   cache.Options{DefaultNamespaces: map[string]cache.Config{ns: {}}},
	})
	if err != nil {
		exkit.Failf("building manager: %v", err)
	}

	r := &labeler{Client: mgr.GetClient()}
	if err := ctrl.NewControllerManagedBy(mgr).For(&corev1.ConfigMap{}).Complete(r); err != nil {
		exkit.Failf("wiring controller: %v", err)
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			exkit.Failf("manager exited: %v", err)
		}
	}()
	if !mgr.GetCache().WaitForCacheSync(ctx) {
		exkit.Failf("manager cache never synced")
	}

	if _, err := cs.CoreV1().ConfigMaps(ns).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "app-config", Namespace: ns},
	}, metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating configmap: %v", err)
	}

	exkit.WaitFor(ctx, "the reconciler to label app-config", func(ctx context.Context) (bool, error) {
		if n := r.errs.Load(); n > 0 {
			return false, fmt.Errorf("the reconciler reported %d error(s) — check how Get and its error are handled", n)
		}
		cm, err := cs.CoreV1().ConfigMaps(ns).Get(ctx, "app-config", metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return cm.Labels["reconciled"] == "true", nil
	})

	// Now the delete: the reconciler gets one more request for an object
	// that no longer exists. A correct Reconcile treats that as success.
	before := r.seen.Load()
	if err := cs.CoreV1().ConfigMaps(ns).Delete(ctx, "app-config", metav1.DeleteOptions{}); err != nil {
		exkit.Failf("deleting configmap: %v", err)
	}
	exkit.WaitFor(ctx, "the delete event to reach the reconciler", func(ctx context.Context) (bool, error) {
		return r.seen.Load() > before, nil
	})
	exkit.AssertEqual("reconcile errors after the delete", r.errs.Load(), int64(0))

	exkit.Successf("Request in, Result out, NotFound is normal: that is the whole Reconcile contract")
}
```

- [ ] **Step 3: Run the solution against kind — expect success**

Run: `go run ./solutions/operator/op1`
Expected: `✓` lines then `🎉 Request in, Result out, NotFound is normal…`, exit 0.

- [ ] **Step 4: Write the broken exercise**

Create `exercises/operator/op1/main.go` — identical to the solution EXCEPT the header comment (marker added), the import list (`types` added, `apierrors` removed), and the top of `Reconcile`:

Header comment ends with:

```go
// Fetch the object the request names, and treat NotFound as "done".
//
// I AM NOT DONE
package main
```

Imports: remove `apierrors "k8s.io/apimachinery/pkg/api/errors"`, add `"k8s.io/apimachinery/pkg/types"`.

`Reconcile` fetch block becomes:

```go
	// A Request already carries the object's full namespace/name — but this
	// key is built by hand and loses the namespace, and EVERY error (even
	// "it's gone") is treated as a failure to retry forever.
	var cm corev1.ConfigMap
	key := types.NamespacedName{Name: req.Name}
	if err := r.Get(ctx, key, &cm); err != nil {
		r.errs.Add(1)
		return ctrl.Result{}, err
	}
```

Everything else (patch block, `main`) is byte-identical to the solution.

- [ ] **Step 5: Run the broken version — expect fast failure**

Run: `go run ./exercises/operator/op1`
Expected: fails within a few seconds with `❌ timed out waiting for the reconciler to label app-config: the reconciler reported N error(s)…`, exit 1.

- [ ] **Step 6: Register op1 in info.toml**

Insert after the closing of the `ctrl3` entry (its hint block ends with `RetryPeriod:     2 * time.Second,"""`), before `[[exercises]]` / `name = "wq1"`:

```toml
[[exercises]]
name = "op1"
path = "exercises/operator/op1/main.go"
mode = "compile"
hint = """
Two fixes, one lesson — the Reconcile contract:

1. The request already carries the full key. Fetch with it:

       if err := r.Get(ctx, req.NamespacedName, &cm); err != nil {

2. NotFound is not a failure — the object was deleted and the work is
   simply gone. Swallow it and report success:

       if apierrors.IsNotFound(err) {
           return ctrl.Result{}, nil
       }
       r.errs.Add(1)
       return ctrl.Result{}, err"""
```

(op1 needs `apierrors "k8s.io/apimachinery/pkg/api/errors"` imported for the fix — the hint shows the calls; the solution file is the reference.)

- [ ] **Step 7: Confirm the runner sees the new exercise**

Run: `mise run build && ./bin/kubeclientlings list | grep op1`
Expected: op1 listed (pending/not-done state).

- [ ] **Step 8: Confirm tool tests still pass**

Run: `mise run test`
Expected: PASS (exercise count is computed live; no assertions should break).

- [ ] **Step 9: Commit**

```bash
git add solutions/operator/op1 exercises/operator/op1 info.toml
git commit -m "feat: add op1 exercise — controller-runtime Reconcile contract"
```

---

### Task 3: op2 — Owns and ownership

**Files:**
- Create: `solutions/operator/op2/main.go`
- Create: `exercises/operator/op2/main.go`
- Modify: `info.toml` (insert after the new `op1` entry)

**Interfaces:**
- Consumes: same exkit helpers as Task 2; same manager boilerplate shape (SetLogger → NewManager → builder → goroutine Start → WaitForCacheSync).
- Produces: nothing consumed later.

- [ ] **Step 1: Write the solution**

Create `solutions/operator/op2/main.go`:

```go
// op2
//
// Owns() is how a controller notices its children. It maps an event on a
// child object back to a reconcile request for the OWNER — by reading the
// child's controller ownerRef. No ownerRef, no mapping: delete the child and
// nobody hears it. SetControllerReference is what stamps that ref.
//
// Stamp the parent's controller reference on the child before creating it.
package main

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

// childMinder makes sure the ConfigMap named "parent" always has its child
// Secret. The child is stamped with a controller ownerRef so Owns() can map
// child events back to the parent.
type childMinder struct {
	client.Client
	scheme *runtime.Scheme
}

func (r *childMinder) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var parent corev1.ConfigMap
	if err := r.Get(ctx, req.NamespacedName, &parent); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// Every namespace also holds a kube-root-ca.crt ConfigMap; this
	// controller only manages the one named "parent".
	if parent.Name != "parent" {
		return ctrl.Result{}, nil
	}

	child := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "parent-child", Namespace: parent.Namespace},
		StringData: map[string]string{"owner": parent.Name},
	}
	// The ownerRef is the whole story: it is what Owns(&corev1.Secret{})
	// reads to turn a child event into a request for the parent.
	if err := controllerutil.SetControllerReference(&parent, child, r.scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Create(ctx, child); err != nil && !apierrors.IsAlreadyExists(err) {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func main() {
	ctx, cancel, cs, ns := exkit.Begin("op2")
	defer cancel()

	ctrl.SetLogger(logr.Discard()) // keep exercise output clean

	mgr, err := ctrl.NewManager(exkit.MustRESTConfig(), ctrl.Options{
		Scheme:  scheme.Scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
		Cache:   cache.Options{DefaultNamespaces: map[string]cache.Config{ns: {}}},
	})
	if err != nil {
		exkit.Failf("building manager: %v", err)
	}

	r := &childMinder{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Complete(r); err != nil {
		exkit.Failf("wiring controller: %v", err)
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			exkit.Failf("manager exited: %v", err)
		}
	}()
	if !mgr.GetCache().WaitForCacheSync(ctx) {
		exkit.Failf("manager cache never synced")
	}

	if _, err := cs.CoreV1().ConfigMaps(ns).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "parent", Namespace: ns},
	}, metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating parent configmap: %v", err)
	}

	exkit.WaitFor(ctx, "the controller to create the child secret", func(ctx context.Context) (bool, error) {
		_, err := cs.CoreV1().Secrets(ns).Get(ctx, "parent-child", metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return err == nil, err
	})

	child, err := cs.CoreV1().Secrets(ns).Get(ctx, "parent-child", metav1.GetOptions{})
	if err != nil {
		exkit.Failf("fetching child secret: %v", err)
	}
	ref := metav1.GetControllerOf(child)
	exkit.AssertTrue("child carries a controller ownerRef pointing at parent",
		ref != nil && ref.Kind == "ConfigMap" && ref.Name == "parent")

	// The payoff: kill the child. Its delete event carries the ownerRef,
	// Owns maps it to "parent", and the reconciler puts the child back.
	oldUID := child.UID
	if err := cs.CoreV1().Secrets(ns).Delete(ctx, "parent-child", metav1.DeleteOptions{}); err != nil {
		exkit.Failf("deleting child secret: %v", err)
	}
	exkit.WaitFor(ctx, "the controller to recreate the deleted child", func(ctx context.Context) (bool, error) {
		s, err := cs.CoreV1().Secrets(ns).Get(ctx, "parent-child", metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return s.UID != oldUID, nil
	})

	exkit.Successf("Owns + ownerRefs: delete a child and the controller puts it right back")
}
```

- [ ] **Step 2: Run the solution against kind — expect success**

Run: `go run ./solutions/operator/op2`
Expected: `✓ child carries a controller ownerRef…`, recreation wait passes, `🎉`, exit 0.

- [ ] **Step 3: Write the broken exercise**

Create `exercises/operator/op2/main.go` — identical to the solution EXCEPT:

Header comment ends with:

```go
// Stamp the parent's controller reference on the child before creating it.
//
// I AM NOT DONE
package main
```

Remove the import `"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"` (unused in the broken version).

In `Reconcile`, the SetControllerReference block is replaced by a comment:

```go
	// The child is created bare — nothing records who owns it. Owns() maps a
	// child event to its owner by reading the child's controller ownerRef…
	// which this child never gets. What helper stamps it? (You will need the
	// manager's Scheme — it is already on r.scheme.)
	if err := r.Create(ctx, child); err != nil && !apierrors.IsAlreadyExists(err) {
		return ctrl.Result{}, err
	}
```

Everything else is byte-identical to the solution. Note: the `scheme` field on `childMinder` stays (unused struct fields compile fine).

- [ ] **Step 4: Run the broken version — expect failure at the ownerRef assert**

Run: `go run ./exercises/operator/op2`
Expected: child gets created, then `❌ child carries a controller ownerRef pointing at parent: expected true`, exit 1 (fast — no timeout involved).

- [ ] **Step 5: Register op2 in info.toml**

Insert directly after the new `op1` entry:

```toml
[[exercises]]
name = "op2"
path = "exercises/operator/op2/main.go"
mode = "compile"
hint = """
Owns(&corev1.Secret{}) maps a child event back to its owner by reading the
child's controller ownerRef — a bare child is invisible to it. Stamp the ref
before creating (controllerutil is sigs.k8s.io/controller-runtime/pkg/controller/controllerutil):

    if err := controllerutil.SetControllerReference(&parent, child, r.scheme); err != nil {
        return ctrl.Result{}, err
    }
    if err := r.Create(ctx, child); err != nil && !apierrors.IsAlreadyExists(err) {"""
```

- [ ] **Step 6: Confirm runner + tests**

Run: `mise run build && ./bin/kubeclientlings list | grep op2 && mise run test`
Expected: op2 listed; tests PASS.

- [ ] **Step 7: Commit**

```bash
git add solutions/operator/op2 exercises/operator/op2 info.toml
git commit -m "feat: add op2 exercise — Owns and SetControllerReference"
```

---

### Task 4: op3 — CreateOrUpdate idempotency

**Files:**
- Create: `solutions/operator/op3/main.go`
- Create: `exercises/operator/op3/main.go`
- Modify: `info.toml` (insert after the new `op2` entry)

**Interfaces:**
- Consumes: same exkit helpers; `exkit.NginxDeployment(ns, name string, replicas int32) *appsv1.Deployment` fixture; same manager boilerplate shape.
- Produces: nothing consumed later.

- [ ] **Step 1: Write the solution**

Create `solutions/operator/op3/main.go`:

```go
// op3
//
// controllerutil.CreateOrUpdate is the reconciler's read-modify-write: it
// GETS the live object into your struct, runs your mutate() to lay desired
// state over it, then Creates or Updates only if something changed. Desired
// state set OUTSIDE mutate() is wiped by that Get — creation works once, but
// the update path becomes a no-op and drift is never repaired.
//
// Move the desired state (and the ownerRef) into the mutate function.
package main

import (
	"context"
	"strconv"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/madhank93/kubeclientlings/internal/exkit"
)

// infoWriter mirrors each Deployment's replica count into a <name>-info
// ConfigMap, owned by the Deployment so drift on the child re-triggers it.
type infoWriter struct {
	client.Client
	scheme *runtime.Scheme
}

func (r *infoWriter) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var dep appsv1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &dep); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name:      dep.Name + "-info",
		Namespace: dep.Namespace,
	}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		// This runs on BOTH paths — after a NotFound (about to Create) and
		// after a successful Get (cm now holds the LIVE object). Lay the
		// desired state over whatever is there.
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		cm.Data["replicas"] = strconv.Itoa(int(*dep.Spec.Replicas))
		return controllerutil.SetControllerReference(&dep, cm, r.scheme)
	})
	return ctrl.Result{}, err
}

func main() {
	ctx, cancel, cs, ns := exkit.Begin("op3")
	defer cancel()

	ctrl.SetLogger(logr.Discard()) // keep exercise output clean

	mgr, err := ctrl.NewManager(exkit.MustRESTConfig(), ctrl.Options{
		Scheme:  scheme.Scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
		Cache:   cache.Options{DefaultNamespaces: map[string]cache.Config{ns: {}}},
	})
	if err != nil {
		exkit.Failf("building manager: %v", err)
	}

	r := &infoWriter{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r); err != nil {
		exkit.Failf("wiring controller: %v", err)
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			exkit.Failf("manager exited: %v", err)
		}
	}()
	if !mgr.GetCache().WaitForCacheSync(ctx) {
		exkit.Failf("manager cache never synced")
	}

	if _, err := cs.AppsV1().Deployments(ns).Create(ctx, exkit.NginxDeployment(ns, "web", 1), metav1.CreateOptions{}); err != nil {
		exkit.Failf("creating deployment: %v", err)
	}

	replicasIs := func(want string) func(context.Context) (bool, error) {
		return func(ctx context.Context) (bool, error) {
			cm, err := cs.CoreV1().ConfigMaps(ns).Get(ctx, "web-info", metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			return cm.Data["replicas"] == want, nil
		}
	}

	// Create path.
	exkit.WaitFor(ctx, "web-info to record replicas=1", replicasIs("1"))

	// Update path: scale the deployment, the mirror must follow.
	if _, err := cs.AppsV1().Deployments(ns).Patch(ctx, "web",
		types.StrategicMergePatchType, []byte(`{"spec":{"replicas":3}}`), metav1.PatchOptions{}); err != nil {
		exkit.Failf("scaling deployment: %v", err)
	}
	exkit.WaitFor(ctx, "web-info to follow the scale-up (replicas=3)", replicasIs("3"))

	// Drift repair: tamper with the child, the ownerRef event brings the
	// reconciler back and CreateOrUpdate restores desired state.
	if _, err := cs.CoreV1().ConfigMaps(ns).Patch(ctx, "web-info",
		types.StrategicMergePatchType, []byte(`{"data":{"replicas":"999"}}`), metav1.PatchOptions{}); err != nil {
		exkit.Failf("tampering with web-info: %v", err)
	}
	exkit.WaitFor(ctx, "the controller to repair the drifted ConfigMap", replicasIs("3"))

	exkit.Successf("CreateOrUpdate with a real mutate fn: create, update, and drift repair from one code path")
}
```

- [ ] **Step 2: Run the solution against kind — expect success**

Run: `go run ./solutions/operator/op3`
Expected: three WaitFor lines pass, `🎉`, exit 0.

- [ ] **Step 3: Write the broken exercise**

Create `exercises/operator/op3/main.go` — identical to the solution EXCEPT:

Header comment ends with:

```go
// Move the desired state (and the ownerRef) into the mutate function.
//
// I AM NOT DONE
package main
```

The `CreateOrUpdate` block in `Reconcile` becomes:

```go
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name:      dep.Name + "-info",
		Namespace: dep.Namespace,
	}}
	// Desired state prepared up front… but CreateOrUpdate GETS the live
	// object into cm first, wiping these fields before the (empty) mutate fn
	// runs. Creation works once; the update path never updates anything.
	cm.Data = map[string]string{"replicas": strconv.Itoa(int(*dep.Spec.Replicas))}
	if err := controllerutil.SetControllerReference(&dep, cm, r.scheme); err != nil {
		return ctrl.Result{}, err
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		return nil
	})
	return ctrl.Result{}, err
```

Everything else is byte-identical to the solution.

- [ ] **Step 4: Run the broken version — expect failure at the scale-up wait**

Run: `go run ./exercises/operator/op3`
Expected: `web-info to record replicas=1` passes (create path works), then `web-info to follow the scale-up (replicas=3)` never turns true → `❌ timed out waiting…` when the 90s Begin deadline expires, exit 1. NOTE: this failure takes the full remaining ~80s — that is the honest failure mode (absence of an update), same as ctrl2's broken timeout.

- [ ] **Step 5: Register op3 in info.toml**

Insert directly after the new `op2` entry:

```toml
[[exercises]]
name = "op3"
path = "exercises/operator/op3/main.go"
mode = "compile"
hint = """
CreateOrUpdate GETS the live object into cm before your mutate fn runs, so
anything set on cm beforehand is wiped on the update path. Desired state
belongs INSIDE mutate — it runs on both the create and the update path:

    _, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
        if cm.Data == nil {
            cm.Data = map[string]string{}
        }
        cm.Data["replicas"] = strconv.Itoa(int(*dep.Spec.Replicas))
        return controllerutil.SetControllerReference(&dep, cm, r.scheme)
    })"""
```

- [ ] **Step 6: Confirm runner + tests**

Run: `mise run build && ./bin/kubeclientlings list | grep op3 && mise run test`
Expected: op3 listed; tests PASS.

- [ ] **Step 7: Commit**

```bash
git add solutions/operator/op3 exercises/operator/op3 info.toml
git commit -m "feat: add op3 exercise — CreateOrUpdate mutate-fn idempotency"
```

---

### Task 5: Full verification gate + follow-through

**Files:**
- Modify: none expected (fix-ups only if gates fail)
- Modify: `/Users/madhan/.claude/projects/-Volumes-work-git-repos-learn-client-go/memory/v2-spec-harvest.md` (memory, outside repo)

- [ ] **Step 1: Static gates**

Run: `mise run build && mise run test && mise run lint && go vet ./solutions/operator/... ./exercises/operator/...`
Expected: all green (broken exercises compile — that is what mode="compile" demands).

- [ ] **Step 2: Full e2e (solutions overlaid onto exercises, verified against kind)**

Run: `mise run e2e`
Expected: all 52 exercises pass. Known pre-existing flake: ctrl3 namespace-teardown timing — if ctrl3 alone fails, re-run it isolated (`go run ./solutions/controllers/ctrl3` after overlay, or re-run e2e) before treating it as a regression.

- [ ] **Step 3: Confirm exercises/ tree restored after e2e**

Run: `git status --short`
Expected: clean (the e2e task does `git checkout -- exercises/` itself).

- [ ] **Step 4: Update project memory**

In `/Users/madhan/.claude/projects/-Volumes-work-git-repos-learn-client-go/memory/v2-spec-harvest.md`: mark the "Deliberately NOT built" controller-runtime paragraph as superseded (operator topic op1-3 built 2026-07-16, kind-backed, dep added) and the justfile infra item as dropped (mise.toml is the sole runner). Update exercise count 49 → 52.

- [ ] **Step 5: Merge**

Per superpowers:finishing-a-development-branch — present merge/PR options to the user. Default expectation (matches previous rename work): merge `operator-topic` into `main` with a merge commit, keep the branch history.

---

## Self-Review Notes

- Spec coverage: op1 (Reconcile contract + error counter + fast-fail cond error) → Task 2; op2 (Owns/SetControllerReference, ownerRef assert, UID-based recreation check) → Task 3; op3 (CreateOrUpdate mutate fn, scale + tamper asserts) → Task 4; dependency + version pin rule → Task 1; verification section → Task 5; justfile drop + memory follow-through → Task 5.
- kube-root-ca.crt: present in every namespace. op1 tolerates it (labeling it is harmless; `seen` counts it but asserts only use deltas/waits). op2 guards by parent name. op3's For() watches Deployments only and Owns() maps via ownerRef, so it is never reconciled.
- Type consistency: `labeler`/`childMinder`/`infoWriter` each defined and used only within their own file; exkit signatures copied from `internal/exkit/exkit.go` as-is.
