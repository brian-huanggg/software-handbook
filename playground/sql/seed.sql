-- SQL Bootcamp seed data (PostgreSQL 16)
-- 資料是確定性的（不使用 random()），每次載入結果相同，方便對照預期筆數。
-- 先執行 schema.sql，再執行本檔。

BEGIN;

-- ---------------------------------------------------------------
-- employees (10)：樹狀結構 CEO -> VP -> 經理 -> 業務
-- ---------------------------------------------------------------
INSERT INTO employees (id, name, title, department, manager_id, salary, hired_at) VALUES
(1,  'Alice Chen',  'CEO',             'Executive',   NULL, 250000, '2018-03-01'),
(2,  'Bob Lin',     'VP Sales',        'Sales',       1,    180000, '2018-09-15'),
(3,  'Carol Wu',    'VP Engineering',  'Engineering', 1,    190000, '2019-01-10'),
(4,  'Dave Huang',  'Sales Manager',   'Sales',       2,    120000, '2019-06-01'),
(5,  'Eva Chang',   'Sales Rep',       'Sales',       4,     70000, '2020-02-17'),
(6,  'Frank Lee',   'Sales Rep',       'Sales',       4,     68000, '2021-05-03'),
(7,  'Grace Tsai',  'Sales Rep',       'Sales',       4,     72000, '2021-11-22'),
(8,  'Henry Wang',  'Sales Rep',       'Sales',       2,     65000, '2022-08-08'),
(9,  'Ivy Liu',     'Sales Rep',       'Sales',       2,     66000, '2023-03-20'),
(10, 'Jack Yang',   'Engineer',        'Engineering', 3,    110000, '2022-01-05');

-- ---------------------------------------------------------------
-- customers (20)：含 NULL phone、NULL birth_date、推薦關係
-- 顧客 19、20 不會有任何訂單
-- ---------------------------------------------------------------
INSERT INTO customers (id, name, email, phone, city, country, birth_date, referred_by, created_at) VALUES
(1,  '王小明',        'ming.wang@example.com',   '0912-345-678', 'Taipei',    'Taiwan', '1990-04-12', NULL, '2023-01-05 10:00+08'),
(2,  '李美玲',        'meiling.li@example.com',  '0922-111-222', 'Taipei',    'Taiwan', '1988-11-30', 1,    '2023-01-20 14:30+08'),
(3,  '陳大文',        'david.chen@example.com',  NULL,           'Taichung',  'Taiwan', '1995-07-08', 1,    '2023-02-11 09:15+08'),
(4,  '林雅婷',        'yating.lin@example.com',  '0933-222-333', 'Kaohsiung', 'Taiwan', '1992-02-25', 2,    '2023-02-28 16:45+08'),
(5,  '張志豪',        'chihao.chang@example.com','0955-444-555', 'Taipei',    'Taiwan', NULL,         NULL, '2023-03-14 11:20+08'),
(6,  'Alice Johnson', 'alice.j@example.com',     '212-555-0101', 'New York',  'USA',    '1985-06-15', NULL, '2023-03-30 08:00-05'),
(7,  'Bob Smith',     'bob.smith@example.com',   '415-555-0102', 'San Francisco', 'USA', '1979-09-03', 6,   '2023-04-18 12:00-07'),
(8,  'Carol White',   'carol.w@example.com',     NULL,           'Seattle',   'USA',    '1991-12-21', 6,    '2023-05-02 15:30-07'),
(9,  'Daniel Brown',  'daniel.b@example.com',    '312-555-0104', 'Chicago',   'USA',    NULL,         NULL, '2023-05-25 09:45-05'),
(10, 'Emma Davis',    'emma.d@example.com',      '617-555-0105', 'Boston',    'USA',    '1998-03-09', 7,    '2023-06-10 13:10-04'),
(11, '田中太郎',      'taro.tanaka@example.com', '090-1234-5678','Tokyo',     'Japan',  '1987-01-19', NULL, '2023-06-29 10:30+09'),
(12, '佐藤花子',      'hanako.sato@example.com', '080-2345-6789','Osaka',     'Japan',  '1993-08-27', 11,   '2023-07-15 17:00+09'),
(13, '鈴木一郎',      'ichiro.suzuki@example.com',NULL,          'Tokyo',     'Japan',  '1980-05-05', 11,   '2023-08-01 08:40+09'),
(14, '高橋美咲',      'misaki.t@example.com',    '070-3456-7890','Kyoto',     'Japan',  '1996-10-14', 12,   '2023-08-19 11:55+09'),
(15, 'Oliver Wilson', 'oliver.w@example.com',    '020-7946-0111','London',    'UK',     '1989-02-02', NULL, '2023-09-07 09:00+01'),
(16, 'Sophia Taylor', 'sophia.t@example.com',    '020-7946-0112','Manchester','UK',     '1994-07-17', 15,   '2023-10-03 14:20+01'),
(17, 'Lucas Martin',  'lucas.m@example.com',     NULL,           'Paris',     'France', '1983-11-11', NULL, '2023-10-26 16:10+02'),
(18, 'Chloe Bernard', 'chloe.b@example.com',     '01-23-45-67-89','Lyon',     'France', '1997-04-04', 17,   '2023-11-14 10:05+01'),
(19, '黃建國',        'jianguo.h@example.com',   '0966-777-888', 'Tainan',    'Taiwan', '1975-12-01', NULL, '2023-12-01 09:00+08'),
(20, 'Mia Anderson',  'mia.a@example.com',       NULL,           'Sydney',    'Australia','2000-01-23', NULL, '2023-12-20 18:30+11');

