-- =============================================================
-- E-Commerce Schema for PostgreSQL 15
-- Usage: psql -U postgres -f ecommerce.sql
-- psql -h chaos-sandbox-pg-qa-usa.dms.gannettdigital.com -U admin -d postgres -f ecommerce.sql
-- psql -h chaos-sandbox-pg-qa-usa.dms.gannettdigital.com -U admin -d postgres < ecommerce.sql
-- psql -h chaos-sandbox-pg-qa-usa.dms.gannettdigital.com -U readonly -d postgres -c "\du"
-- psql -h chaos-sandbox-pg-qa-usa.dms.gannettdigital.com -U readonly -d postgres -c "\dt store.*"
-- =============================================================

-- -------------------------------------------------------------
-- Database
-- -------------------------------------------------------------
CREATE DATABASE ecommerce
    WITH ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.utf8'
    LC_CTYPE   = 'en_US.utf8'
    TEMPLATE   = template0;

\c ecommerce

-- -------------------------------------------------------------
-- Schema
-- -------------------------------------------------------------
CREATE SCHEMA IF NOT EXISTS store;

SET search_path TO store, public;

-- -------------------------------------------------------------
-- Tables
-- -------------------------------------------------------------

CREATE TABLE store.customers (
    id              SERIAL          PRIMARY KEY,
    email           VARCHAR(255)    NOT NULL UNIQUE,
    first_name      VARCHAR(100)    NOT NULL,
    last_name       VARCHAR(100)    NOT NULL,
    phone           VARCHAR(20),
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE TABLE store.addresses (
    id              SERIAL          PRIMARY KEY,
    customer_id     INT             NOT NULL REFERENCES store.customers(id) ON DELETE CASCADE,
    label           VARCHAR(50)     NOT NULL DEFAULT 'home',   -- home, work, etc.
    street          VARCHAR(255)    NOT NULL,
    city            VARCHAR(100)    NOT NULL,
    state           CHAR(2)         NOT NULL,
    zip             VARCHAR(10)     NOT NULL,
    country         CHAR(2)         NOT NULL DEFAULT 'US',
    is_default      BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE TABLE store.categories (
    id              SERIAL          PRIMARY KEY,
    name            VARCHAR(100)    NOT NULL UNIQUE,
    slug            VARCHAR(100)    NOT NULL UNIQUE,
    parent_id       INT             REFERENCES store.categories(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE TABLE store.products (
    id              SERIAL          PRIMARY KEY,
    category_id     INT             REFERENCES store.categories(id) ON DELETE SET NULL,
    sku             VARCHAR(100)    NOT NULL UNIQUE,
    name            VARCHAR(255)    NOT NULL,
    description     TEXT,
    price           NUMERIC(10, 2)  NOT NULL CHECK (price >= 0),
    stock_qty       INT             NOT NULL DEFAULT 0 CHECK (stock_qty >= 0),
    is_active       BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE TABLE store.orders (
    id              SERIAL          PRIMARY KEY,
    customer_id     INT             NOT NULL REFERENCES store.customers(id),
    shipping_addr_id INT            REFERENCES store.addresses(id),
    status          VARCHAR(50)     NOT NULL DEFAULT 'pending'
                                    CHECK (status IN ('pending','confirmed','shipped','delivered','cancelled','refunded')),
    subtotal        NUMERIC(10, 2)  NOT NULL DEFAULT 0,
    shipping_cost   NUMERIC(10, 2)  NOT NULL DEFAULT 0,
    tax             NUMERIC(10, 2)  NOT NULL DEFAULT 0,
    total           NUMERIC(10, 2)  GENERATED ALWAYS AS (subtotal + shipping_cost + tax) STORED,
    notes           TEXT,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE TABLE store.order_items (
    id              SERIAL          PRIMARY KEY,
    order_id        INT             NOT NULL REFERENCES store.orders(id) ON DELETE CASCADE,
    product_id      INT             NOT NULL REFERENCES store.products(id),
    quantity        INT             NOT NULL CHECK (quantity > 0),
    unit_price      NUMERIC(10, 2)  NOT NULL CHECK (unit_price >= 0),
    line_total      NUMERIC(10, 2)  GENERATED ALWAYS AS (quantity * unit_price) STORED
);

CREATE TABLE store.payments (
    id              SERIAL          PRIMARY KEY,
    order_id        INT             NOT NULL REFERENCES store.orders(id),
    method          VARCHAR(50)     NOT NULL CHECK (method IN ('credit_card','debit_card','paypal','bank_transfer')),
    status          VARCHAR(50)     NOT NULL DEFAULT 'pending'
                                    CHECK (status IN ('pending','completed','failed','refunded')),
    amount          NUMERIC(10, 2)  NOT NULL CHECK (amount > 0),
    transaction_ref VARCHAR(255),
    paid_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- -------------------------------------------------------------
-- Indexes
-- -------------------------------------------------------------

-- customers
CREATE INDEX idx_customers_email        ON store.customers (email);
CREATE INDEX idx_customers_last_name    ON store.customers (last_name);

-- addresses
CREATE INDEX idx_addresses_customer_id  ON store.addresses (customer_id);

-- products
CREATE INDEX idx_products_sku           ON store.products (sku);
CREATE INDEX idx_products_category_id   ON store.products (category_id);
CREATE INDEX idx_products_is_active     ON store.products (is_active);
CREATE INDEX idx_products_price         ON store.products (price);

-- orders
CREATE INDEX idx_orders_customer_id     ON store.orders (customer_id);
CREATE INDEX idx_orders_status          ON store.orders (status);
CREATE INDEX idx_orders_created_at      ON store.orders (created_at DESC);

-- order_items
CREATE INDEX idx_order_items_order_id   ON store.order_items (order_id);
CREATE INDEX idx_order_items_product_id ON store.order_items (product_id);

-- payments
CREATE INDEX idx_payments_order_id      ON store.payments (order_id);
CREATE INDEX idx_payments_status        ON store.payments (status);

-- -------------------------------------------------------------
-- Sample Data
-- -------------------------------------------------------------

-- Categories
INSERT INTO store.categories (name, slug, parent_id) VALUES
    ('Electronics',         'electronics',          NULL),
    ('Clothing',            'clothing',              NULL),
    ('Home & Kitchen',      'home-kitchen',          NULL),
    ('Laptops',             'laptops',               1),
    ('Smartphones',         'smartphones',           1),
    ('Men''s Clothing',     'mens-clothing',         2),
    ('Women''s Clothing',   'womens-clothing',       2),
    ('Kitchen Appliances',  'kitchen-appliances',    3);

-- Products
INSERT INTO store.products (category_id, sku, name, description, price, stock_qty) VALUES
    (4, 'LAP-001', 'ProBook 15 Laptop',       '15" laptop, 16GB RAM, 512GB SSD',      1299.99,  42),
    (4, 'LAP-002', 'UltraSlim 13 Laptop',     '13" ultrabook, 8GB RAM, 256GB SSD',     899.99,  18),
    (5, 'PHN-001', 'Galaxy X Pro',             '6.7" AMOLED, 256GB, 5G',               999.99,  75),
    (5, 'PHN-002', 'Pixel 8',                  '6.2" display, 128GB, Google AI',        699.99,  60),
    (6, 'CLM-001', 'Classic Oxford Shirt',     '100% cotton, slim fit',                  49.99, 200),
    (6, 'CLM-002', 'Chino Pants',              'Stretch fabric, tapered fit',            59.99, 150),
    (7, 'CLW-001', 'Floral Summer Dress',      'Lightweight rayon, midi length',         69.99, 130),
    (7, 'CLW-002', 'Denim Jacket',             'Classic fit, raw denim',                 89.99,  90),
    (8, 'KIT-001', 'Espresso Machine',         '15-bar pressure, milk frother',         249.99,  35),
    (8, 'KIT-002', 'Air Fryer 5.8QT',          'Digital display, 8 presets',             89.99,  55);

-- Customers
INSERT INTO store.customers (email, first_name, last_name, phone) VALUES
    ('alice@example.com',   'Alice',   'Johnson',  '702-555-0101'),
    ('bob@example.com',     'Bob',     'Martinez', '702-555-0102'),
    ('carol@example.com',   'Carol',   'Williams', '702-555-0103'),
    ('dave@example.com',    'Dave',    'Brown',    '702-555-0104'),
    ('eve@example.com',     'Eve',     'Davis',    '702-555-0105');

-- Addresses
INSERT INTO store.addresses (customer_id, label, street, city, state, zip, is_default) VALUES
    (1, 'home',  '123 Main St',       'Las Vegas',   'NV', '89101', TRUE),
    (2, 'home',  '456 Desert Rd',     'Henderson',   'NV', '89002', TRUE),
    (3, 'home',  '789 Sunset Blvd',   'Las Vegas',   'NV', '89109', TRUE),
    (4, 'work',  '321 Business Ave',  'North Las Vegas', 'NV', '89030', TRUE),
    (5, 'home',  '654 Spring Valley', 'Las Vegas',   'NV', '89147', TRUE);

-- Orders
INSERT INTO store.orders (customer_id, shipping_addr_id, status, subtotal, shipping_cost, tax) VALUES
    (1, 1, 'delivered',  1299.99,  9.99, 117.00),
    (1, 1, 'shipped',      49.99,  4.99,   4.50),
    (2, 2, 'confirmed',   999.99,  0.00,  90.00),
    (3, 3, 'pending',     339.98,  9.99,  30.60),
    (4, 4, 'delivered',   249.99,  9.99,  22.50),
    (5, 5, 'cancelled',    69.99,  4.99,   6.30);

-- Order Items
INSERT INTO store.order_items (order_id, product_id, quantity, unit_price) VALUES
    (1, 1, 1, 1299.99),   -- Alice: laptop
    (2, 5, 2,   49.99),   -- Alice: 2x oxford shirts
    (3, 3, 1,  999.99),   -- Bob: smartphone
    (4, 9, 1,  249.99),   -- Carol: espresso machine
    (4, 10,1,   89.99),   -- Carol: air fryer
    (5, 9, 1,  249.99),   -- Dave: espresso machine
    (6, 7, 1,   69.99);   -- Eve: dress (cancelled)

-- Payments
INSERT INTO store.payments (order_id, method, status, amount, transaction_ref, paid_at) VALUES
    (1, 'credit_card',  'completed', 1426.98, 'TXN-A1B2C3', NOW() - INTERVAL '10 days'),
    (2, 'credit_card',  'completed',   59.48, 'TXN-D4E5F6', NOW() - INTERVAL '3 days'),
    (3, 'paypal',       'completed',  1089.99, 'TXN-G7H8I9', NOW() - INTERVAL '1 day'),
    (4, 'debit_card',   'pending',     380.57, NULL,          NULL),
    (5, 'credit_card',  'completed',   282.48, 'TXN-J1K2L3', NOW() - INTERVAL '15 days'),
    (6, 'paypal',       'refunded',     81.28, 'TXN-M4N5O6', NOW() - INTERVAL '5 days');

-- -------------------------------------------------------------
-- Sanity check
-- -------------------------------------------------------------
SELECT
    t.schemaname,
    t.tablename,
    c.reltuples::BIGINT AS approx_rows
FROM pg_tables t
JOIN pg_class c ON c.relname = t.tablename
WHERE t.schemaname = 'store'
ORDER BY t.tablename;
