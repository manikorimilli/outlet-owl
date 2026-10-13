# Deploy the OutletOwl demo on Azure

How to put the public demo online, as decided in
[ADR-0010](../adr/0010-deploy-the-demo-on-azure-container-apps.md): one
container on Azure Container Apps serves the web app and the API from one
HTTPS address, with Azure PostgreSQL behind it. The engineer runs every step;
nothing here deploys by itself. Portal steps first, with a command where one
is simpler.

```
Browser ──HTTPS──> Azure Container App (one container: web app + Go API)
                        └──> Azure Database for PostgreSQL
                        └──> OpenRouter (free model)
```

## 0. Before you start

- An Azure subscription and the resource group `outletowl-rg` in Central
  India, with the providers `Microsoft.App`, `Microsoft.OperationalInsights`
  and `Microsoft.DBforPostgreSQL` registered.
- A budget alert: Cost Management, Budgets, + Add, USD 20, your email.
- The deployment PR merged to main. The `image` workflow (GitHub, Actions)
  then publishes `ghcr.io/manikorimilli/outlet-owl:latest`; wait for it to go
  green.
- The local demo database already tagged (`make db` running; the overview
  says "All reviews tagged").

## 1. Make the image public (GitHub)

Container Apps pulls the image without a password only if it is public. The
image holds the compiled app and no secret.

1. GitHub, your profile, **Packages**, `outlet-owl`.
2. **Package settings**, **Change visibility**, **Public**.

## 2. Create the database (Azure portal)

1. Search **Azure Database for PostgreSQL flexible servers**, **+ Create**.
2. Basics:
   - Resource group: `outletowl-rg`
   - Server name: `outletowl-db` (must be unique; add letters if taken)
   - Region: **Central India** (if it says no capacity, use **South India**)
   - PostgreSQL version: **16**
   - Workload type: **Development**
   - Compute + storage: **Configure server**, **Burstable**, **Standard_B1ms**,
     storage **32 GiB**, autogrow off, high availability off. On a free
     account this size is free for 12 months (750 hours a month).
   - Authentication: **PostgreSQL authentication only**; admin `outletowl`
     and a strong password. Keep it in your password manager.
3. Networking: **Public access**; tick **Allow public access from any Azure
   service within Azure to this server** (the Container App needs it); click
   **+ Add current client IP address** (your laptop, for loading the data).
4. **Review + create**, **Create**. About 5 to 10 minutes.
5. Open the server, **Databases**, **+ Add**, name `outlet_owl`, **Save**.

The connection string, used below as `AZURE_DATABASE_URL` (fill in your
server name and password; a password with `@`, `:` or `/` must be URL-encoded):

```
postgres://outletowl:<password>@outletowl-db.postgres.database.azure.com:5432/outlet_owl?sslmode=require
```

## 3. Load the demo data (your laptop)

Copy the local database, which already has the 500 tagged reviews, the
outlets and the budget log. Run from the repository root.

Dump the local database:

```bash
docker compose exec -T postgres sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --no-owner --no-acl -Fc' > outletowl.dump
```

Restore it into Azure with a PostgreSQL 16 client in a throwaway container
(paste your connection string):

```bash
docker run --rm -i postgres:16 pg_restore --no-owner --no-acl -d 'postgres://outletowl:<password>@outletowl-db.postgres.database.azure.com:5432/outlet_owl?sslmode=require' < outletowl.dump
```

Then delete `outletowl.dump`; it holds the demo data. To start empty instead:
`make migrate DATABASE_URL='<AZURE_DATABASE_URL>'`, then
`make seed DATABASE_URL='<AZURE_DATABASE_URL>' SEED_ARGS="-tag=false"`; the
server tags the reviews after it starts.

## 4. Create the Container App (Azure portal)

1. Search **Container Apps**, **+ Create**, **Container App**.
2. Basics:
   - Resource group `outletowl-rg`, name `outletowl`, region **Central India**
   - Container Apps environment: **Create new**, name `outletowl-env`,
     **Consumption only**, zone redundancy off.
3. Container (untick **Use quickstart image**):
   - Image source: **Docker Hub or other registries**, image type **Public**
   - Registry login server: `ghcr.io`
   - Image and tag: `manikorimilli/outlet-owl:latest`
   - CPU and memory: **0.25 CPU, 0.5 Gi**
   - Environment variables: leave for step 4b.