-- ---------------------------------------------------------------
-- categories (12)：兩層到三層的分類樹
-- ---------------------------------------------------------------
INSERT INTO categories (id, name, parent_id) VALUES
(1,  'Electronics',  NULL),
(2,  'Books',        NULL),
(3,  'Home',         NULL),
(4,  'Clothing',     NULL),
(5,  'Phones',       1),
(6,  'Laptops',      1),
(7,  'Accessories',  1),
(8,  'Fiction',      2),
(9,  'Technical',    2),
(10, 'Kitchen',      3),
(11, 'Cables',       7),
(12, 'Chargers',     7);

-- ---------------------------------------------------------------
-- products (25)：商品 24（Winter Jacket）從未售出；商品 25 已停產
-- ---------------------------------------------------------------
INSERT INTO products (id, name, category_id, price, stock, discontinued, created_at) VALUES
(1,  'Phone X1',            5,  699.00, 120, false, '2023-01-01 00:00+00'),
(2,  'Phone X1 Pro',        5,  999.00,  80, false, '2023-01-01 00:00+00'),
(3,  'Phone Lite',          5,  299.00, 300, false, '2023-01-01 00:00+00'),
(4,  'Laptop Air 13',       6, 1099.00,  45, false, '2023-01-01 00:00+00'),
(5,  'Laptop Pro 16',       6, 1999.00,  20, false, '2023-01-01 00:00+00'),
(6,  'Laptop Basic 14',     6,  549.00,  60, false, '2023-01-01 00:00+00'),
(7,  'USB-C Cable 1m',     11,    9.99, 900, false, '2023-01-01 00:00+00'),
(8,  'USB-C Cable 2m',     11,   14.99, 700, false, '2023-01-01 00:00+00'),
(9,  'Lightning Cable',    11,   12.99, 500, false, '2023-01-01 00:00+00'),
(10, 'Fast Charger 30W',   12,   24.99, 400, false, '2023-01-01 00:00+00'),
(11, 'Wireless Charger',   12,   34.99, 250, false, '2023-01-01 00:00+00'),
(12, 'Bluetooth Mouse',     7,   19.99, 350, false, '2023-01-01 00:00+00'),
(13, 'Mechanical Keyboard', 7,   89.00, 150, false, '2023-01-01 00:00+00'),
(14, 'Laptop Sleeve',       7,   29.00, 200, false, '2023-01-01 00:00+00'),
(15, 'SQL Basics',          9,   35.00, 100, false, '2023-01-01 00:00+00'),
(16, 'Database Internals',  9,   59.00,  70, false, '2023-01-01 00:00+00'),
(17, 'Designing Data Apps', 9,   62.00,  65, false, '2023-01-01 00:00+00'),
(18, 'Mystery Night',       8,   15.50,  90, false, '2023-01-01 00:00+00'),
(19, 'Space Odyssey',       8,   18.00,  85, false, '2023-01-01 00:00+00'),
(20, 'Chef Knife',         10,   45.00, 110, false, '2023-01-01 00:00+00'),
(21, 'Non-stick Pan',      10,   32.00, 130, false, '2023-01-01 00:00+00'),
(22, 'Blender',            10,   79.00,  55, false, '2023-01-01 00:00+00'),
(23, 'T-Shirt',             4,   19.00, 500, false, '2023-01-01 00:00+00'),
(24, 'Winter Jacket',       4,  129.00,  40, false, '2023-01-01 00:00+00'),
(25, 'Old Phone Z0',        5,  199.00,   0, true,  '2023-01-01 00:00+00');

