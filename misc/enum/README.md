# Go+ editor tooling

## Install Go+

On Windows amd64, download `goplus-installer-windows-amd64.exe` from the
[releases page](https://github.com/RestartFU/goplus/releases) and run it. The
installer downloads the matching precompiled Go+ archive, installs it under
`%LOCALAPPDATA%\GoPlus`, and adds its `bin` directory to the user PATH. It does
not require Go or Git. Pass `-prefix C:\path\to\GoPlus` to choose another
location or `-no-path-update` to leave the user PATH unchanged.

### Build from source

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

The patches add the Go+ AST nodes to x/tools' optimized inspector and make
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

`try` is a contextual propagation statement. It binds every result except a
final `error` or `bool`. A non-nil error or false boolean returns zero values
for the enclosing function's earlier results and propagates the final result:

```go
func loadUser() (User, error) { /* ... */ }

func currentUser() (User, error) {
	try user := loadUser()
	return user, nil
}
```

The called expression's final result and the enclosing function's final result
must both be `error` or both be `bool`. This makes comma-ok expressions concise:

```go
func displayName(values map[string]User, name string) (string, bool) {
	try user := values[name]
	return user.DisplayName(), true
}
```

Multiple success values may be bound with `try value, count := load()`.
Because `try` is contextual, ordinary uses such as `try := 1` remain valid.

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
