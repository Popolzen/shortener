CREATE TABLE IF NOT EXISTS shortened_urls (
    id BIGSERIAL PRIMARY KEY,
    original_url TEXT NOT NULL,
    short_code VARCHAR(20) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT chk_original_url_length CHECK (length(original_url) <= 2048),
    CONSTRAINT chk_short_code_length CHECK (length(short_code) >= 4)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_shortened_urls_short_code ON shortened_urls(short_code);
CREATE INDEX IF NOT EXISTS idx_shortened_urls_created_at ON shortened_urls(created_at);
