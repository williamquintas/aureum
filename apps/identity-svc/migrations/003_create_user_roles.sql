-- Migration: 003_create_user_roles.sql
-- Create user_roles table for RBAC

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('admin', 'user', 'readonly')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role)
);

CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
