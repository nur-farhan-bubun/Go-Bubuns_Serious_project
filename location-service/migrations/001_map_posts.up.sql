-- Location Service: PostGIS spatial data migration
-- Creates the map_posts table for permanent pins and spot post threads.

-- 1. Enable PostGIS extension (run once per database)
CREATE EXTENSION IF NOT EXISTS postgis;

-- 2. Spot and post threads table
CREATE TABLE map_posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(36) NOT NULL,          -- Who posted (Logical Link to User Service)
    title VARCHAR(255) NOT NULL,           -- Post title
    content TEXT,                           -- Detailed description
    category VARCHAR(50) NOT NULL,         -- 'RESTAURANT', 'SOCIAL_LIFE', 'EVENT'
    image_urls TEXT[],                      -- Array of S3 image URLs

    -- GEOMETRY(Point, 4326): precise lat/lng spot on the map
    geom GEOMETRY(Point, 4326) NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- GiST index for fast spatial bounding-box queries
CREATE INDEX idx_map_posts_geom ON map_posts USING GIST(geom);
