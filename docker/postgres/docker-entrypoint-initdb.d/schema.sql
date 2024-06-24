-- Create the orders table
CREATE TABLE IF NOT EXISTS orders
(
    id VARCHAR(15) PRIMARY KEY DEFAULT substring(replace(uuid_generate_v4()::text, '-', '') FROM 3 FOR 15),
    is_done BOOLEAN NOT NULL
);

-- Create the items table
CREATE TABLE IF NOT EXISTS items
(
    id   SERIAL PRIMARY KEY,
    name TEXT
);

-- Create the order_items table
CREATE TABLE IF NOT EXISTS order_items
(
    id       SERIAL PRIMARY KEY,
    order_id VARCHAR(15) NOT NULL,
    item_id  INTEGER     NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders (id),
    FOREIGN KEY (item_id) REFERENCES items (id)
);
