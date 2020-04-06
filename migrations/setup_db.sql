SELECT 'CREATE DATABASE btc_listings'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'btc_listings')
