# SimBank — Sistema de Banca en Línea

Plataforma bancaria demo con balance en TigerBeetle (contabilidad de doble entrada),
autenticación JWT + sesiones revocables, y un asistente IA que puede ejecutar
operaciones bancarias mediante MCP (Model Context Protocol).

## Stack Tecnológico

| Capa        | Tecnología                                     |
|-------------|-----------------------------------------------|
| Backend     | Go 1.25, Chi router, pgx v5, godotenv         |
| Frontend    | React 19, Vite, Tailwind CSS 4, React Router 6, Axios |
| Base de datos | PostgreSQL 16 (usuarios y sesiones)         |
| Motor contable | TigerBeetle 0.17.7 (cuentas y transfers)   |
| Chat IA     | OpenRouter (API compatible OpenAI), MCP SDK   |
| Contenedores | Docker Compose, multi-stage builds           |

## Cómo correr el proyecto

```bash
# Clonar el repositorio
git clone <url-del-repo>
cd SimBank

# (Opcional) Configurar variables de entorno
cp .env.example .env
# Editar .env si es necesario, especialmente OPENROUTER_API_KEY

# Levantar todo el sistema con un solo comando
docker compose up -d

# Verificar que los servicios están corriendo
docker compose ps
```

El comando `docker compose up -d` inicia automáticamente:

1. **PostgreSQL** — esquema y tablas vía `db/init/`
2. **TigerBeetle** — formateo e inicialización del archivo de datos
3. **Backend** — compila el binario Go y se conecta a PostgreSQL y TigerBeetle
4. **Frontend** — compila la SPA de React y la sirve con nginx

Accede a la aplicación en **http://localhost:5173**.

### Credenciales de prueba

El seed carga automáticamente estos usuarios (archivo `data/datos-prueba-HNL.json`):

| Email               | Contraseña   | Cuenta    | Saldo inicial |
|---------------------|-------------|-----------|---------------|
| alice@example.com   | password123 | 001-0001  | $100,000.00   |
| bob@example.com     | password123 | 001-0002  | $50,000.00    |
| carol@example.com   | password123 | 001-0003  | $25,000.00    |

### Variables de entorno necesarias

Todas las variables tienen valores por defecto en `docker-compose.yml`.
Para el chat IA, crear un archivo `.env` en la raíz con:

```
OPENROUTER_API_KEY=sk-or-v1-tu-api-key-aqui
```

Ver `.env.example` para la lista completa.

## Arquitectura

### TigerBeetle — Contabilidad de doble entrada

TigerBeetle es un banco contable de doble entrada. Cada cuenta tiene débitos y
créditos, y el saldo se calcula como `credits_posted - debits_posted`. El
sistema usa una **cuenta de control del banco** (ID = 1) como contraparte
interna:

- **Depósito**: el banco (cuenta 1) acredita al usuario
- **Retiro**: el usuario debita al banco (cuenta 1)
- **Transferencia**: el usuario debita al destino, el destino acredita del usuario
  (transferencia directa entre cuentas de usuario, sin pasar por el banco)

TigerBeetle garantiza atomicidad y consistencia con su modelo de
`CreateTransfers`, rechazando transfers que excedan el saldo disponible
(`TransferExceedsCredits`).

El Go SDK de TigerBeetle v0.17.7 usa CGo con io_uring, lo que requiere
`privileged: true` en Docker Desktop para Windows.

### MCP (Model Context Protocol)

El chat IA se conecta a las operaciones bancarias mediante MCP. Cuando el
usuario pide al asistente que realice una operación, el flujo es:

1. El frontend envía el mensaje a `POST /api/chat`
2. El backend construye un sistema prompt con las herramientas disponibles
   (get_balance, deposit, withdraw, get_transaction_history) y las pasa a
   OpenRouter como function calling
3. OpenRouter decide qué herramienta(s) llamar según la solicitud
4. El backend ejecuta la herramienta vía MCP in-memory y devuelve el resultado
5. Para transferencias, **no se ejecutan automáticamente**: el backend devuelve
   `requires_confirmation: true` con los detalles de la acción pendiente
6. El frontend muestra al usuario los detalles y pide confirmación explícita
7. Al confirmar, se llama a `POST /api/chat/confirm` que ejecuta la transferencia

Cada sesión de chat crea un servidor MCP por usuario con un transporte
en memoria. Esto asegura que las herramientas solo operen sobre la cuenta
del usuario autenticado.

## Decisiones Técnicas y Ambigüedades Resueltas

### 1. Token en sessionStorage (no localStorage)

Los tokens JWT se guardan en **sessionStorage**, no en localStorage. La
diferencia es que sessionStorage se borra al cerrar la pestaña/ventana del
navegador, mientras que localStorage persiste indefinidamente. Esto reduce
la ventana de exposición si un atacante obtiene acceso al mismo navegador
(por ejemplo, en un equipo compartido).

