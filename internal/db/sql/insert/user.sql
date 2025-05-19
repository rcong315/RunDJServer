INSERT INTO "user" (
        user_id,
        email,
        display_name,
        country,
        followers,
        product,
        image_urls
    )
VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (user_id) DO
UPDATE
SET email = EXCLUDED.email,
    display_name = EXCLUDED.display_name,
    country = EXCLUDED.country,
    followers = EXCLUDED.followers,
    product = EXCLUDED.product,
    image_urls = EXCLUDED.image_urls,
    updated_at = NOW();