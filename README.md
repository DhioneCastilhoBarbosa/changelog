<div align="center">

# Firmware Changelog API

**API em Go para gestão de releases de firmware, changelogs e homologações**

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![Gin](https://img.shields.io/badge/Gin-Web%20Framework-08A4E4?style=for-the-badge)](https://gin-gonic.com/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-GORM-336791?style=for-the-badge&logo=postgresql&logoColor=white)](https://gorm.io/)
[![JWT](https://img.shields.io/badge/Auth-JWT-000000?style=for-the-badge&logo=jsonwebtokens&logoColor=white)](https://jwt.io/)

<br />

Catálogo versionado de firmwares · upload via WebDAV · fluxo de aprovações · RBAC com JWT

[Começar](#-começando) · [API](#-api) · [Variáveis de ambiente](#-variáveis-de-ambiente) · [Deploy](#-deploy)

</div>

---

## Visão geral

Backend REST que centraliza o ciclo de vida de firmwares: versões, módulos, notas de changelog, links de download e documentos de homologação.

Feito para alimentar frontends de documentação e changelog (ex.: `changelog` e `doc`), com leitura pública e escrita protegida por papéis.

```text
Cliente (Web)
     │
     ▼
┌─────────────────┐     JWT + RBAC      ┌──────────────────┐
│  Firmware       │ ──────────────────► │  PostgreSQL      │
│  Changelog API  │                     │  (releases,      │
│  (Gin + GORM)   │                     │   users, etc.)   │
└────────┬────────┘                     └──────────────────┘
         │
         │  WebDAV PUT / DELETE
         ▼
┌─────────────────┐
│  File Server    │
│  (firmware /    │
│   approvals)    │
└─────────────────┘
```

---

## Funcionalidades

| Área | O que faz |
|------|-----------|
| **Releases** | CRUD de versões com produto, categoria, status, OTA e notas importantes |
| **Changelog** | Entradas classificadas (`Novo`, `Otimização`, `Correção`, `Segurança`) |
| **Módulos** | Versões por módulo/PCB vinculadas ao release |
| **Arquivos** | Upload multipart (até 512 MiB) ou links JSON; publicação via WebDAV |
| **Homologações** | Registro de aprovações com estabelecimento, produto e arquivo anexo |
| **Auth** | Login JWT, seed de admin e usuários com papéis `admin` / `editor` / `viewer` |
| **Consulta pública** | Listagem e detalhe de releases sem autenticação |

### Status de firmware

| Status | Uso |
|--------|-----|
| `revisao` | Em validação |
| `producao` | Liberado (padrão) |
| `descontinuado` | Fora de linha |

---

## Stack

- **Go 1.23** — runtime
- **Gin** — HTTP, CORS e multipart
- **GORM + PostgreSQL** — persistência e AutoMigrate
- **JWT (HS256)** — autenticação com leeway de 5s
- **bcrypt** — hash de senhas
- **WebDAV** — upload/remoção de firmwares e documentos
- **godotenv** — `.env` local
- **Nixpacks** — build/start para deploy

---

## Estrutura do projeto

```text
.
├── cmd/server/          # entrypoint (env, migrate, seed, listen)
├── internal/
│   ├── config/          # PORT, DATABASE_URL, JWT_SECRET
│   ├── db/              # conexão PostgreSQL
│   ├── models/          # User, Release, Module, Changelog, Approval…
│   ├── repository/      # acesso a dados
│   ├── service/         # regras de negócio
│   └── http/
│       ├── handlers/    # auth, users, releases, approvals
│       ├── midleware/   # JWT + RequireRole
│       └── router/      # rotas e CORS
├── nixpacks.toml
├── go.mod
└── go.sum
```

Arquitetura em camadas: **handler → service → repository → model**.

---

## Começando

### Pré-requisitos

- Go **1.23+**
- PostgreSQL
- (Opcional) servidor de arquivos com WebDAV para uploads

### 1. Clone e dependências

```bash
git clone https://github.com/DhioneCastilhoBarbosa/changelog.git
cd changelog
go mod download
```

### 2. Variáveis de ambiente

Crie um `.env` na raiz:

```env
PORT=8080
DATABASE_URL=postgres://user:pass@localhost:5432/firmware_changelog?sslmode=disable
JWT_SECRET=troque-por-um-segredo-forte

# Seed opcional do admin (só cria se o e-mail ainda não existir)
ADMIN_EMAIL=admin@empresa.com
ADMIN_PASSWORD=senha-segura
ADMIN_NAME=Admin

# File server — firmwares
FILE_PUBLIC_BASE=https://files.seudominio.com/firmware
FILE_SERVER_BASE=https://files.seudominio.com/firmware
FILE_SERVER_USER=uploader
FILE_SERVER_PASS=

# File server — homologações
FILE_PUBLIC_BASE_APPROVALS=https://files.seudominio.com/approvals
FILE_SERVER_BASE_APPROVALS=https://files.seudominio.com/approvals

# Timeout HTTP para upload (ex.: 120 ou 120s)
HTTP_TIMEOUT=120s
```

### 3. Subir a API

```bash
go run ./cmd/server
```

A API sobe em `http://localhost:8080`. Na primeira execução o GORM aplica o AutoMigrate nas tabelas.

---

## Variáveis de ambiente

| Variável | Obrigatória | Descrição |
|----------|:-----------:|-----------|
| `PORT` | ✅ | Porta HTTP (padrão no config: `8080`) |
| `DATABASE_URL` | ✅ | DSN PostgreSQL |
| `JWT_SECRET` | ✅ | Segredo HS256 do token |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` / `ADMIN_NAME` | — | Seed do usuário admin |
| `FILE_PUBLIC_BASE` | — | Base pública dos links de firmware |
| `FILE_SERVER_BASE` | — | Base WebDAV para PUT/DELETE de firmware |
| `FILE_PUBLIC_BASE_APPROVALS` | — | Base pública dos anexos de homologação |
| `FILE_SERVER_BASE_APPROVALS` | — | Base WebDAV das homologações |
| `FILE_SERVER_USER` / `FILE_SERVER_PASS` | — | Basic Auth do file server |
| `HTTP_TIMEOUT` | — | Timeout de upload (`120` ou `120s`) |

---

## API

Base: `/api`

### Autenticação e usuários

| Método | Rota | Auth | Descrição |
|--------|------|:----:|-----------|
| `POST` | `/api/auth/login` | — | Login → `{ token, user }` |
| `POST` | `/api/users` | — | Cria usuário com papel `viewer` |

```bash
curl -s -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@empresa.com","password":"senha-segura"}'
```

Use o token nas rotas protegidas:

```http
Authorization: Bearer <token>
```

### Releases (público)

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/api/releases` | Lista com filtros |
| `GET` | `/api/releases/:id` | Detalhe |

**Query params em `GET /api/releases`:**

| Param | Exemplo | Efeito |
|-------|---------|--------|
| `q` | `camera` | Busca textual |
| `version` | `1.2.0` | Filtra versão |
| `date_from` | `2024-01-01` | Data inicial (`YYYY-MM-DD`) |
| `date_to` | `2024-12-31` | Data final |

### Releases (protegido)

Papéis: **admin** ou **editor** (delete de release/arquivo: só **admin**).

| Método | Rota | Descrição |
|--------|------|-----------|
| `POST` | `/api/releases` | Cria (JSON ou multipart) |
| `PUT` | `/api/releases/:id` | Atualiza (JSON ou multipart) |
| `DELETE` | `/api/releases/:id` | Remove release e tenta apagar arquivos no DAV |
| `DELETE` | `/api/releases/file` | Remove arquivo avulso (`url` ou `path`) |

**JSON (exemplo resumido):**

```json
{
  "version": "1.4.0",
  "previousVersion": "1.3.2",
  "ota": true,
  "otaObs": "Atualização recomendada",
  "releaseDate": "2024-06-01T00:00:00Z",
  "importantNote": "Reinicie o equipamento após o flash",
  "status": "producao",
  "productCategory": "CFTV",
  "productName": "VIP 1230 B G3",
  "modules": [
    { "module": "PCB A7", "version": "1.3033.0", "updated": true }
  ],
  "entries": [
    {
      "itemOrder": 1,
      "classification": "Correção",
      "observation": "Corrige falha de sincronismo NTP"
    }
  ],
  "links": [
    {
      "module": "Principal",
      "description": "Firmware completo",
      "url": "https://files.seudominio.com/firmware/vip-1230.bin"
    }
  ]
}
```

**Multipart:** campo `data` (JSON) + um ou mais arquivos (`file` / arquivos). Upload enviado ao file server via WebDAV PUT; a API devolve a URL pública.

### Homologações (protegido)

Grupo: `/api/v1` (requer JWT; create/list/get/delete conforme rotas abaixo).

| Método | Rota | Descrição |
|--------|------|-----------|
| `POST` | `/api/v1` | Cria homologação (JSON ou multipart) |
| `GET` | `/api/v1/approvals` | Lista |
| `GET` | `/api/v1/approvals/:id` | Detalhe |
| `DELETE` | `/api/v1/approvals/:id` | Remove |

Campos principais: `establishment`, `date`, `productName`, `category`, `description`, `fileUrl` (ou arquivo no multipart).

---

## Papéis (RBAC)

| Papel | Capacidade |
|-------|------------|
| `viewer` | Papel padrão no cadastro público; leitura via rotas abertas |
| `editor` | Cria/atualiza releases e homologações |
| `admin` | Tudo do editor + delete de release e arquivo no file server |

O middleware JWT valida assinatura HS256, extrai `uid`/`sub` e `role`, e o `RequireRole` restringe mutações.

---

## CORS

Origens liberadas no router:

- `http://localhost:5173`
- `https://changelog.intelbras-cve-pro.com.br`
- `https://doc.intelbras-cve-pro.com.br`

Métodos: `GET`, `POST`, `PUT`, `DELETE`, `OPTIONS` · credentials habilitadas.

---

## Deploy

O repositório inclui `nixpacks.toml`:

```toml
[phases.setup]
nixpkgs = ["go"]

[phases.install]
cmds = ["go mod download"]

[phases.build]
cmds = ["go build -o app ./cmd/server"]

[start]
cmd = "./app"
```

**Build local de produção:**

```bash
go build -o app ./cmd/server
./app
```

Configure `DATABASE_URL`, `JWT_SECRET` e `PORT` no ambiente do host/plataforma.

---

## Modelo de domínio (resumo)

```text
User ──┐
       ├── Release ── Modules
       │           ├── ChangelogEntries
       │           └── FirmwareLinks
       └── Approval (+ FileURL)
```

- **Release** — versão, produto, status, OTA, autor
- **ReleaseModule** — módulo/PCB + versão
- **ChangelogEntry** — classificação + observação ordenada
- **FirmwareLink** — módulo, descrição e URL do binário
- **Approval** — homologação com anexo no file server

---

## Desenvolvimento

```bash
# testes
go test ./...

# build
go build -o app ./cmd/server
```

Boas práticas rápidas:

1. Nunca commitar `.env` (já está no `.gitignore`)
2. Usar `JWT_SECRET` forte e distinto por ambiente
3. Restringir WebDAV com usuário/senha e bases separadas para firmware e approvals
4. Validar `status` e classificações de changelog no cliente e na API

---

## Licença

Uso interno / projeto privado — ajuste conforme a política do repositório.

---

<div align="center">

Feito com Go · Gin · GORM · PostgreSQL

</div>