-- ---------------------------------------------------------------
-- orders (240)：每 70 小時一張，涵蓋 2024-01 到 2025-12
--   status：g%10=0 cancelled、=1 pending、=2 shipped、其餘 completed
--   employee_id：g%13=0 為線上訂單（NULL），其餘由業務 5–9 輪流負責
--   customer_id：只落在顧客 1–18（顧客 19、20 無訂單）
-- ---------------------------------------------------------------
INSERT INTO orders (id, customer_id, employee_id, status, ordered_at, shipped_at)
SELECT
    g,
    1 + (g * 7) % 18,
    CASE WHEN g % 13 = 0 THEN NULL ELSE 5 + g % 5 END,
    CASE g % 10 WHEN 0 THEN 'cancelled' WHEN 1 THEN 'pending' WHEN 2 THEN 'shipped' ELSE 'completed' END,
    timestamptz '2024-01-01 08:00+00' + g * interval '70 hours',
    CASE WHEN g % 10 IN (0, 1) THEN NULL
         ELSE timestamptz '2024-01-01 08:00+00' + g * interval '70 hours' + interval '2 days' END
FROM generate_series(1, 240) AS g;

-- ---------------------------------------------------------------
-- order_items：每張訂單 1–4 項商品，商品 24、25 不會被賣出
--   unit_price 取自 products.price；部分訂單套用 10% 折扣（g%7=0）
-- ---------------------------------------------------------------
INSERT INTO order_items (order_id, product_id, quantity, unit_price)
SELECT
    g,
    p.id,
    1 + (g + k) % 3,
    CASE WHEN g % 7 = 0 THEN round(p.price * 0.9, 2) ELSE p.price END
FROM generate_series(1, 240) AS g
CROSS JOIN LATERAL generate_series(1, 1 + g % 4) AS k
JOIN products p ON p.id = 1 + (g * 5 + k * 11) % 23;

-- ---------------------------------------------------------------
-- payments：已出貨與已完成的訂單才有付款
--   g%20=5 只付一半（部分付款）、g%17=0 分兩次付清
-- ---------------------------------------------------------------
WITH totals AS (
    SELECT o.id, o.ordered_at, o.status,
           SUM(oi.quantity * oi.unit_price) AS total
    FROM orders o
    JOIN order_items oi ON oi.order_id = o.id
    WHERE o.status IN ('shipped', 'completed')
    GROUP BY o.id, o.ordered_at, o.status
)
INSERT INTO payments (order_id, amount, method, paid_at)
SELECT id,
       CASE WHEN id % 20 = 5  THEN round(total / 2, 2)
            WHEN id % 17 = 0  THEN round(total / 2, 2)
            ELSE total END,
       (ARRAY['credit_card', 'bank_transfer', 'cash_on_delivery'])[1 + id % 3],
       ordered_at + interval '1 hour'
FROM totals
UNION ALL
SELECT id,
       total - round(total / 2, 2),
       (ARRAY['credit_card', 'bank_transfer', 'cash_on_delivery'])[1 + id % 3],
       ordered_at + interval '3 days'
FROM totals
WHERE id % 17 = 0 AND id % 20 <> 5;

-- ---------------------------------------------------------------
-- 重設 identity 序列，之後 INSERT 不帶 id 時才不會衝突
-- ---------------------------------------------------------------
SELECT setval(pg_get_serial_sequence('employees',   'id'), (SELECT max(id) FROM employees));
SELECT setval(pg_get_serial_sequence('customers',   'id'), (SELECT max(id) FROM customers));
SELECT setval(pg_get_serial_sequence('categories',  'id'), (SELECT max(id) FROM categories));
SELECT setval(pg_get_serial_sequence('products',    'id'), (SELECT max(id) FROM products));
SELECT setval(pg_get_serial_sequence('orders',      'id'), (SELECT max(id) FROM orders));
SELECT setval(pg_get_serial_sequence('order_items', 'id'), (SELECT max(id) FROM order_items));
SELECT setval(pg_get_serial_sequence('payments',    'id'), (SELECT max(id) FROM payments));

COMMIT;

ANALYZE;
