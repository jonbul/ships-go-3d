# ships-go-3d

Backend for **Ships 3D**: players design spaceships from 3D primitives in the
browser editor ([ships-vue-3d](../ships-vue-3d)) and fly them against each
other in open space. Go + Gin + MongoDB + gorilla/websocket.

It is a sibling of the 2D game's backend (`ships-go`), not a fork of it: it
shares the **users** and **sessions** collections with it (one account, one
login for both games), but keeps its ship designs in a collection of its own,
`paintingProjects3d`. No NPCs and no admin panel yet.

## Run

```bash
scripts/runDev.sh # backend + ships-vue-3d frontend together; prints the URLs
go run .          # backend only; needs MongoDB and the env vars below
go test ./...
```

Production (the home server): `scripts/deployProd.sh [all|back|front]
[release|snapshot] [-y]` deploys the latest release (or snapshot) of both
repos over SSH and checks the result; see the workspace `CLAUDE.md`,
"Production server".

`scripts/runDev.sh` checks the setup first (shared `.env`, symlinks, free
ports, `SHIPS3D_ALLOWED_ORIGINS`), runs both with prefixed logs, prints the
URLs once both answer, and stops both on Ctrl+C. MongoDB must already be
running.

Listens on the 2D backend's port, `PORT` (default **3000**), over HTTPS when
`SSL_CERT_PATH`/`SSL_KEY_PATH` are set, plain HTTP otherwise.

## Environment

`.env` is a symlink to the workspace's shared `../files/.env` (the same file
ships-go reads); `ssl` links to `../files/ssl`. Variables used:

| Variable | Notes |
| --- | --- |
| `MONGODB_URI` | Required. Shared with ships-go. |
| `MONGODB_DATABASE` | Default `jaes`, the 2D game's database. |
| `SSL_CERT_PATH`, `SSL_KEY_PATH` | Shared with ships-go. |
| `PORT` | Shared with ships-go, default `3000`. The 3D site uses the 2D site's ports, so the two games can't run at the same time. |
| `ALLOWED_ORIGINS` | ships-go's CORS list (`\|`-separated). Covers the 3D site, which runs on the same ports. |
| `SHIPS3D_PORT`, `SHIPS3D_ALLOWED_ORIGINS` | Optional, 3D game only: override the port / add origins (e.g. a production domain). |

## API

| Method | Path | |
| --- | --- | --- |
| POST | `/register` | `{username, email, password, cpassword}` |
| POST | `/login` | `{email, password, rememberMe}`, sets the `token` cookie |
| POST | `/logout` | |
| GET | `/userInfo` | `{user}` (`null` when logged out) |
| POST | `/changePassword` | `{currentPassword, newPassword}`; logs out other sessions |
| GET/POST | `/projects` | list / create own projects |
| GET/PUT/DELETE | `/projects/:id` | own projects only; others' are a 404 |
| GET | `/game/ships` | `{defaults, own}` ships a player can fly |
| GET | `/ws` | the game websocket (protocol in `game/protocol.go`) |
| GET | `/status` | health + player count |

## Layout

- `main.go` — config, MongoDB, HTTP server, graceful shutdown.
- `config/` — env vars.
- `models/` — `User`, `Session` (bson-compatible with ships-go), `Project3d`
  with validation and the bounding radius used to normalise ship size.
- `dataAccess/` — one shared MongoDB client; every project query is scoped
  by owner.
- `controllers/` — HTTP routes, session cookie handling, origin checks.
- `game/` — the websocket hub: protocol, settings, built-in ships, hit
  validation. Tested end to end with real websocket clients (`hub_test.go`).
