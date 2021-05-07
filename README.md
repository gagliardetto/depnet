# depnet

`depnet` is a CLI and library for retrieving GitHub Dependency Network info.

It currently only supports retrieving dependents (i.e. repositories/packages that **depend** on a given repo/package).

## Examples

```bash
# Get info/stats on dependents of a repository:
depnet --info --pretty eslint/eslint

# Get the first 3 dependent repositories for `eslint/eslint`:
depnet --type=REPOSITORY --json --limit=3 eslint/eslint

# Get the first 3 dependent packages for `eslint/eslint`:
depnet --type=PACKAGE --json --limit=3 eslint/eslint

# Get the first 3 dependent packages for `eslint/eslint`, specifying a package (see info output):
depnet --type=PACKAGE --json --limit=3 --pkg="eslint-config-eslint" eslint/eslint

# Get the first 10 dependent repositories of `eslint/eslint`, with newline output:
depnet --type=REPOSITORY --limit=10 eslint/eslint

# Get the first 3 dependent packages for `eslint/eslint`, but one JSON object per line (no pretty-printing):
depnet --type=REPOSITORY --json --limit=3 eslint/eslint

# Get the first dependent package for `eslint/eslint`, and add repository info from GitHub:
# NOTE: need a github token (you can set it with $GH_TOKEN env var).
depnet --type=REPOSITORY --json --rich --limit=1 eslint/eslint

# Same as above, but pretty-printed so you can see the JSON format:
depnet --type=REPOSITORY --json --rich --pretty --limit=1 eslint/eslint
```
