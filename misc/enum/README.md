# Enum-aware gopls

The Go repository does not contain the gopls module. To build gopls against
this fork, apply both companion patches to compatible `golang.org/x/tools`
and `golang.org/x/tools/gopls` checkouts, then build gopls with this fork as
`GOROOT`.

The patches add the enum AST nodes to x/tools' optimized inspector and make
gopls enum-aware throughout parsing, type-reference indexing, hover,
definition, references, rename, semantic tokens, workspace/document symbols,
fill-switch actions, and narrowed-case completion. Compiler-generated marker
methods are hidden from editor surfaces. The x/tools patch also teaches SSA,
CFG, satisfy, and inline analysis about enum switches and local enum
declarations, and supports shallow export of Go 1.28 generic methods.

They were verified with x/tools commit `635ae9663724` and gopls v0.22.0:

```sh
git -C /path/to/tools apply /path/to/go/misc/enum/x-tools.patch
git -C /path/to/gopls apply /path/to/go/misc/enum/gopls.patch
GOROOT=/path/to/go go build ./...
```

The x/tools inspector changes are also mirrored in `src/cmd/vendor` so the
fork's bundled analyzers and `go vet` work without an external patch. The
x/tools patch also carries a shallow export/import regression test for generic
enums, protecting gopls' persistent package cache.
