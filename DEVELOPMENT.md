# Development Guide for kubectl-cost

## Building

The build process uses [govvv](https://github.com/ahmetb/govvv) to set info 
for the `version` subcommand until there is
[in-compiler support](https://github.com/golang/go/issues/37475)
for getting version info. If you don't have `govvv` installed, you can always
edit the Makefile to use `go` instead of `govvv`.

Build:

``` sh
make build
```

Install:

If your `GOPATH` is default and you have `/home/USERNAME/go/bin` in your path, you can use `make install`. Otherwise:

``` sh
chmod +x cmd/kubectl-cost/kubectl-cost
cp cmd/kubectl-cost /somewhere/in/your/PATH/kubectl-cost
```

As long as the binary is still named `kubectl-cost` and is somewhere in your `PATH`, it will be usable.

## Releasing

Tag from `main` with a valid SemVer version (e.g. `v0.2.0`) that is after the most recent release. There is a [GitHub Actions workflow](https://github.com/kubecost/kubectl-cost/blob/v0.1.3/.github/workflows/build-release.yaml) that handles building and publishing release binaries and archives to a GitHub release. It will be triggered automatically by any tag pushed that is prefixed with `v`.

Once the release completes, the [Krew manifest](https://github.com/kubernetes-sigs/krew-index/pull/1158) should be updated with the latest version and checksums. The release workflow conveniently also outputs the checksums needed for this update during the build step.
