VICE Tower refactor Pass 2A

Behavior-preserving changes:
- Adds server/tower.go.
- Moves facility selection, parent ARTCC lookup, scenario-mode inference,
  mode helpers, and facility-selector validation out of server/scenario.go.
- Includes the Pass 1B scenario-mode helper refactor.
- Does not change scenario JSON, simulation behavior, traffic movement, or UI.

Apply from the VICE repository root:

  cp -a server/scenario.go server/scenario.go.before-pass2a
  tar -xzf ~/Downloads/vice-tower-refactor-pass2a.tar.gz --strip-components=1
  gofmt -w server/scenario.go server/tower.go
  ./build.sh
