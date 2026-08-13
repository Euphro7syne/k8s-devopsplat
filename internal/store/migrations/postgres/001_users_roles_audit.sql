CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL DEFAULT '',
  mfa_secret TEXT NOT NULL DEFAULT '',
  provider TEXT NOT NULL DEFAULT 'local',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (status IN ('active', 'disabled'))
);

CREATE TABLE IF NOT EXISTS roles (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS user_roles (
  user_id BIGINT NOT NULL,
  role_id BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, role_id),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS clusters (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  kubeconfig_ref TEXT NOT NULL DEFAULT '',
  in_cluster BOOLEAN NOT NULL DEFAULT TRUE,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (status IN ('active', 'disabled'))
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NULL,
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_name TEXT NOT NULL DEFAULT '',
  namespace TEXT NOT NULL DEFAULT '',
  request_body TEXT NOT NULL DEFAULT '',
  ip TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

INSERT INTO roles(name) VALUES
  ('viewer'),
  ('operator'),
  ('configadmin'),
  ('auditor'),
  ('admin')
ON CONFLICT (name) DO NOTHING;

INSERT INTO clusters(name, kubeconfig_ref, in_cluster, status) VALUES
  ('in-cluster', '', TRUE, 'active')
ON CONFLICT (name) DO NOTHING;
