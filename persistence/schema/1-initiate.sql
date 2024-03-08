CREATE ROLE mergedmining WITH LOGIN ENCRYPTED PASSWORD 'asdfasdf';
CREATE DATABASE mergedmining OWNER mergedmining;
GRANT ALL PRIVILEGES ON DATABASE mergedmining TO mergedmining;
-- Switch databases
-- \c mergedmining postgres
GRANT ALL ON SCHEMA public TO mergedmining;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO mergedmining;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO mergedmining;