**Trade-off**: sessionStorage no está disponible entre pestañas/ventanas
del mismo origen. Si el usuario abre una segunda pestaña, tendrá que
iniciar sesión de nuevo. Para esta aplicación demo es el equilibrio correcto
entre seguridad y usabilidad. En un banco real se usarían HTTP-only cookies
con Secure + SameSite=Strict para eliminar completamente el XSS.

### 2. TigerBeetle: hostname a IP

El cliente nativo de TigerBeetle v0.17.x rechaza hostnames y requiere
direcciones IP. El backend resuelve automáticamente `tigerbeetle:3000` a
su IP al iniciar la conexión mediante `net.LookupHost`.

### 3. io_uring y Docker Desktop

El Go SDK de TigerBeetle usa CGo con io_uring, que no está disponible por
defecto en Docker Desktop para Windows. Se requiere `privileged: true` en
el contenedor del backend.

### 4. MCP por sesión de chat

En lugar de un servidor MCP global compartido, se crea una instancia de
`ServerFactory` por solicitud de chat, con el `userID` inyectado en closure
de cada herramienta. Esto evita mezclar contextos de usuario y simplifica
la autorización.

### 5. Seed idempotente

El seed verifica si el email ya existe antes de crear cuentas. Si se reinicia
el contenedor sin borrar el volumen de PostgreSQL, el seed no duplica datos.
Los montos iniciales se depositan mediante transfers TB, que también son
idempotentes (cada transfer tiene un ID único).

### 6. tipo NUMERIC(39,0) → TEXT para tigerbeetle_account_id

Originalmente la columna era `NUMERIC(39,0)` para almacenar el Uint128 de
TigerBeetle. Sin embargo, `tb.ID().String()` devuelve una cadena hexadecimal
(ej: `19eef43caa6c3cbe46530888fc014b9`), no un número decimal. Se cambió a
`TEXT` para evitar errores de conversión.

## Endpoints de la API

### Autenticación (públicos)

| Método | Ruta                | Descripción                       |
|--------|---------------------|-----------------------------------|
| POST   | `/api/auth/register` | Crear cuenta (email, password, full_name) |
| POST   | `/api/auth/login`    | Iniciar sesión, devuelve JWT      |

### Autenticación (requieren JWT en header `Authorization: Bearer <token>`)

| Método | Ruta                         | Descripción                        |
|--------|------------------------------|------------------------------------|
| POST   | `/api/auth/logout`           | Revocar sesión actual              |
| GET    | `/api/account/me`            | Perfil del usuario                 |
| GET    | `/api/account`               | Información de la cuenta bancaria  |
| GET    | `/api/account/balance`       | Saldo actual                       |
| POST   | `/api/transactions/deposit`  | Depositar fondos                   |
| POST   | `/api/transactions/withdraw` | Retirar fondos                     |
| POST   | `/api/transactions/transfer` | Transferir a otra cuenta           |
| GET    | `/api/transactions/history`  | Historial de transacciones         |
| POST   | `/api/chat`                  | Enviar mensaje al asistente IA     |
| POST   | `/api/chat/confirm`          | Confirmar transferencia pendiente  |
| GET    | `/health`                    | Health check                       |

## Estructura del Proyecto

```
SimBank/
├── backend/                  # Servidor Go
│   ├── cmd/server/main.go    # Punto de entrada
│   ├── internal/
│   │   ├── ai/               # Chat IA + OpenRouter
│   │   ├── auth/             # bcrypt, JWT, SHA-256
│   │   ├── config/           # Variables de entorno
│   │   ├── db/               # PostgreSQL (usuarios, sesiones)
│   │   ├── handlers/         # HTTP handlers (auth, bank, chat)
│   │   ├── ledger/           # Cliente TigerBeetle
│   │   ├── mcp/              # Servidor MCP in-memory
│   │   ├── middleware/        # Auth middleware (JWT)
│   │   ├── models/           # Tipos compartidos
│   │   └── seed/             # Carga de datos de prueba
│   ├── go.mod / go.sum
│   └── Dockerfile
├── frontend/                 # SPA React + Vite
│   ├── src/
│   │   ├── api/client.js     # Axios con interceptor JWT
│   │   ├── context/          # AuthContext
│   │   ├── pages/            # Login, Register, Dashboard, etc.
│   │   └── components/       # Navbar, ChatWidget, etc.
│   ├── package.json
│   └── Dockerfile
├── db/init/                  # Schema SQL
├── data/                     # Seed data + archivos TB
├── .env.example
├── docker-compose.yml
└── README.md
```
