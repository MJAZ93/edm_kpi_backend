-- Ensure aggregation_type column exists on tasks table
-- Run this on production if the column is missing

ALTER TABLE tasks ADD COLUMN IF NOT EXISTS aggregation_type VARCHAR(20) DEFAULT 'SUM_UP';

-- Backfill any NULL values
UPDATE tasks SET aggregation_type = 'SUM_UP' WHERE aggregation_type IS NULL OR aggregation_type = '';
