# go-namecheck

Source code analyzer that helps you to maintain variable/field naming conventions inside your project.

## Quick start / Installation

To install `go-namecheck` binary under your `$(go env GOPATH)/bin`:

```bash
go get -v github.com/Quasilyte/go-namecheck
```

If `$GOPATH/bin` is under your system `$PATH`, `go-namecheck` command should be available after that.<br>
This should print the help message:

```bash
go-namecheck --help
```

In big teams, same things end up being called differently eventually.
Sometimes you bring inconsistencies on your own.
Suppose it's considered idiomatic to call `string` parameter `s` if
you can't figure a more descriptive name, but sometimes you see `str`
names used by other programmers from your team.
This is where `go-namecheck` can help.

For a better illustration, suppose we also want to catch regexp
variables that use `re` prefix and propose `RE` suffix instead,
so `var reFoo *regexp.Regexp` becomes `var fooRE *regexp.Regexp`.

```json
{
  "string": {"param": {"str": "s"}},
  "regexp\\.Regexp": {
    "local+global": {"^re[A-Z]\\w*$": "use RE suffix instead of re prefix"}
  }
}
```

Rules above implement checks we described.

First key describes regular expression that matches a type.
For that key there is an object for scopes.
Scope can be one of:

* `param` - function input params
* `receiver` - method receiver
* `global` - any global constant or variable
* `local` - any local constant or variable
* `field` - struct field

You can combine several scopes like `param+receiver+local`, etc.

Inside a scope there is an JSON object that maps "from" => "to" pair.
In the simplest form, it's a simple literal matching that suggests
to replace one name with another, like in `str`=>`s` rule.
Key can also be a regular expression, in this case, the "to" part
does not describe exact substitution, but rather describes
how to make name idiomatic (what to change).

You start by creating your rules file (or borrowing someone else set).
Then you can run `go-namecheck` like this:

```bash
go-namecheck foo.go bar.go mypkg
```

You can also use `std`, `./...` and other conventional targets that are normally
understood by Go tools.
