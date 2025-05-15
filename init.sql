CREATE TABLE medicine (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    price FLOAT NOT NULL
);

CREATE TYPE usage_type_enum AS ENUM ('one_time', 'multi_use', 'time_based');
CREATE TYPE discount_type_enum AS ENUM ('flat', 'percentage', 'free_delivery');
CREATE TYPE discount_target_enum AS ENUM ('inventory', 'charges', 'inventory_and_charges');

CREATE TABLE coupon (
    coupon_code VARCHAR(100) PRIMARY KEY,
    expiry_date TIMESTAMP NOT NULL,
    usage_type usage_type_enum NOT NULL,
    min_order_value FLOAT,
    valid_from TIMESTAMP,
    valid_until TIMESTAMP,
    discount_type discount_type_enum NOT NULL,
    discount_value FLOAT NOT NULL,
    discount_target discount_target_enum NOT NULL,
    terms_and_conditions VARCHAR(1000),
    max_usage_per_user INT
);

CREATE TABLE coupon_medicine_map (
    coupon_code VARCHAR(100) REFERENCES coupon(coupon_code),
    medicine_id UUID REFERENCES medicine(id),
    PRIMARY KEY (coupon_code, medicine_id)
);

CREATE TABLE coupon_category_map (
    coupon_code VARCHAR(100) REFERENCES coupon(coupon_code),
    category_name VARCHAR(100),
    PRIMARY KEY (coupon_code, category_name)
);
