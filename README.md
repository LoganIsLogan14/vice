# Tower settings deduplication — Pass 1C

This pass changes only `cmd/vice/ui.go`.

It prevents `TowerCabPane` from being added to the Settings window twice when it is already the active pane in Tower mode. Tower Cab settings remain available while STARS or ERAM is active.

Apply from the VICE repository root:

```bash
cp -a cmd/vice/ui.go cmd/vice/ui.go.before-pass1c
tar -xzf ~/Downloads/vice-tower-settings-pass1c.tar.gz --strip-components=1
gofmt -w cmd/vice/ui.go
./build.sh
```
