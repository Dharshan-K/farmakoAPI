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

CREATE TABLE coupon_usage (
  user_id UUID NOT NULL,
  coupon_code TEXT NOT NULL,
  usage INT NOT NULL DEFAULT 1,
  PRIMARY KEY (user_id, coupon_code),
  FOREIGN KEY (coupon_code) REFERENCES coupon(coupon_code)
);

INSERT INTO medicine (id, name, category, price) VALUES
('3fa85f64-5717-4562-b3fc-2c963f66afa6', 'Paracetamol 500mg',  'Pain Relief',  25),
('7b9d71f6-0b8e-4e12-b6b3-1cd5cf9243a0', 'Amoxicillin 250mg',  'Antibiotics',  50),
('c9d3e5a4-135a-4bb8-90ab-52e123a21abc', 'Cetirizine 10mg',    'Allergy',      15),
('6f1f4c62-c420-49a6-8854-5d76f8d99770', 'Metformin 500mg',    'Diabetes',     35),
('1a2c3d4e-5f6a-7b8c-9d0e-1f2a3b4c5d6e', 'Atorvastatin 10mg',  'Cholesterol',  45),
('9e9b2e5c-d215-4e3d-9f6b-7b5e2e60f92b', 'Ibuprofen 200mg',    'Pain Relief',  20),
('8e4cf3a9-d032-4709-a9c2-e3c4f9085e15', 'Azithromycin 500mg', 'Antibiotics',  65),
('2d471a6f-d6ff-43b1-8a62-121eb8500db6', 'Loratadine 10mg',    'Allergy',      18),
('ac9bf4cd-3490-4aa1-a94e-71cc36d0a215', 'Glibenclamide 5mg',  'Diabetes',     30),
('52ab1b2f-fce4-4f1f-8fa9-c2f5b3c3b3d5', 'Simvastatin 20mg',   'Cholesterol',  40);


INSERT INTO coupon (
    coupon_code, expiry_date, usage_type, min_order_value, valid_from, valid_until,
    discount_type, discount_value, terms_and_conditions, max_usage_per_user, discount_target
) VALUES
('SUMMER2024',        '2024-12-31 23:59:59', 'multi_use',  100.5, '2024-06-01 00:00:00', '2024-12-31 23:59:59', 'flat',       15, 'Valid on orders above $100',                    3, 'inventory'),
('SAVE10',            '2025-12-31 23:59:59', 'one_time',   100,   '2025-05-15 00:00:00', '2025-12-31 23:59:59', 'percentage', 10, 'Valid on all Allergy medicines above ₹100.',   1, 'inventory'),
('TOO_MUCH_DISCOUNT', '2025-12-31 23:59:59', 'one_time',   100,   '2025-05-15 00:00:00', '2025-12-31 23:59:59', 'percentage',120, 'Should fail for >100% discount',               1, 'inventory'),
('FLAT20PAIN',        '2025-12-31 23:59:59', 'one_time',    50,   '2025-01-01 00:00:00', '2025-12-31 23:59:59', 'flat',       20, 'Flat discount on Pain Relief.',                1, 'inventory'),
('PERCENT10ALLERGY',  '2025-12-31 23:59:59', 'multi_use',   40,   '2025-01-01 00:00:00', '2025-12-31 23:59:59', 'percentage', 10, '10% off on allergy meds.',                     3, 'inventory'),
('NEWYEAR2025',       '2025-01-10 23:59:59', 'time_based', 100,   '2025-01-01 00:00:00', '2025-01-10 23:59:59', 'flat',       25, 'New Year offer for select categories.',        2, 'inventory'),
('DIAB10',            '2026-01-01 00:00:00', 'multi_use',   60,   '2025-05-01 00:00:00', '2025-12-31 23:59:59', 'percentage', 10, '10% off on diabetes meds.',                    4, 'inventory'),
('SAVE5ALL',          '2025-12-31 23:59:59', 'multi_use',   30,   '2025-01-01 00:00:00', '2025-12-31 23:59:59', 'percentage',  5, '5% off site-wide.',                             10, 'inventory'),
('CHOL30OFF',         '2025-12-01 23:59:59', 'multi_use',   80,   '2025-06-01 00:00:00', '2025-12-01 23:59:59', 'flat',       30, 'Big savings on cholesterol care.',             2, 'inventory'),
('AMOX25',            '2025-08-01 23:59:59', 'one_time',    45,   '2025-06-01 00:00:00', '2025-08-01 23:59:59', 'flat',       25, 'One-time discount on Amoxicillin.',            1, 'inventory'),
('ANTIBIO10',         '2025-11-30 23:59:59', 'multi_use',   70,   '2025-04-01 00:00:00', '2025-11-30 23:59:59', 'percentage', 10, 'Save on antibiotics.',                         3, 'inventory'),
('LORA15',            '2025-09-01 23:59:59', 'time_based',  25,   '2025-05-01 00:00:00', '2025-09-01 23:59:59', 'flat',       15, 'Loratadine flat discount.',                    1, 'inventory'),
('WELCOME50',         '2026-01-01 00:00:00', 'one_time',   150,   '2025-01-01 00:00:00', '2025-12-31 23:59:59', 'percentage', 20, 'First-time buyer welcome discount.',           1, 'inventory');
