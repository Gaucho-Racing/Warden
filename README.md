# Warden

[![build](https://github.com/Gaucho-Racing/Warden/actions/workflows/build.yml/badge.svg)](https://github.com/Gaucho-Racing/Warden/actions/workflows/build.yml)
[![Release](https://img.shields.io/github/v/release/Gaucho-Racing/Warden?style=flat-square)](https://github.com/Gaucho-Racing/Warden/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Warden is Gaucho Racing's Sentinel integration for Minecraft servers.
It links a player's Minecraft account to their Sentinel entity, then turns their Sentinel group membership into in-game permissions, so who can play and what they can do follows the same access model as every other internal tool.

The flow runs in one direction: Sentinel groups are the source, Minecraft permissions are the sink.
Warden's service account can read group membership and never write it, and nothing in this repo teaches Sentinel about Minecraft.

Warden also runs the game server itself, relays chat between the game and Discord, backs the whole server up to Depot on a schedule, and reports player statistics and server health back to a web portal.

Portal: [warden.gauchoracing.com](https://warden.gauchoracing.com)
Game server: `mc.gauchoracing.com`

## Components

| Directory | Description |
| --- | --- |
| `warden/` | Go API. Accounts, link tokens, group bindings, permission resolution, statistics, backups, audit logs, and the Discord bridge. |
| `web/` | React/Vite portal for linking an account, browsing the player roster, watching backups, and managing bindings and the backup schedule. |
| `plugin/` | Paper plugin. Resolves permissions at login, applies them through LuckPerms, confines unlinked players, archives the server on command, and speaks the bridge protocol. |
| `minecraft-server/` | Dockerized Paper server with the Warden jar and the rest of the plugin set baked in. |

## Features

- Sentinel SSO, with `MinecraftAdmins` gating bindings, the backup schedule, and the audit log.
- Account linking through a short-lived, single-use token claimed in the browser.
- Group bindings mapping a Sentinel group onto a managed LuckPerms group, its permission nodes, and its weight.
- Permission resolution at prelogin plus a periodic full reconcile, both sending complete desired state rather than deltas.
- Fail-closed posture: if Warden is unreachable a player still joins, but with no managed groups and confined to spawn.
- Discord chat bridge over a plugin WebSocket, with game chat carrying each player's name and head.
- Player statistics and server health, with daily snapshots for seven-day deltas.
- Scheduled whole-server backups to Depot, uploaded straight from the game server through a presigned URL.
- Staff tooling: `/fly` and `/vanish`, both permission-gated.
- Audit logs for link tokens, account links, binding changes, and backups.
- Multi-architecture server, web, and Minecraft images published to GitHub Container Registry.

## Getting Started

### Prerequisites

- Docker with Docker Compose
- A Sentinel OAuth client configured with `http://localhost:10310/auth/login` as a redirect URI
- A Sentinel service account token with `groups:read`, and write access to the Depot `minecraft` bucket if you want backups
- Membership in `MinecraftAdmins` to manage bindings and the backup schedule

### Local Development

1. Clone the repository and enter the project directory.

   ```sh
   git clone https://github.com/Gaucho-Racing/Warden.git
   cd Warden
   ```

2. Create the local environment file.

   ```sh
   cp example.env .env
   ```

3. Set `SENTINEL_CLIENT_ID`, `SENTINEL_CLIENT_SECRET`, and `SENTINEL_SA_TOKEN` in `.env` using the local Sentinel credentials, then generate a plugin token.

   ```sh
   openssl rand -hex 32
   ```

4. Start Warden.

   ```sh
   docker compose up --build
   ```

5. Open [http://localhost:10310](http://localhost:10310).

The local stack runs the React frontend, Go API, Postgres, and the Kerbecs gateway.
Database migrations are applied automatically when the API starts.

The game server is not part of the compose stack.
To point a local Paper server at it, build the plugin and set `warden.base-url` to `http://localhost:10310` and `warden.token` to the same value as `PLUGIN_TOKEN` in `plugin/src/main/resources/config.yml`.

### Configuration

| Variable | Description |
| --- | --- |
| `SENTINEL_URL` | Base URL of the Sentinel deployment. |
| `SENTINEL_CLIENT_ID` | Sentinel OAuth client ID used by the API and web application. |
| `SENTINEL_CLIENT_SECRET` | Sentinel OAuth client secret used for authorization-code exchange. |
| `SENTINEL_REDIRECT_URI` | OAuth callback URL registered with Sentinel. |
| `SENTINEL_SA_TOKEN` | Service account token used to read entity group membership. |
| `PLUGIN_TOKEN` | Bearer the Minecraft plugin presents on `/plugin/*`. Generate with `openssl rand -hex 32`. |
| `PUBLIC_BASE_URL` | Public origin players reach the link portal on. Used to build the link URL sent in chat. |
| `LINK_TOKEN_TTL` | Lifetime of a link token. Defaults to `15m`. |
| `GROUP_SYNC_INTERVAL` | How often the full permission reconcile runs. Defaults to `60s`. |
| `DISCORD_TOKEN` | Bot token for the chat bridge. The bridge runs only when this and the channel ID are both set. |
| `DISCORD_CHANNEL_ID` | Channel the bridge relays to and from. |
| `BACKUP_TIMEZONE` | IANA zone backup cron expressions are evaluated in. Defaults to `America/Los_Angeles`. |
| `BACKUP_WARNING_LEAD` | How far ahead of a scheduled backup players are warned. Defaults to `5m`. |
| `BACKUP_MANUAL_DELAY` | Grace period between the warning and the archive on a manual backup. Defaults to `15s`. |
| `BACKUP_TIMEOUT` | When to give up on a backup the game server never reported the end of. Defaults to `60m`. |

The Depot origin and bucket are compiled in rather than configured, because the bucket's write grant names the Warden application specifically and pointing at a different one is a Depot-side change anyway.
Backups stay off entirely without `SENTINEL_SA_TOKEN`, which is the only credential the upload path needs.

The plugin token is deliberately scoped to Warden alone.
It lives in a config file on the game server, so it is a lower-trust credential than the Sentinel client secret, and no route it authenticates can read or write Sentinel state.

### Validation

```sh
cd web
npm ci
npm run lint
npm run build

cd ../warden
go build ./...
go vet ./...

cd ../plugin
./mvnw -B clean package
```

## Bindings

A binding maps one Sentinel group onto a managed LuckPerms group, a set of permission nodes, and a weight.
Only groups linked to the Warden application in Sentinel are bindable — linking a group to the app is the deliberate act that makes it available to Minecraft.

Warden owns the `warden-` prefixed LuckPerms groups outright: their nodes and weight are replaced from the binding on every reconcile.
Every other group is left alone, so hand-granted permissions belong on a group Warden does not manage.

`warden.play` is the permission that lets a player leave spawn and play in survival.
It is granted only through a binding, never by op, so access follows Sentinel group membership.
Players without it join normally but are confined to spawn in adventure mode.

## Backups

Warden archives the whole game server on a schedule and uploads it to the `minecraft` bucket in [Depot](https://github.com/Gaucho-Racing/Depot).

The world lives on a ReadWriteOnce volume mounted only into the game server pod, so Warden cannot read it.
The plugin does the archiving: Warden mints a presigned upload URL from Depot and hands it over the bridge socket, and the `.tar.gz` goes from the game server straight to object storage.
It is staged on the game server's own volume first, so a backup needs roughly half the uncompressed server size free to run, and only one runs at a time.

Every world is flushed and autosave paused for the length of the archive, so region files are never captured mid-write.
Plugin databases stay writable and land crash-consistent, which SQLite and H2 recover from but which is not the clean point-in-time snapshot the worlds get.

The schedule is a standard five-field cron expression evaluated in `BACKUP_TIMEZONE`, edited in Settings by `MinecraftAdmins`.
Everyone signed in can see it, the run in progress, and the history.
Players are warned before a backup and again when it starts and finishes, in Discord and in game chat.

## Release

Run the release script from a clean, up-to-date `main` branch:

```sh
./scripts/release.sh 1.9.0
```

The script bumps the version in `warden/config/config.go` and `plugin/pom.xml`, creates the release commit, and publishes a GitHub release.
The release workflows publish multi-architecture `warden-server`, `warden-web`, and `warden-minecraft` images to GitHub Container Registry, then open an infrastructure pull request with the new image tags.

## Contributing

If you have a suggestion that would make this better, please fork the repo and create a pull request. You can also open an issue with the tag `enhancement`.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b gh-username/my-amazing-feature`)
3. Commit your Changes (`git commit -m 'Add my amazing feature'`)
4. Push to the Branch (`git push origin gh-username/my-amazing-feature`)
5. Open a Pull Request

## License

Distributed under the MIT License. See `LICENSE.txt` for more information.
