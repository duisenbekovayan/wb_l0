-- создаём отдельного пользователя для приложения
CREATE ROLE wb_app WITH LOGIN PASSWORD 'wb_app_pass';

-- если БД ещё нет — создадим и сразу назначим владельца
DO $$
BEGIN
   IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'wb_orders') THEN
      PERFORM dblink_exec('dbname=postgres', 'CREATE DATABASE wb_orders');
   END IF;
END$$;

-- права и владение
ALTER DATABASE wb_orders OWNER TO wb_app;

\connect wb_orders

-- владелец схемы и права
ALTER SCHEMA public OWNER TO wb_app;
GRANT USAGE, CREATE ON SCHEMA public TO wb_app;

-- на будущие таблицы/последовательности
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO wb_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO wb_app;