4. Ingress: **Enabled**, **Accepting traffic from anywhere**, transport
   **HTTP**, target port **8080**. Leave insecure connections off (HTTPS only).
5. **Review + create**, **Create**. The first revision fails to start until
   the settings below exist; that is expected.

### 4a. Secrets

App, **Settings, Secrets**, **+ Add**, each as a Container Apps secret:

| Name | Value |
| --- | --- |
| `database-url` | the `AZURE_DATABASE_URL` from step 2 |
| `jwt-secret` | output of `openssl rand -hex 32` (new, not your local one) |
| `openrouter-api-key` | your OpenRouter key |
| `users-json` | the whole content of your local `users.json` (demo accounts, password `outletowl-demo`) |

### 4b. Settings, volume and scale

App, **Application, Containers**, **Edit and deploy**, select the container,
**Edit**.

Environment variables (Source **Reference a secret** for the first three):

| Name | Value |
| --- | --- |
| `DATABASE_URL` | secret `database-url` |
| `JWT_SECRET` | secret `jwt-secret` |
| `OPENROUTER_API_KEY` | secret `openrouter-api-key` |
| `BRAND_NAME` | `Neem Tree Kitchens` |
| `BRAND_TIMEZONE` | `Asia/Kolkata` |
| `MODEL_GATEWAY_MODE` | `live` |
| `MODEL_ID` | `nvidia/nemotron-3-ultra-550b-a55b:free` |
| `USERS_FILE` | `/config/users-json` |
| `COOKIE_SECURE` | `true` |

`HOST`, `PORT` and `WEB_DIST` are set in the image; leave them out.

Volume mounts tab: **+ Add** a volume, type **Secret**, name `config`, secret
**users-json**; mount it at `/config`.

Scale tab: min replicas **1**, max replicas **1**. One replica on purpose: the
tagging worker and the sign-in limit live in the process (ADR-0010).

**Create** the revision. When it runs, the app's **Overview** shows the
**Application Url**, like
`https://outletowl.<words>.centralindia.azurecontainerapps.io`. That is the
demo link. Logs: **Monitoring, Log stream**.

## 5. Smoke test

On the Application Url:

1. `<Application Url>/api/v1/health` answers `{"status":"ok"}`.
2. The sign-in form is prefilled and says Demo; **Sign in** opens the Overview
   with the tiles and charts.
3. Reviews, Themes and Outlets load.
4. Import `testdata/sample-import.csv`: 10 imported, 2 rejected.
5. Sign out, sign in as `manager2@example.in`, open a review: the AI drafting
   animation, then a draft (or "Drafting is unavailable" when OpenRouter's
   free limit is used up).
6. Six wrong passwords for one email: the sixth says to wait 15 minutes.

The digest has no mail server on Azure and answers that the mail service did
not respond; that is expected until an SMTP service is configured.

## Updating

Every merge to main publishes a new image tagged with the commit (`image`
workflow). Container Apps does not pull it on its own: app, **Revisions and
replicas**, **Create new revision**, set the image tag to the new commit sha
(or keep `latest`), **Create**.

## Costs and teardown

- Container App: one replica of 0.25 CPU always on, a few dollars a month
  beyond the free grant, paid from the credits. Database: free 12 months on
  a free account at B1ms, else about USD 13 a month.
- Watch **Cost Management, Cost analysis** weekly.
- To remove everything: delete the resource group `outletowl-rg` (the app,
  the environment and the database).

## When something goes wrong

| What you see | Cause | Fix |
| --- | --- | --- |
| Revision fails, log says `config: ... is required` | a setting or secret is missing | add it in step 4b, create a new revision |
| Log says the users file cannot be read | the volume is not mounted or `USERS_FILE` differs | mount `config` at `/config`, `USERS_FILE=/config/users-json` |
| Revision fails pulling the image | the package is private | step 1 |
| Health says the database is unavailable | firewall or connection string | step 2: allow Azure services; check `sslmode=require` and the password encoding |
| Sign-in loops back to the form | `COOKIE_SECURE=true` on plain HTTP | open the `https://` Application Url only |
| Overview says reviews are not tagged | free model limit (429) | wait; the worker tags on the next import or revision restart |
