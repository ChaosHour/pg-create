-- 1. Create the database (run as a superuser or role with CREATEDB)
CREATE DATABASE demo_db;

-- 2. Connect to the new database
-- In psql: \c demo_db
-- Or set it in your client/connection string

-- 3. Create a schema inside demo_db
CREATE SCHEMA IF NOT EXISTS app;

-- 4. Create tables in the app schema
CREATE TABLE app.users (
    id         SERIAL PRIMARY KEY,
    username   VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE app.posts (
    id         SERIAL PRIMARY KEY,
    user_id    INT NOT NULL REFERENCES app.users(id) ON DELETE CASCADE,
    title      VARCHAR(200) NOT NULL,
    body       TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. (Optional) Insert some sample data
INSERT INTO app.users (username) VALUES
    ('alice'),
    ('bob');

INSERT INTO app.posts (user_id, title, body) VALUES
    (1, 'Hello World', 'My first post'),
    (1, 'Another Post', 'More content'),
    (2, 'Hi there', 'Bob here');

-- 6. (Optional) Query to verify
SELECT u.username, p.title, p.created_at
FROM app.users u
JOIN app.posts p ON p.user_id = u.id
ORDER BY p.created_at;

