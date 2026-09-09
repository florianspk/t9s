# Arch Linux packaging

Two PKGBUILDs, both building from source with the distro's Go toolchain and
injecting the version into `main.version`:

| File | Package | Source |
|---|---|---|
| `PKGBUILD` | `t9s` | the `v$pkgver` release tarball |
| `PKGBUILD-git` | `t9s-git` | the git tip; version derived from `git describe` |

## Build locally

```bash
cd packaging/aur
makepkg -si                      # release build (needs a v<version> tag to exist)
cp PKGBUILD-git /tmp/b/PKGBUILD  # or the git build
```

`makepkg` runs `go test ./...` in `check()`; pass `--nocheck` to skip it.

## Publishing to the AUR

1. Tag and push a release so the tarball exists.
2. Bump `pkgver`, then `updpkgsums` to fill in `sha256sums` (the checked-in value
   is `SKIP`).
3. `makepkg --printsrcinfo > .SRCINFO`, commit both files to the AUR repo.

## Runtime dependencies

`talosctl` must be on `$PATH` — t9s drives it for everything. It is not in the
official repositories, so it sits in `optdepends` rather than `depends`; install
it from the AUR (`talosctl` / `talosctl-bin`) or from the Sidero release page.
`crane` is optional and only needed to browse the extension catalog.
