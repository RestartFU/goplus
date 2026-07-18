# Enum-aware gopls

## Install Go+

The repository installers build a private full GOROOT and the patched editor
tooling. They expose four commands without replacing an existing Go install:
`go+`, `gofmt+`, `gopls+`, and `goimports+`. The `gopls+` and `goimports+`
launchers always use the private Go+ toolchain.

On Linux:

```sh
./install-goplus.sh
```

The default GOROOT is `$HOME/.local/share/goplus/go`, with command links in
`$HOME/.local/bin`. Use `--prefix` and `--bin-dir` to change them.

On Windows PowerShell:

```powershell
.\install-goplus.ps1
```

The default installation is `%LOCALAPPDATA%\GoPlus`. The installer adds its
`bin` directory to the user PATH; pass `-NoPathUpdate` to skip that change.
Use `-Prefix C:\path\to\GoPlus` for a custom location.

Both installers require Git and a working Go bootstrap toolchain. They build
this checkout, fetch the pinned x/tools and gopls sources listed below, apply
the compatibility patches, build all four commands, and verify `go+` and
`gopls+` before replacing a previous Go+ installation.

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

Methods belong to the enum type itself, never to an individual variant. A
method that needs variant fields switches on its enum receiver and uses the
case-narrowed receiver inside each arm.

Enums use a contextual `enum` marker in a type declaration, so `enum` remains
available as an ordinary identifier elsewhere:

```go
type Result[T any] enum {
	Ok { Value T }
	Err { Err error }
}
```

Variants are namespaced by their enum when no target type is available:

```go
x := Result.Ok{Value: 1}
y := Option.Some[string]{Value: "ok"}
```

When an assignment or return context already supplies the enum type, the
short constructor is inferred (`var x Result = Ok{}`); enum switch cases also
use short names (`case Ok:`). Variants are constructors only: `Result.Ok{}` is
valid, but `Result.Ok` cannot be used as a field, parameter, alias, or other
standalone type.

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
