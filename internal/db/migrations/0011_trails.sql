-- Server-owned catalog of hiking trails / peaks fetched from the keyless
-- Overpass (OpenStreetMap) API, mirroring places_catalog: it grows into our own
-- content asset and a fallback when the external API is slow or unavailable.

CREATE TABLE trails (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    lat        DOUBLE PRECISION NOT NULL DEFAULT 0,
    lng        DOUBLE PRECISION NOT NULL DEFAULT 0,
    source     TEXT NOT NULL DEFAULT '',
    first_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen  TIMESTAMPTZ NOT NULL DEFAULT now()
);
