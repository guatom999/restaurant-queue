-- ตาราง user ของระบบ (ลูกค้าที่จอง)
CREATE TABLE users
(
    id BIGSERIAL PRIMARY KEY,
    phone VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_phone ON users(phone);

-- ตารางร้านอาหารที่เปิดให้จองคิว
CREATE TABLE restaurants
(
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(20) UNIQUE NOT NULL,
    -- เช่น 'SUKI01', 'MK-CENTRAL'
    name VARCHAR(150) NOT NULL,
    -- ชื่อร้าน
    address TEXT,
    phone VARCHAR(20),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_restaurants_active ON restaurants(is_active) WHERE is_active = TRUE;

-- ตารางหลักของการจองคิว
CREATE TABLE reservations
(
    id BIGSERIAL PRIMARY KEY,
    reservation_code VARCHAR(30) UNIQUE NOT NULL,
    -- เช่น 'A001', 'B042'
    user_id BIGINT NOT NULL REFERENCES users(id),
    restaurant_id BIGINT NOT NULL REFERENCES restaurants(id),
    queue_number INT NOT NULL,
    -- เลขคิวของวันนั้น
    party_size INT NOT NULL DEFAULT 1,
    -- จำนวนคนที่มาทาน
    status VARCHAR(20) NOT NULL DEFAULT 'WAITING',
    reserved_for_date DATE NOT NULL,
    -- วันที่มาใช้บริการ
    reserved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    called_at TIMESTAMPTZ,
    -- เวลาที่ถูกเรียก
    completed_at TIMESTAMPTZ,
    -- เวลาที่เสร็จสิ้น (นั่งโต๊ะแล้ว)
    cancelled_at TIMESTAMPTZ,
    note TEXT,

    CONSTRAINT chk_status CHECK (status IN ('WAITING','CALLED','SEATED','COMPLETED','CANCELLED','NO_SHOW')),
    CONSTRAINT chk_party_size CHECK (party_size > 0)
);

-- Index สำหรับ query ที่ใช้บ่อย
CREATE INDEX idx_reservations_user ON reservations(user_id);
CREATE INDEX idx_reservations_restaurant_date ON reservations(restaurant_id, reserved_for_date);
CREATE INDEX idx_reservations_status ON reservations(status) WHERE status IN ('WAITING','CALLED','SEATED');
CREATE UNIQUE INDEX idx_reservations_queue ON reservations(restaurant_id, reserved_for_date, queue_number);

-- เวลาทำการของร้านในแต่ละวัน + การแบ่ง slot
CREATE TABLE restaurant_business_hours
(
    id BIGSERIAL PRIMARY KEY,
    restaurant_id BIGINT NOT NULL REFERENCES restaurants(id),
    day_of_week SMALLINT NOT NULL,
    -- 0 = อาทิตย์, 1 = จันทร์, ..., 6 = เสาร์
    open_time TIME NOT NULL,
    close_time TIME NOT NULL,
    slot_duration_minutes INT NOT NULL DEFAULT 60,
    max_capacity_per_slot INT NOT NULL DEFAULT 10,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_day_of_week       CHECK (day_of_week BETWEEN 0 AND 6),
    CONSTRAINT chk_open_before_close CHECK (open_time < close_time),
    CONSTRAINT chk_slot_duration     CHECK (slot_duration_minutes > 0),
    CONSTRAINT chk_max_capacity      CHECK (max_capacity_per_slot > 0),
    CONSTRAINT uq_restaurant_day     UNIQUE (restaurant_id, day_of_week)
);

CREATE INDEX idx_biz_hours_restaurant ON restaurant_business_hours(restaurant_id);
CREATE INDEX idx_biz_hours_active     ON restaurant_business_hours(restaurant_id, day_of_week, is_active);

-- Outbox pattern: สำหรับส่ง event ไป Kafka อย่างปลอดภัย
CREATE TABLE event_outbox
(
    id BIGSERIAL PRIMARY KEY,
    aggregate_type VARCHAR(50) NOT NULL,
    -- เช่น 'reservation'
    aggregate_id BIGINT NOT NULL,
    -- id ของ reservation
    event_type VARCHAR(50) NOT NULL,
    -- เช่น 'reservation.created'
    payload JSONB NOT NULL,
    -- ข้อมูลที่จะส่งไป Kafka
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
    -- NULL = ยังไม่ได้ publish
);

CREATE INDEX idx_outbox_unpublished ON event_outbox(created_at) WHERE published_at IS NULL;