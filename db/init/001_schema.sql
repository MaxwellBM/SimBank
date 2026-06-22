-- SimBank - Schema de PostgreSQL
-- Tablas: users (cuentas de usuario), sessions (sesiones JWT revocables)

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Funcion para actualizar updated_at automaticamente
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- Tabla: users
-- ============================================================
-- Almacena SOLO datos de usuario y autenticacion.
-- Las cuentas financieras y transacciones viven en TigerBeetle.
-- tigerbeetle_account_id es TEXT porque almacenamos la
-- representacion hex del Uint128 de TigerBeetle (ej: "19ee...").
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    tigerbeetle_account_id TEXT NOT NULL UNIQUE,
    account_number VARCHAR(30) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_account_number ON users (account_number);

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- Tabla: sessions
-- ============================================================
-- Permite revocar sesiones activas (logout) sin depender solo
-- de la expiracion del JWT. Almacenamos el HASH del token,
-- nunca el token en texto plano.
-- ============================================================
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_token_hash ON sessions (token_hash);
