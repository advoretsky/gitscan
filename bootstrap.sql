-- create user gitscanner;

-- Create user with password
CREATE USER gitscan WITH PASSWORD '123456';

-- Grant privileges on database
GRANT ALL PRIVILEGES ON DATABASE gitscan TO gitscan;

GRANT ALL PRIVILEGES ON SCHEMA public TO gitscan;

-- -- Grant all privileges on all tables in the schema
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA gitscan TO gitscan;

CREATE TABLE commits (
     owner VARCHAR NOT NULL,
     repo VARCHAR NOT NULL,
     commit VARCHAR NOT NULL,
     info VARCHAR,
     CONSTRAINT pk_owner_repo_commit PRIMARY KEY (owner, repo, commit)
);