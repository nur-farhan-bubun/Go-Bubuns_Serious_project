-- 1. Create Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 2. Create Enums
CREATE TYPE user_role_enum AS ENUM (
    'JOB_SEEKER',
    'DATING_ONLY',
    'SOCIAL_ONLY',
    'POWER_USER'
);

-- 3. Identity & Auth Table
-- Stored as VARCHAR(36) to support external auth provider IDs (e.g., Clerk, Firebase) or UUIDs
CREATE TABLE users (
    id            VARCHAR(36) PRIMARY KEY,
    email         VARCHAR(255) UNIQUE NOT NULL,
    phone         VARCHAR(20),
    password_hash TEXT,                  -- NULL for OAuth-only users, set for password-auth users
    role          user_role_enum NOT NULL DEFAULT 'SOCIAL_ONLY',
    is_onboarded  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 4. Core Universal Profiles Table (1:1 with users)
CREATE TABLE profiles (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      VARCHAR(36) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    avatar_url   TEXT,
    bio          TEXT,
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_profiles_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 5. Dating Specific Profiles Table (1:1 with users)
CREATE TABLE dating_profiles (
    user_id          VARCHAR(36) PRIMARY KEY,
    gender           VARCHAR(20) NOT NULL,
    interested_in    VARCHAR(20) NOT NULL,
    birth_date       DATE NOT NULL,
    height_cm        INT,
    relationship_goal VARCHAR(50),
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_dating_profiles_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 6. Professional / Worker Specific Profiles Table (1:1 with users)
CREATE TABLE worker_profiles (
    user_id             VARCHAR(36) PRIMARY KEY,
    skills              TEXT[],
    hourly_rate         NUMERIC(10, 2),
    is_available        BOOLEAN NOT NULL DEFAULT TRUE,
    completed_jobs_count INT NOT NULL DEFAULT 0,
    rating_avg          NUMERIC(3, 2) NOT NULL DEFAULT 0.00,
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_worker_profiles_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 7. Multi-Asset Gallery Photo Table (1:N with users)
CREATE TABLE profile_photos (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    VARCHAR(36) NOT NULL,
    s3_url     TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_profile_photos_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 8. Performance Indexes
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_profiles_user_id ON profiles(user_id);
CREATE INDEX idx_photos_user_id ON profile_photos(user_id);
