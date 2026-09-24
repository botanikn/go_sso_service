-- Secrets shorter than 32 bytes are rejected by the service (and the seeded one
-- was committed to git). Replace them with 64 random hex chars generated in the DB.
-- gen_random_uuid() is built in since PostgreSQL 13 and uses a CSPRNG.
UPDATE apps
SET secret = replace(gen_random_uuid()::text || gen_random_uuid()::text, '-', '')
WHERE length(secret) < 32;
