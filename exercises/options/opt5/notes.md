## opt5 — paging with Limit and Continue

```go
cont := ""
for {
	list, err := cs.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{
		LabelSelector: "paged=yes",
		Limit:         5,
		Continue:      cont,
	})
	...
	cont = list.Continue
	if cont == "" {
		break
	}
}
```

**Why it works**

- `Limit` caps how many items come back in one response. The server then puts a
  **continue token** in `list.metadata.continue` — an opaque, server-side cursor
  encoding "resume from this key at this resourceVersion".
- Feeding it back as `ListOptions.Continue` resumes exactly where the last page
  stopped. An **empty** token in the response means that was the final page,
  which is the loop's termination condition. Don't try to detect the end with
  `len(list.Items) < limit` — a page can legitimately come back short.
- The token is opaque. Never parse, construct or persist one across processes.

**Key detail:** paging is not a snapshot in the way a single `List` is, but it
is not incoherent either. All pages are served at the resourceVersion pinned by
the *first* request, so you get a consistent view — at the price that the
server must keep that revision available. If your paging loop is slow enough
that the revision is compacted out of etcd, the next page fails with
`410 Gone` and `apierrors.IsResourceExpired(err)` is how you detect it. Recovery
is to start the whole listing again (or opt into
`ResourceVersionMatch: NotOlderThan` and accept a less consistent view).

Two things that follow from this and matter in practice:

- **Set `Limit` on any list that could be large.** An unbounded `List` of a big
  resource makes the apiserver materialise and serialise the entire collection
  in memory. This is the classic way a well-meaning controller degrades a
  cluster for everyone.
- **Informers do this for you.** A `Reflector`'s initial list is chunked
  automatically, so "use an informer" also buys you paging.

When you only need names or labels — a common case for a garbage-collector
style loop — pair paging with a **metadata-only** client
(`k8s.io/client-go/metadata`), which requests `PartialObjectMetadata` and drops
spec/status on the wire entirely.

**References**

- Retrieving large results in chunks: https://kubernetes.io/docs/reference/using-api/api-concepts/#retrieving-large-results-sets-in-chunks
- `metav1.ListOptions`: https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#ListOptions
- `apierrors.IsResourceExpired`: https://pkg.go.dev/k8s.io/apimachinery/pkg/api/errors#IsResourceExpired
- Metadata-only client: https://pkg.go.dev/k8s.io/client-go/metadata
