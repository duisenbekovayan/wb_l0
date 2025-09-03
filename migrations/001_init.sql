CREATE TABLE IF NOT EXISTS deliveries (
  id SERIAL PRIMARY KEY,
  name TEXT, phone TEXT, zip TEXT, city TEXT,
  address TEXT, region TEXT, email TEXT
);

CREATE TABLE IF NOT EXISTS payments (
  id SERIAL PRIMARY KEY,
  transaction TEXT UNIQUE,
  request_id TEXT, currency TEXT, provider TEXT,
  amount INT, payment_dt BIGINT, bank TEXT,
  delivery_cost INT, goods_total INT, custom_fee INT
);

CREATE TABLE IF NOT EXISTS orders (
  order_uid TEXT PRIMARY KEY,
  track_number TEXT,
  entry TEXT,
  locale TEXT,
  internal_signature TEXT,
  customer_id TEXT,
  delivery_service TEXT,
  shardkey TEXT, sm_id INT,
  date_created TIMESTAMPTZ,
  oof_shard TEXT,
  delivery_id INT REFERENCES deliveries(id) ON DELETE CASCADE,
  payment_id INT REFERENCES payments(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS items (
  id SERIAL PRIMARY KEY,
  chrt_id BIGINT,
  track_number TEXT,
  price INT,
  rid TEXT,
  name TEXT,
  sale INT,
  size TEXT,
  total_price INT,
  nm_id BIGINT,
  brand TEXT,
  status INT,
  order_uid TEXT REFERENCES orders(order_uid) ON DELETE CASCADE
);

-- индексы
CREATE INDEX IF NOT EXISTS idx_items_order ON items(order_uid);
