ALTER TABLE products DROP CONSTRAINT products_category_slug_fkey;

ALTER TABLE products
    ADD CONSTRAINT products_category_slug_fkey
        FOREIGN KEY (category_slug)
            REFERENCES categories (slug);