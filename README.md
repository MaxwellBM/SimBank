# SimBank

Sistema de banca en línea — prueba técnica para desarrollador Jr con enfoque en IA.

## Stack

- **Backend**: Go (chi router, pgx, tigerbeetle-go)
- **Frontend**: React + Vite
- **Base de datos financiera**: TigerBeetle (doble entrada)
- **Base de datos usuarios**: PostgreSQL
- **Autenticación**: JWT
- **Chat IA**: MCP + OpenRouter

## Requisitos

- Docker + Docker Compose (con Compose V2)

## Arranque rápido

1. Clona el repositorio.
2. Copia el archivo de datos de prueba a `data/datos-prueba-HNL.json`.
3. (Opcional) Crea un archivo `.env` basado en `.env.example` y configura `OPENROUTER_API_KEY`.
4. Ejecuta:

```bash
docker-compose up --build
```

5. Accede a:
   - Frontend: http://localhost:5173
   - API: http://localhost:8080
   - Health check: http://localhost:8080/health

## Variables de entorno

Ver `.env.example` para la lista completa.
