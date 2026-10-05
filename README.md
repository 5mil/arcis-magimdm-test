# Arcis + MagiMDM test

Local joint run of the two Windows desks. Not a release.

- Arcis library host: `127.0.0.1:9090` from `5mil/arcis-windows`
- MagiMDM parent desk: `127.0.0.1:8788` from `5mil/magimdm-windows`
- Ease-of-use desk in this repo: `desk/`, tried on `127.0.0.1:8790`

The first Arcis task was to make the MagiMDM server easier for a parent. `/home` returns devices, the four policies, and Arcis status in one call. Students still do not get Arcis. A parent hour log posts to Arcis `/library/ingest`.

Repos were not changed for that trial. This repo is the copy.

```bash
# Arcis, from a checkout of arcis-windows
ARCIS_DATA=/tmp/arcis-test go run ./cmd/arcis --port 9090 --no-window

# this desk
cd desk
MAGIMDM_DATA=/tmp/magi-ease-data go run ./cmd/magimdm --port 8790 --no-window
```

Observed in the trial: Arcis health up, study-pc enrolled on SchoolDay, hour log stored as Arcis library book 3.
