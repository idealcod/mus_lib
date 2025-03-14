CREATE TABLE IF NOT EXISTS songs (
                                     id SERIAL PRIMARY KEY,
                                     group_name VARCHAR(255) NOT NULL,
                                     song_name VARCHAR(255) NOT NULL,
                                     release_date VARCHAR(10),
                                     text TEXT,
                                     link VARCHAR(255),
                                     created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                     updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);