-- RUN as postgres user

CREATE DATABASE gitscan;
CREATE USER gitscan WITH PASSWORD '123456';

-- connect to database gitscan

GRANT ALL PRIVILEGES ON DATABASE gitscan TO gitscan;
GRANT ALL PRIVILEGES ON SCHEMA public TO gitscan;

-- RUN as gitscan user

CREATE TABLE commits (
     owner VARCHAR NOT NULL,
     repo VARCHAR NOT NULL,
     commit VARCHAR NOT NULL,
     info VARCHAR,
     CONSTRAINT pk_owner_repo_commit PRIMARY KEY (owner, repo, commit)
);